package tests

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Contract: admin-only source listing, paginated history and asynchronous starts;
// malformed/absent sources, concurrent conflicts, durable outcomes and retry without vocabulary changes.
func TestDictionaryAdminPermissionsAndValidation(t *testing.T) {
	testkit.Truncate(t)
	dictionary := seedDictionary(t, "enwiktionary", "http://unused.invalid/dump.gz")
	user := testkit.CreateUser(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	endpoints := []struct{ method, path string }{
		{http.MethodGet, "/api/admin/dictionaries"},
		{http.MethodGet, "/api/admin/dictionaries/" + dictionary.ID.String() + "/imports"},
		{http.MethodPost, "/api/admin/dictionaries/" + dictionary.ID.String() + "/imports"},
	}
	for _, endpoint := range endpoints {
		testkit.RequireStatus(t, testkit.Request(t, endpoint.method, endpoint.path, nil), http.StatusUnauthorized)
		testkit.RequireStatus(t, testkit.AuthedRequest(t, user, endpoint.method, endpoint.path, nil), http.StatusForbidden)
	}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, method, "/api/admin/dictionaries/no-id/imports", nil), http.StatusBadRequest)
		testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, method, "/api/admin/dictionaries/"+uuid.NewString()+"/imports", nil), http.StatusNotFound)
	}
	for _, page := range []string{"0", "-1", "abc", "1000001"} {
		testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodGet, endpoints[1].path+"?page="+page, nil), http.StatusBadRequest)
	}
	rec := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/dictionaries", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	var sources []models.Dictionary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sources))
	require.Len(t, sources, 1)
	assert.Equal(t, dictionary, sources[0])
	var count int64
	require.NoError(t, db.DB.Model(&models.DictionaryImportJob{}).Count(&count).Error)
	assert.Zero(t, count)
	require.NoError(t, db.DB.Model(&dictionary).Update("edition", "unsupported").Error)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, endpoints[2].path, nil), http.StatusBadRequest)
}

