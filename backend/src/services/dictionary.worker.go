package services

import (
	"bufio"
	"compress/gzip"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"termorize/src/enums"
	"termorize/src/integrations/kaikki"
	"termorize/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const dictionaryWorkerLock int64 = 814760923
const dictionaryBatchSize = 500
const dictionaryMaxLineBytes = 8 << 20
const dictionaryMaxDownloadBytes int64 = 10 << 30

// A dedicated SQL connection holds the worker lock and performs every job write.
// Losing that connection prevents an old worker from writing after another worker takes over.
type DictionaryWorker struct {
	DB      *gorm.DB
	Client  *http.Client
	TempDir string
}

func (w *DictionaryWorker) Run(ctx context.Context) error {
	return w.DB.Connection(func(conn *gorm.DB) error {
		var locked bool
		if err := conn.WithContext(ctx).Raw("SELECT pg_try_advisory_lock(?)", dictionaryWorkerLock).Scan(&locked).Error; err != nil {
			return err
		}
		if !locked {
			return nil
		}
		defer func() {
			unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := conn.WithContext(unlockCtx).Exec("SELECT pg_advisory_unlock(?)", dictionaryWorkerLock).Error; err != nil {
				if sqlConn, ok := conn.Statement.ConnPool.(*sql.Conn); ok {
					_ = sqlConn.Raw(func(any) error { return driver.ErrBadConn })
				}
			}
		}()
		conn = conn.WithContext(ctx)
		if err := w.cleanTemporaryFiles(); err != nil {
			return err
		}
		if err := conn.Model(&models.DictionaryImportJob{}).
			Where("status IN ?", []string{"downloading", "importing"}).
			Updates(map[string]any{"status": "interrupted", "error": "Worker interrupted. Retry the import; committed batches are preserved.", "finished_at": time.Now().UTC(), "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		var job models.DictionaryImportJob
		if err := conn.Where("status = ?", "queued").Order("created_at, id").First(&job).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		now := time.Now().UTC()
		job.Status, job.StartedAt = "downloading", &now
		if err := conn.Save(&job).Error; err != nil {
			return err
		}
		runErr := w.importJob(ctx, conn, &job)
		// Reload only committed progress; a rolled-back batch must not appear in the outcome.
		finishCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn = conn.WithContext(finishCtx)
		if err := conn.First(&job, "id = ?", job.ID).Error; err != nil {
			return err
		}
		now = time.Now().UTC()
		job.FinishedAt, job.Status = &now, "succeeded"
		if runErr != nil {
			job.Status, job.Error = "failed", runErr.Error()
			if ctx.Err() != nil {
				job.Status = "interrupted"
			}
		}
		return conn.Save(&job).Error
	})
}

func (w *DictionaryWorker) cleanTemporaryFiles() error {
	if err := os.MkdirAll(w.TempDir, 0700); err != nil {
		return fmt.Errorf("create import temporary directory: %w", err)
	}
	entries, err := os.ReadDir(w.TempDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".jsonl.gz") {
			continue
		}
		if _, err := uuid.Parse(strings.TrimSuffix(name, ".jsonl.gz")); err != nil {
			continue
		}
		if err := os.Remove(filepath.Join(w.TempDir, name)); err != nil {
			return fmt.Errorf("clean abandoned import: %w", err)
		}
	}
	return nil
}

func (w *DictionaryWorker) importJob(ctx context.Context, conn *gorm.DB, job *models.DictionaryImportJob) (resultErr error) {
	if !kaikki.SupportedEdition(job.Edition) {
		return ErrDictionaryEdition
	}
	path := filepath.Join(w.TempDir, job.ID.String()+".jsonl.gz")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("create temporary download: %w", err)
	}
	defer func() {
		closeErr := file.Close()
		removeErr := os.Remove(path)
		resultErr = errors.Join(resultErr, closeErr, removeErr)
	}()
	if err := w.download(ctx, conn, job, file); err != nil {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	reader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer reader.Close()
	job.Status = "importing"
	if err := conn.Save(job).Error; err != nil {
		return err
	}
	return w.extract(ctx, conn, job, reader)
}

func (w *DictionaryWorker) download(ctx context.Context, conn *gorm.DB, job *models.DictionaryImportJob, file *os.File) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, job.DownloadURL, nil)
	if err != nil {
		return errors.New("invalid source download URL")
	}
	request.Header.Set("User-Agent", "Termorize dictionary importer")
	request.Header.Set("Accept-Encoding", "identity")
	response, err := w.Client.Do(request)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > dictionaryMaxDownloadBytes {
		return errors.New("download exceeds 10 GiB limit")
	}
	if response.ContentLength >= 0 {
		job.TotalBytes = &response.ContentLength
	}
	if err := conn.Save(job).Error; err != nil {
		return err
	}
	buffer := make([]byte, 128<<10)
	lastSaved := time.Now()
	for {
		n, readErr := response.Body.Read(buffer)
		if n > 0 {
			job.DownloadedBytes += int64(n)
			if job.DownloadedBytes > dictionaryMaxDownloadBytes {
				return errors.New("download exceeds 10 GiB limit")
			}
			if _, err := file.Write(buffer[:n]); err != nil {
				return fmt.Errorf("write download: %w", err)
			}
		}
		if time.Since(lastSaved) >= time.Second || readErr != nil {
			if err := conn.Save(job).Error; err != nil {
				return err
			}
			lastSaved = time.Now()
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return fmt.Errorf("read download: %w", readErr)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
}

func (w *DictionaryWorker) extract(ctx context.Context, conn *gorm.DB, job *models.DictionaryImportJob, input io.Reader) error {
	reader := bufio.NewReaderSize(input, 64<<10)
	batch := make([]kaikki.Idiom, 0, dictionaryBatchSize)
	pending := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		line, tooLarge, err := readDictionaryLine(reader)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read dictionary: %w", err)
		}
		if len(line) == 0 && !tooLarge && err == io.EOF {
			break
		}
		job.Processed++
		pending++
		var idiom *kaikki.Idiom
		var recordErr error
		if tooLarge {
			recordErr = errors.New("record exceeds 8 MiB limit")
		} else {
			idiom, recordErr = kaikki.Extract(job.Edition, line)
		}
		if recordErr != nil {
			job.Failed++
			if len(job.RecordErrors) < 10 {
				job.RecordErrors = append(job.RecordErrors, fmt.Sprintf("Line %d: %s", job.Processed, recordErr))
			}
		} else if idiom == nil {
			job.Skipped++
		} else {
			batch = append(batch, *idiom)
		}
		if pending == dictionaryBatchSize {
			if err := commitDictionaryBatch(conn, job, batch); err != nil {
				return err
			}
			pending, batch = 0, batch[:0]
		}
		if err == io.EOF {
			break
		}
	}
	return commitDictionaryBatch(conn, job, batch)
}

func readDictionaryLine(reader *bufio.Reader) ([]byte, bool, error) {
	var line []byte
	tooLarge := false
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(line)+len(fragment) > dictionaryMaxLineBytes {
			tooLarge = true
		}
		if !tooLarge {
			line = append(line, fragment...)
		}
		if err != bufio.ErrBufferFull {
			return line, tooLarge, err
		}
	}
}

func commitDictionaryBatch(conn *gorm.DB, job *models.DictionaryImportJob, batch []kaikki.Idiom) error {
	return conn.Transaction(func(tx *gorm.DB) error {
		// Share the ordinary vocabulary writer's lock, including its case-insensitive lookup.
		if len(batch) > 0 {
			if err := lockWordWrites(tx); err != nil {
				return err
			}
		}
		for _, idiom := range batch {
			word, created, err := getOrCreateWordLocked(tx, idiom.Word, idiom.Language)
			if err != nil {
				return err
			}
			if word.Type == enums.WordTypeIdiom {
				job.Skipped++
				continue
			}
			if err := tx.Model(word).Update("type", enums.WordTypeIdiom).Error; err != nil {
				return err
			}
			if created {
				job.Inserted++
			} else {
				job.Classified++
			}
		}
		return tx.Save(job).Error
	})
}
