package classification

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"termorize/src/enums"
	"termorize/src/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryStore struct {
	sync.Mutex
	words                                    map[uuid.UUID]models.Word
	pages                                    []uuid.UUID
	loadFailures, saveFailures, scanFailures int
}

func (s *memoryStore) Load(ctx context.Context, id uuid.UUID) (*models.Word, error) {
	s.Lock()
	defer s.Unlock()
	if s.loadFailures > 0 {
		s.loadFailures--
		return nil, errors.New("load failed")
	}
	word, ok := s.words[id]
	if !ok {
		return nil, nil
	}
	return &word, nil
}

func (s *memoryStore) Save(ctx context.Context, input models.Word, category enums.PartOfSpeech) error {
	s.Lock()
	defer s.Unlock()
	if s.saveFailures > 0 {
		s.saveFailures--
		return errors.New("save failed")
	}
	word, ok := s.words[input.ID]
	if ok && word.PartOfSpeech == nil && word.Word == input.Word && word.Language == input.Language {
		word.PartOfSpeech = &category
		s.words[input.ID] = word
	}
	return nil
}

func (s *memoryStore) Pending(ctx context.Context, after uuid.UUID, limit int) ([]uuid.UUID, error) {
	s.Lock()
	defer s.Unlock()
	s.pages = append(s.pages, after)
	if s.scanFailures > 0 {
		s.scanFailures--
		return nil, errors.New("scan failed")
	}
	ids := []uuid.UUID{}
	for id, word := range s.words {
		if word.PartOfSpeech == nil && id.String() > after.String() {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

func newStore(count int) *memoryStore {
	s := &memoryStore{words: map[uuid.UUID]models.Word{}}
	for i := 1; i <= count; i++ {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", i))
		s.words[id] = models.Word{ID: id, Word: fmt.Sprint(i), Language: enums.LanguageIt}
	}
	return s
}

func launch(t *testing.T, w *Worker) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	w.backoff, w.timeout, w.batchSize = time.Millisecond, time.Second, 2
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); w.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("worker did not stop")
		}
	})
	return cancel, done
}

func TestBoundedDeliveryDeduplicationAndShutdown(t *testing.T) {
	s := newStore(3)
	var calls atomic.Int32
	w := NewWorker(s, func(context.Context, string, enums.Language) (enums.PartOfSpeech, error) {
		calls.Add(1)
		return enums.PartOfSpeechUnknown, nil
	})
	w.capacity = 2
	ids, _ := s.Pending(context.Background(), uuid.Nil, 3)
	require.True(t, w.Enqueue(ids[0]))
	require.True(t, w.Enqueue(ids[0]))
	require.True(t, w.Enqueue(ids[1]))
	require.False(t, w.Enqueue(ids[2]))
	cancel, done := launch(t, w)
	require.Eventually(t, func() bool { s.Lock(); defer s.Unlock(); return s.words[ids[1]].PartOfSpeech != nil }, time.Second, time.Millisecond)
	assert.Equal(t, int32(2), calls.Load())
	w.RequestSweep()
	require.Eventually(t, func() bool { s.Lock(); defer s.Unlock(); return s.words[ids[2]].PartOfSpeech != nil }, time.Second, time.Millisecond)
	assert.Equal(t, int32(3), calls.Load(), "completed unknowns must not be retried")
	cancel()
	<-done
	assert.False(t, w.Enqueue(uuid.New()))
	assert.False(t, w.RequestSweep())
}