func TestDictionaryImportPromotesExistingWordsAndPreservesVocabulary(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	vocabulary := exerciseSeedVocabulary(t, admin.ID, "Rain Cats and Dogs", "лить как из ведра", enums.LanguageEn, enums.LanguageRu)
	var translation models.Translation
	require.NoError(t, db.DB.First(&translation, "id = ?", vocabulary.TranslationID).Error)
	var oldWord models.Word
	require.NoError(t, db.DB.First(&oldWord, "id = ?", translation.OriginalID).Error)
	assert.Equal(t, enums.TypeUnknown, oldWord.Type)
	var originalVocabulary models.Vocabulary
	require.NoError(t, db.DB.First(&originalVocabulary, "id = ?", vocabulary.ID).Error)
	fixture, err := os.ReadFile("src/integrations/kaikki/testdata/enwiktionary.jsonl")
	require.NoError(t, err)
	data := string(fixture) + `{"word":"RAIN CATS AND DOGS","lang_code":"en","tags":["idiomatic"]}` + "\n" + `{broken` + "\n"
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Equal(t, "identity", r.Header.Get("Accept-Encoding"))
		_, _ = w.Write(gzipDictionary(t, data))
	}))
	defer server.Close()
	dictionary := seedDictionary(t, "enwiktionary", server.URL)
	worker := newDictionaryWorker(t, server.Client())
	startPath := "/api/admin/dictionaries/" + dictionary.ID.String() + "/imports"
	rec := testkit.AuthedRequest(t, admin, http.MethodPost, startPath, nil)
	testkit.RequireStatus(t, rec, http.StatusAccepted)
	var queued models.DictionaryImportJob
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &queued))
	require.NotEqual(t, uuid.Nil, queued.ID)
	assert.Equal(t, "queued", queued.Status)
	assert.Equal(t, dictionary.DownloadURL, queued.DownloadURL)
	assert.Equal(t, dictionary.Edition, queued.Edition)
	assert.Equal(t, dictionary.Name, queued.SourceName)
	assert.Zero(t, requests.Load(), "the request must only enqueue work")
	testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, startPath, nil), http.StatusConflict)
	require.NoError(t, worker.Run(context.Background()))
	job := loadDictionaryJob(t, queued.ID)
	assert.Equal(t, "succeeded", job.Status)
	assert.Equal(t, int64(7), job.Processed)
	assert.Equal(t, int64(2), job.Inserted)
	assert.Equal(t, int64(1), job.Classified)
	assert.Equal(t, int64(3), job.Skipped)
	assert.Equal(t, int64(1), job.Failed)
	assert.Equal(t, []string{"Line 7: invalid entry JSON or field types"}, job.RecordErrors)
	assert.NotNil(t, job.StartedAt)
	assert.NotNil(t, job.FinishedAt)
	assert.Greater(t, job.DownloadedBytes, int64(0))
	require.NotNil(t, job.TotalBytes)
	assert.Equal(t, *job.TotalBytes, job.DownloadedBytes)
	var promoted models.Word
	require.NoError(t, db.DB.First(&promoted, "id = ?", oldWord.ID).Error)
	assert.Equal(t, enums.TypeIdiom, promoted.Type)
	promoted.Type = oldWord.Type
	assert.Equal(t, oldWord, promoted)
	var unchanged models.Vocabulary
	require.NoError(t, db.DB.First(&unchanged, "id = ?", vocabulary.ID).Error)
	assert.Equal(t, originalVocabulary, unchanged)
	var unchangedTranslation models.Translation
	require.NoError(t, db.DB.First(&unchangedTranslation, "id = ?", translation.ID).Error)
	assert.Equal(t, translation, unchangedTranslation)
	requireDictionaryTempEmpty(t, worker.TempDir)

	repeated, err := services.StartDictionaryImport(dictionary.ID)
	require.NoError(t, err)
	require.NoError(t, worker.Run(context.Background()))
	repeated = loadDictionaryJob(t, repeated.ID)
	assert.Equal(t, "succeeded", repeated.Status)
	assert.Zero(t, repeated.Inserted)
	assert.Zero(t, repeated.Classified)
	assert.Equal(t, int64(6), repeated.Skipped)
	assert.Equal(t, int32(2), requests.Load(), "retries must download again")
	var words, vocabularies, translations int64
	require.NoError(t, db.DB.Model(&models.Word{}).Count(&words).Error)
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).Count(&vocabularies).Error)
	require.NoError(t, db.DB.Model(&models.Translation{}).Count(&translations).Error)
	assert.Equal(t, int64(4), words)
	assert.Equal(t, int64(1), vocabularies)
	assert.Equal(t, int64(1), translations)
	saved, err := services.GetOrCreateWord(db.DB, "rain cats and dogs", enums.LanguageEn)
	require.NoError(t, err)
	assert.Equal(t, oldWord.ID, saved.ID)
	assert.Equal(t, enums.TypeIdiom, saved.Type)
	ordinary, err := services.GetOrCreateWord(db.DB, "ordinary", enums.LanguageEn)
	require.NoError(t, err)
	assert.Equal(t, enums.TypeUnknown, ordinary.Type)
	rec = testkit.AuthedRequest(t, admin, http.MethodGet, startPath, nil)
	testkit.RequireStatus(t, rec, http.StatusOK)
	var history services.DictionaryImportHistory
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &history))
	require.Len(t, history.Data, 2)
	assert.Equal(t, repeated.ID, history.Data[0].ID)
	assert.Equal(t, int64(2), history.Pagination.Total)
	requireDictionaryTempEmpty(t, worker.TempDir)
}

func TestDictionaryImportSerializesWorkersAndConcurrentStarts(t *testing.T) {
	testkit.Truncate(t)
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = w.Write(gzipDictionary(t, `{"word":"piece of cake","lang_code":"en","tags":["idiomatic"]}`))
	}))
	defer server.Close()
	// Release a blocked handler before Server.Close if an assertion aborts the test.
	defer once.Do(func() { close(release) })
	dictionary := seedDictionary(t, "enwiktionary", server.URL)
	const attempts = 8
	results := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		go func() { _, err := services.StartDictionaryImport(dictionary.ID); results <- err }()
	}
	accepted, conflicts := 0, 0
	for i := 0; i < attempts; i++ {
		err := <-results
		if err == nil {
			accepted++
		} else {
			require.ErrorIs(t, err, services.ErrDictionaryImportActive)
			conflicts++
		}
	}
	assert.Equal(t, 1, accepted)
	assert.Equal(t, attempts-1, conflicts)
	worker := newDictionaryWorker(t, server.Client())
	done := make(chan error, 1)
	go func() { done <- worker.Run(context.Background()) }()
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("worker did not start download")
	}
	otherWorker := newDictionaryWorker(t, server.Client())
	require.NoError(t, otherWorker.Run(context.Background()))
	var active models.DictionaryImportJob
	require.NoError(t, db.DB.First(&active).Error)
	assert.Equal(t, "downloading", active.Status)
	once.Do(func() { close(release) })
	require.NoError(t, <-done)
	assert.Equal(t, "succeeded", loadDictionaryJob(t, active.ID).Status)
}

