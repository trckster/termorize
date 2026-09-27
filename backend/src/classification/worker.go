package classification

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"termorize/src/enums"
	"termorize/src/logger"
	"time"

	"github.com/google/uuid"
)

type Decide func(context.Context, string, enums.Language) (enums.PartOfSpeech, error)

type task struct {
	id       uuid.UUID
	attempts int
	ready    time.Time
}

type Worker struct {
	store                            Store
	decide                           Decide
	capacity, batchSize, maxAttempts int
	timeout, backoff                 time.Duration
	mu                               sync.Mutex
	pending                          map[uuid.UUID]*task
	queue                            []*task
	sweeping                         bool
	sweepRequested                   bool
	stopped                          bool
	wake                             chan struct{}
	running                          atomic.Bool
}

func NewWorker(store Store, decide Decide) *Worker {
	return &Worker{
		store: store, decide: decide, capacity: 1024, batchSize: 64, maxAttempts: 3,
		timeout: 15 * time.Second, backoff: time.Second,
		pending: make(map[uuid.UUID]*task), wake: make(chan struct{}, 1),
	}
}

func (w *Worker) signal() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

func (w *Worker) Enqueue(id uuid.UUID) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return false
	}
	if _, exists := w.pending[id]; exists {
		return true
	}
	if len(w.pending) >= w.capacity {
		return false
	}
	item := &task{id: id}
	w.pending[id] = item
	w.queue = append(w.queue, item)
	w.signal()
	return true
}

func (w *Worker) RequestSweep() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return false
	}
	if !w.sweeping {
		w.sweeping, w.sweepRequested = true, true
		w.signal()
	}
	return true
}

// Run owns inference, retries, and keyset scans in one goroutine. A panic loses only in-memory work.
func (w *Worker) Run(ctx context.Context) {
	if !w.running.CompareAndSwap(false, true) {
		return
	}
	defer w.stop()
	for ctx.Err() == nil {
		if err := w.runSafely(ctx, false); err != nil && ctx.Err() == nil {
			logger.L().Errorw("classification worker restarting", "error", err)
			w.mu.Lock()
			w.pending = make(map[uuid.UUID]*task)
			w.queue = nil
			w.sweeping, w.sweepRequested = true, true
			w.mu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.backoff):
			}
		}
	}
}

func (w *Worker) RunSweep(ctx context.Context) error {
	if !w.running.CompareAndSwap(false, true) {
		return errors.New("classification worker already started")
	}
	defer w.stop()
	w.RequestSweep()
	return w.runSafely(ctx, true)
}

func (w *Worker) stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stopped = true
	w.pending = make(map[uuid.UUID]*task)
	w.queue = nil
}

func (w *Worker) runSafely(ctx context.Context, stopAfterSweep bool) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("panic: %v", value)
		}
	}()
	var cursor uuid.UUID
	var batch []uuid.UUID
	sweep := false
	scanAttempts := 0
	var scanFailure error
	failedWords := 0
	var nextScan time.Time
	preferSweep := false
	for ctx.Err() == nil {
		w.mu.Lock()
		if w.sweepRequested {
			w.sweepRequested = false
			sweep, cursor, batch, scanAttempts, nextScan = true, uuid.Nil, nil, 0, time.Time{}
			logger.L().Infow("classification sweep started")
		}
		w.mu.Unlock()

		if sweep && len(batch) == 0 && !time.Now().Before(nextScan) {
			scanCtx, cancel := context.WithTimeout(ctx, w.timeout)
			var scanErr error
			batch, scanErr = w.store.Pending(scanCtx, cursor, w.batchSize)
			cancel()
			if scanErr != nil {
				scanAttempts++
				logger.L().Warnw("classification sweep query failed", "attempt", scanAttempts, "error", scanErr)
				nextScan = time.Now().Add(w.backoff * time.Duration(scanAttempts))
				if scanAttempts >= w.maxAttempts {
					sweep = false
					scanFailure = fmt.Errorf("classification sweep query failed: %w", scanErr)
				}
			} else {
				scanAttempts = 0
				if len(batch) == 0 {
					sweep = false
				}
			}
			if !sweep {
				w.mu.Lock()
				w.sweeping = false
				w.mu.Unlock()
				logger.L().Infow("classification sweep finished", "scan_failed", scanErr != nil)
			}
		}

		w.mu.Lock()
		var item *task
		if preferSweep && len(batch) > 0 && len(w.pending) < w.capacity {
			id := batch[0]
			batch, cursor = batch[1:], id
			if _, exists := w.pending[id]; !exists {
				item = &task{id: id}
				w.pending[id] = item
			}
		}
		if item == nil {
			for i, candidate := range w.queue {
				if !time.Now().Before(candidate.ready) {
					item = candidate
					w.queue = append(w.queue[:i], w.queue[i+1:]...)
					break
				}
			}
		}
		preferSweep = !preferSweep
		w.mu.Unlock()
		if item != nil {
			attemptCtx, cancel := context.WithTimeout(ctx, w.timeout)
			err := w.process(attemptCtx, item.id)
			cancel()
			w.mu.Lock()
			item.attempts++
			if err != nil && item.attempts < w.maxAttempts && ctx.Err() == nil {
				item.ready = time.Now().Add(w.backoff * time.Duration(item.attempts))
				w.queue = append(w.queue, item)
			} else {
				delete(w.pending, item.id)
				if err != nil {
					failedWords++
				}
			}
			w.mu.Unlock()
			if err != nil {
				logger.L().Warnw("word classification failed", "word_id", item.id, "attempt", item.attempts, "error", err)
			}
			continue
		}
		if len(batch) > 0 {
			w.mu.Lock()
			room := len(w.pending) < w.capacity
			w.mu.Unlock()
			if room {
				continue
			}
		}
		if stopAfterSweep && !sweep {
			w.mu.Lock()
			idle := len(w.pending) == 0
			w.mu.Unlock()
			if idle {
				if failedWords > 0 {
					return errors.Join(scanFailure, fmt.Errorf("classification failed for %d words", failedWords))
				}
				return scanFailure
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.wake:
		case <-time.After(50 * time.Millisecond):
		}
	}
	return ctx.Err()
}

func (w *Worker) process(ctx context.Context, id uuid.UUID) error {
	word, err := w.store.Load(ctx, id)
	if err != nil || word == nil {
		return err
	}
	if word.PartOfSpeech != nil {
		return nil
	}
	category, err := w.decide(ctx, word.Word, word.Language)
	if err != nil {
		return err
	}
	if !category.Valid() {
		return errors.New("classifier returned an invalid category")
	}
	return w.store.Save(ctx, *word, category)
}