func TestSweepVisitsAllBatchesCoalescesAndLetsFreshWordsProgress(t *testing.T) {
	s := newStore(9)
	started := make(chan string, 20)
	release := make(chan struct{})
	w := NewWorker(s, func(ctx context.Context, term string, language enums.Language) (enums.PartOfSpeech, error) {
		assert.Equal(t, enums.LanguageIt, language)
		started <- term
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-release:
		}
		return enums.PartOfSpeechNoun, nil
	})
	w.RequestSweep()
	launch(t, w)
	assert.Equal(t, "1", <-started)
	for i := 0; i < 20; i++ {
		require.True(t, w.RequestSweep())
	}
	fresh := models.Word{ID: uuid.New(), Word: "fresh", Language: enums.LanguageIt}
	s.Lock()
	s.words[fresh.ID] = fresh
	s.Unlock()
	w.Enqueue(fresh.ID)
	close(release)
	assert.Equal(t, "fresh", <-started, "fresh words must progress during backfill")
	require.Eventually(t, func() bool {
		s.Lock()
		defer s.Unlock()
		for _, word := range s.words {
			if word.PartOfSpeech == nil {
				return false
			}
		}
		return true
	}, time.Second, time.Millisecond)
	require.Eventually(t, func() bool { w.mu.Lock(); defer w.mu.Unlock(); return !w.sweeping }, time.Second, time.Millisecond)
	s.Lock()
	defer s.Unlock()
	assert.Equal(t, uuid.Nil, s.pages[0])
	for i := 1; i < len(s.pages); i++ {
		assert.Greater(t, s.pages[i].String(), s.pages[i-1].String())
	}
}

func TestFailedRowsHaveBoundedRetriesAndWaitForLaterSweep(t *testing.T) {
	s := newStore(5)
	var mu sync.Mutex
	calls := map[string]int{}
	w := NewWorker(s, func(ctx context.Context, term string, language enums.Language) (enums.PartOfSpeech, error) {
		mu.Lock()
		defer mu.Unlock()
		calls[term]++
		if term == "1" {
			return "", errors.New("provider unavailable")
		}
		return enums.PartOfSpeechNoun, nil
	})
	w.RequestSweep()
	launch(t, w)
	finished := func() bool { w.mu.Lock(); defer w.mu.Unlock(); return !w.sweeping && len(w.pending) == 0 }
	require.Eventually(t, finished, time.Second, time.Millisecond)
	mu.Lock()
	assert.Equal(t, 3, calls["1"])
	assert.Equal(t, 1, calls["5"])
	mu.Unlock()
	w.RequestSweep()
	require.Eventually(t, finished, time.Second, time.Millisecond)
	mu.Lock()
	assert.Equal(t, 6, calls["1"])
	assert.Equal(t, 1, calls["5"])
	mu.Unlock()
}

func TestDatabaseFailuresRetryAndRestartRecoversNULLRows(t *testing.T) {
	s := newStore(3)
	s.loadFailures, s.saveFailures, s.scanFailures = 1, 1, 1
	var panicked atomic.Bool
	w := NewWorker(s, func(context.Context, string, enums.Language) (enums.PartOfSpeech, error) {
		if panicked.CompareAndSwap(false, true) {
			panic("unexpected failure")
		}
		return enums.PartOfSpeechNoun, nil
	})
	w.RequestSweep()
	launch(t, w)
	require.Eventually(t, func() bool {
		s.Lock()
		defer s.Unlock()
		for _, word := range s.words {
			if word.PartOfSpeech == nil {
				return false
			}
		}
		return true
	}, 2*time.Second, time.Millisecond)
}

func TestCancellationLeavesNULLAndNewWorkerRecovers(t *testing.T) {
	s := newStore(1)
	started := make(chan struct{})
	w := NewWorker(s, func(ctx context.Context, _ string, _ enums.Language) (enums.PartOfSpeech, error) {
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	})
	w.RequestSweep()
	cancel, done := launch(t, w)
	<-started
	cancel()
	<-done
	ids, err := s.Pending(context.Background(), uuid.Nil, 2)
	require.NoError(t, err)
	require.Len(t, ids, 1)
	next := NewWorker(s, func(context.Context, string, enums.Language) (enums.PartOfSpeech, error) {
		return enums.PartOfSpeechNoun, nil
	})
	next.RequestSweep()
	launch(t, next)
	require.Eventually(t, func() bool { s.Lock(); defer s.Unlock(); return s.words[ids[0]].PartOfSpeech != nil }, time.Second, time.Millisecond)
}