func TestDictionaryImportRecoversInterruptedJobsAndCleansAbandonedFiles(t *testing.T) {
	testkit.Truncate(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(gzipDictionary(t, `{"word":"piece of cake","lang_code":"en","tags":["idiomatic"]}`))
	}))
	defer server.Close()
	dictionary := seedDictionary(t, "enwiktionary", server.URL)
	interrupted, err := services.StartDictionaryImport(dictionary.ID)
	require.NoError(t, err)
	require.NoError(t, db.DB.Model(interrupted).Updates(map[string]any{"status": "importing", "processed": 500, "skipped": 500}).Error)
	worker := newDictionaryWorker(t, server.Client())
	require.NoError(t, os.WriteFile(filepath.Join(worker.TempDir, interrupted.ID.String()+".jsonl.gz"), []byte("abandoned"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(worker.TempDir, "unrelated.txt"), []byte("keep"), 0600))
	require.NoError(t, worker.Run(context.Background()))
	interrupted = loadDictionaryJob(t, interrupted.ID)
	assert.Equal(t, "interrupted", interrupted.Status)
	assert.Equal(t, int64(500), interrupted.Processed)
	assert.NotEmpty(t, interrupted.Error)
	assert.NotNil(t, interrupted.FinishedAt)
	entries, err := os.ReadDir(worker.TempDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "unrelated.txt", entries[0].Name())
	retry, err := services.StartDictionaryImport(dictionary.ID)
	require.NoError(t, err)
	require.NoError(t, worker.Run(context.Background()))
	assert.Equal(t, "succeeded", loadDictionaryJob(t, retry.ID).Status)
	assert.Equal(t, "interrupted", loadDictionaryJob(t, interrupted.ID).Status)
}

func TestDictionaryImportReportsDownloadFailuresAndCanRetry(t *testing.T) {
	for _, tt := range []struct {
		name      string
		status    int
		data      []byte
		errorText string
	}{
		{"http failure", http.StatusBadGateway, nil, "HTTP 502"},
		{"invalid gzip", http.StatusOK, []byte("not a gzip dictionary"), "open gzip"},
		{"truncated gzip", http.StatusOK, gzipDictionary(t, `{"word":"test","lang_code":"en"}`)[:15], "unexpected EOF"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testkit.Truncate(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tt.status); _, _ = w.Write(tt.data) }))
			defer server.Close()
			dictionary := seedDictionary(t, "enwiktionary", server.URL)
			job, err := services.StartDictionaryImport(dictionary.ID)
			require.NoError(t, err)
			worker := newDictionaryWorker(t, server.Client())
			require.NoError(t, worker.Run(context.Background()))
			job = loadDictionaryJob(t, job.ID)
			assert.Equal(t, "failed", job.Status)
			assert.Contains(t, job.Error, tt.errorText)
			assert.NotNil(t, job.FinishedAt)
			requireDictionaryTempEmpty(t, worker.TempDir)
			_, err = services.StartDictionaryImport(dictionary.ID)
			require.NoError(t, err)
		})
	}
}

func TestDictionaryImportOverlappingSourcesAndBoundedMalformedRecords(t *testing.T) {
	testkit.Truncate(t)
	data := strings.Repeat("{malformed}\n", 12) + strings.Repeat("x", (8<<20)+1) + "\n" +
		`{"word":"al dente","lang_code":"it","categories":["Italian idioms"]}` + "\n" +
		strings.Repeat(`{"word":"ordinary","lang_code":"en"}`+"\n", 500)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ru" {
			_, _ = w.Write(gzipDictionary(t, `{"word":"AL DENTE","lang_code":"it","categories":["Фразеологизмы/it"]}`))
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		_, _ = w.Write(gzipDictionary(t, data))
	}))
	defer server.Close()
	dictionary := seedDictionary(t, "enwiktionary", server.URL)
	otherDictionary := seedDictionary(t, "ruwiktionary", server.URL+"/ru")
	first, err := services.StartDictionaryImport(dictionary.ID)
	require.NoError(t, err)
	second, err := services.StartDictionaryImport(otherDictionary.ID)
	require.NoError(t, err)
	worker := newDictionaryWorker(t, server.Client())
	require.NoError(t, worker.Run(context.Background()))
	require.NoError(t, worker.Run(context.Background()))
	first = loadDictionaryJob(t, first.ID)
	second = loadDictionaryJob(t, second.ID)
	assert.Equal(t, "succeeded", first.Status)
	assert.Equal(t, int64(514), first.Processed)
	assert.Equal(t, int64(13), first.Failed)
	assert.Equal(t, int64(500), first.Skipped)
	assert.Equal(t, int64(1), first.Inserted)
	assert.Nil(t, first.TotalBytes, "chunked downloads must not invent a total")
	assert.Len(t, first.RecordErrors, 10)
	assert.Equal(t, "succeeded", second.Status)
	assert.Zero(t, second.Inserted)
	assert.Zero(t, second.Classified)
	assert.Equal(t, int64(1), second.Skipped)
	var words []models.Word
	require.NoError(t, db.DB.Find(&words).Error)
	require.Len(t, words, 1)
	assert.Equal(t, enums.LanguageIt, words[0].Language)
	assert.Equal(t, enums.TypeIdiom, words[0].Type)
	requireDictionaryTempEmpty(t, worker.TempDir)
}

func TestDictionaryHistoryPagination(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	dictionary := seedDictionary(t, "enwiktionary", "http://unused.invalid")
	for i := 0; i < 23; i++ {
		require.NoError(t, db.DB.Create(&models.DictionaryImportJob{DictionaryID: dictionary.ID, SourceName: dictionary.Name, Edition: dictionary.Edition, DownloadURL: dictionary.DownloadURL, Status: "succeeded", RecordErrors: []string{}, CreatedAt: time.Date(2026, 1, 1, 0, i, 0, 0, time.UTC)}).Error)
	}
	for _, tt := range []struct{ page, size int }{{1, 20}, {2, 3}, {3, 0}} {
		rec := testkit.AuthedRequest(t, admin, http.MethodGet, fmt.Sprintf("/api/admin/dictionaries/%s/imports?page=%d", dictionary.ID, tt.page), nil)
		testkit.RequireStatus(t, rec, http.StatusOK)
		var history services.DictionaryImportHistory
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &history))
		assert.Len(t, history.Data, tt.size)
		assert.Equal(t, tt.page, history.Pagination.Page)
		assert.Equal(t, 2, history.Pagination.TotalPages)
		assert.Equal(t, int64(23), history.Pagination.Total)
		for i := 1; i < len(history.Data); i++ {
			assert.True(t, history.Data[i-1].CreatedAt.After(history.Data[i].CreatedAt))
		}
	}
}

func seedDictionary(t *testing.T, edition, url string) models.Dictionary {
	t.Helper()
	dictionary := models.Dictionary{Name: edition, Edition: edition, URL: "https://kaikki.org", License: "Fixture", Attribution: "Fixture", DownloadURL: url}
	require.NoError(t, db.DB.Create(&dictionary).Error)
	return dictionary
}