func TestWorkerReloadsAndSkipsDeletedOrCompletedWords(t *testing.T) {
	s := newStore(2)
	ids, _ := s.Pending(context.Background(), uuid.Nil, 2)
	w := NewWorker(s, func(context.Context, string, enums.Language) (enums.PartOfSpeech, error) {
		t.Error("completed/deleted word reached Jev")
		return enums.PartOfSpeechVerb, nil
	})
	w.Enqueue(ids[0])
	w.Enqueue(ids[1])
	category := enums.PartOfSpeechUnknown
	word := s.words[ids[0]]
	word.PartOfSpeech = &category
	s.words[ids[0]] = word
	delete(s.words, ids[1])
	launch(t, w)
	require.Eventually(t, func() bool { w.mu.Lock(); defer w.mu.Unlock(); return len(w.pending) == 0 }, time.Second, time.Millisecond)
}

func TestRunSweepDrainsRetriesAndStopsAfterAllBatches(t *testing.T) {
	s := newStore(5)
	calls := map[string]int{}
	w := NewWorker(s, func(_ context.Context, term string, _ enums.Language) (enums.PartOfSpeech, error) {
		calls[term]++
		if term == "1" || term == "5" && calls[term] == 1 {
			return "", errors.New("provider unavailable")
		}
		return enums.PartOfSpeechUnknown, nil
	})
	w.backoff, w.batchSize = time.Millisecond, 2
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.ErrorContains(t, w.RunSweep(ctx), "classification failed for 1 words")
	assert.Equal(t, map[string]int{"1": 3, "2": 1, "3": 1, "4": 1, "5": 2}, calls)
	ids, err := s.Pending(ctx, uuid.Nil, 10)
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.Equal(t, "1", s.words[ids[0]].Word)
	assert.False(t, w.Enqueue(uuid.New()), "finite worker must stop accepting work")

	next := NewWorker(s, func(_ context.Context, term string, _ enums.Language) (enums.PartOfSpeech, error) {
		assert.Equal(t, "1", term, "later jobs must only retry unfinished words")
		return enums.PartOfSpeechNoun, nil
	})
	require.NoError(t, next.RunSweep(ctx))
	ids, err = s.Pending(ctx, uuid.Nil, 10)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestRunSweepEmptyDatabaseExits(t *testing.T) {
	w := NewWorker(newStore(0), nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, w.RunSweep(ctx))
}

func TestRunSweepReportsScanFailuresAndPanics(t *testing.T) {
	for _, scenario := range []string{"scan failure", "panic"} {
		t.Run(scenario, func(t *testing.T) {
			s := newStore(1)
			if scenario == "scan failure" {
				s.scanFailures = 10
			}
			w := NewWorker(s, func(context.Context, string, enums.Language) (enums.PartOfSpeech, error) {
				panic("unexpected failure")
			})
			w.backoff = time.Millisecond
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := w.RunSweep(ctx)
			if scenario == "scan failure" {
				require.ErrorContains(t, err, "classification sweep query failed")
				assert.Len(t, s.pages, 3, "query retries must be bounded")
			} else {
				require.ErrorContains(t, err, "panic: unexpected failure")
			}
			for _, word := range s.words {
				assert.Nil(t, word.PartOfSpeech)
			}
		})
	}
}

func TestRunSweepCancellationLeavesUnfinishedWordsPending(t *testing.T) {
	s := newStore(1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := NewWorker(s, func(ctx context.Context, _ string, _ enums.Language) (enums.PartOfSpeech, error) {
		cancel()
		return "", ctx.Err()
	})
	require.ErrorIs(t, w.RunSweep(ctx), context.Canceled)
	for _, word := range s.words {
		assert.Nil(t, word.PartOfSpeech)
	}
}