func gzipDictionary(t *testing.T, text string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	_, err := writer.Write([]byte(text))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func newDictionaryWorker(t *testing.T, client *http.Client) *services.DictionaryWorker {
	t.Helper()
	return &services.DictionaryWorker{DB: db.DB, Client: client, TempDir: t.TempDir()}
}

func loadDictionaryJob(t *testing.T, id uuid.UUID) *models.DictionaryImportJob {
	t.Helper()
	var job models.DictionaryImportJob
	require.NoError(t, db.DB.First(&job, "id = ?", id).Error)
	return &job
}

func requireDictionaryTempEmpty(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(path)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestDictionaryImportCancellationPreservesRetryableOutcome(t *testing.T) {
	testkit.Truncate(t)
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	dictionary := seedDictionary(t, "enwiktionary", server.URL)
	job, err := services.StartDictionaryImport(dictionary.ID)
	require.NoError(t, err)
	worker := newDictionaryWorker(t, server.Client())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("download did not start")
	}
	cancel()
	require.NoError(t, <-done)
	job = loadDictionaryJob(t, job.ID)
	assert.Equal(t, "interrupted", job.Status)
	assert.NotEmpty(t, job.Error)
	assert.NotNil(t, job.FinishedAt)
	requireDictionaryTempEmpty(t, worker.TempDir)
	_, err = services.StartDictionaryImport(dictionary.ID)
	require.NoError(t, err)
}

func TestDictionaryImportRollsBackWordsAndProgressTogether(t *testing.T) {
	testkit.Truncate(t)
	require.NoError(t, db.DB.Exec(`CREATE FUNCTION reject_dictionary_test_word() RETURNS trigger AS $$
 BEGIN IF NEW.word = 'reject me' THEN RAISE EXCEPTION 'test batch rejection'; END IF; RETURN NEW; END;
 $$ LANGUAGE plpgsql;
 CREATE TRIGGER reject_dictionary_test_word BEFORE INSERT ON words FOR EACH ROW EXECUTE FUNCTION reject_dictionary_test_word();`).Error)
	t.Cleanup(func() {
		require.NoError(t, db.DB.Exec("DROP TRIGGER IF EXISTS reject_dictionary_test_word ON words; DROP FUNCTION IF EXISTS reject_dictionary_test_word();").Error)
	})
	data := strings.Repeat(`{"word":"ordinary","lang_code":"en"}`+"\n", 500) +
		`{"word":"piece of cake","lang_code":"en","tags":["idiomatic"]}` + "\n" +
		`{"word":"reject me","lang_code":"en","tags":["idiomatic"]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(gzipDictionary(t, data)) }))
	defer server.Close()
	dictionary := seedDictionary(t, "enwiktionary", server.URL)
	job, err := services.StartDictionaryImport(dictionary.ID)
	require.NoError(t, err)
	worker := newDictionaryWorker(t, server.Client())
	require.NoError(t, worker.Run(context.Background()))
	job = loadDictionaryJob(t, job.ID)
	assert.Equal(t, "failed", job.Status)
	assert.Equal(t, int64(500), job.Processed)
	assert.Equal(t, int64(500), job.Skipped)
	assert.Zero(t, job.Inserted)
	assert.Contains(t, job.Error, "test batch rejection")
	var words int64
	require.NoError(t, db.DB.Model(&models.Word{}).Count(&words).Error)
	assert.Zero(t, words, "the first word of the failed batch must also be rolled back")
	requireDictionaryTempEmpty(t, worker.TempDir)
}

func TestDictionaryImportAndOrdinarySavesShareCaseInsensitiveIdentity(t *testing.T) {
	testkit.Truncate(t)
	data := `{"word":"PIECE OF CAKE","lang_code":"en","tags":["idiomatic"]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(gzipDictionary(t, data)) }))
	defer server.Close()
	dictionary := seedDictionary(t, "enwiktionary", server.URL)
	job, err := services.StartDictionaryImport(dictionary.ID)
	require.NoError(t, err)
	worker := newDictionaryWorker(t, server.Client())
	results := make(chan error, 9)
	go func() { results <- worker.Run(context.Background()) }()
	for i := 0; i < 8; i++ {
		go func() { _, err := services.GetOrCreateWord(db.DB, "Piece of Cake", enums.LanguageEn); results <- err }()
	}
	for i := 0; i < 9; i++ {
		require.NoError(t, <-results)
	}
	assert.Equal(t, "succeeded", loadDictionaryJob(t, job.ID).Status)
	var words []models.Word
	require.NoError(t, db.DB.Find(&words).Error)
	require.Len(t, words, 1)
	assert.Equal(t, enums.TypeIdiom, words[0].Type)
}

func TestDictionaryWordLockDoesNotBlockSavesDuringGoogleTranslation(t *testing.T) {
	testkit.Truncate(t)
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{TranslateFunc: func(text, source, target string) (string, error) {
		close(started)
		<-release
		return "торт", nil
	}})
	done := make(chan error, 1)
	go func() { _, err := services.Translate("cake", enums.LanguageEn, enums.LanguageRu); done <- err }()
	defer func() { once.Do(func() { close(release) }); require.NoError(t, <-done) }()
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("translation did not start")
	}
	saveCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	word, err := services.GetOrCreateWord(db.DB.WithContext(saveCtx), "unrelated", enums.LanguageEn)
	require.NoError(t, err, "an unrelated save must complete before Google responds")
	assert.Equal(t, "unrelated", word.Word)
	once.Do(func() { close(release) })
}
