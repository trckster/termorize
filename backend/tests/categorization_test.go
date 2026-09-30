package tests

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sync"
	"termorize/src/classification"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func categoryWord(t *testing.T, text string, category *enums.PartOfSpeech) models.Word {
	t.Helper()
	word := models.Word{Word: text, Language: enums.LanguageEn, PartOfSpeech: category}
	require.NoError(t, db.DB.Create(&word).Error)
	return word
}

func TestCategorizationMigrationDefaultsAndManualDatabaseEdits(t *testing.T) {
	migration, err := os.ReadFile("src/data/migrations/0031_add_word_part_of_speech.sql")
	require.NoError(t, err)
	require.NoError(t, db.DB.Connection(func(conn *gorm.DB) error {
		conn = conn.Session(&gorm.Session{NewDB: true})
		schema := "classification_" + uuid.New().String()[:8]
		require.NoError(t, conn.Exec("CREATE SCHEMA "+schema).Error)
		defer conn.Exec("DROP SCHEMA " + schema + " CASCADE")
		require.NoError(t, conn.Exec("SET search_path TO "+schema).Error)
		defer conn.Exec("SET search_path TO public")
		require.NoError(t, conn.Exec("CREATE TABLE words (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), word text, language text, type text DEFAULT 'unknown')").Error)
		require.NoError(t, conn.Exec("INSERT INTO words (word) VALUES ('existing')").Error)
		require.NoError(t, conn.Exec(string(migration)).Error)
		require.NoError(t, conn.Exec("INSERT INTO words (word) VALUES ('new')").Error)
		var pending int64
		require.NoError(t, conn.Raw("SELECT count(*) FROM words WHERE part_of_speech IS NULL").Scan(&pending).Error)
		assert.Equal(t, int64(2), pending)
		assert.Error(t, conn.Exec("UPDATE words SET part_of_speech = 'article'").Error)
		for _, value := range []any{"noun", "verb", "unknown", nil} {
			require.NoError(t, conn.Exec("UPDATE words SET part_of_speech = ?", value).Error)
			var matching int64
			require.NoError(t, conn.Raw("SELECT count(*) FROM words WHERE part_of_speech IS NOT DISTINCT FROM ?::part_of_speech", value).Scan(&matching).Error)
			assert.Equal(t, int64(2), matching)
		}
		require.NoError(t, conn.Exec("UPDATE words SET type = 'idiom'").Error)
		return nil
	}))
}

func TestWordEventsWaitForOutermostCommitAndDiscardRollbacks(t *testing.T) {
	testkit.Truncate(t)
	var events []uuid.UUID
	t.Cleanup(db.SetWordCreatedHandler(func(id uuid.UUID) bool {
		var word models.Word
		assert.NoError(t, db.DB.First(&word, "id = ?", id).Error, "event must be externally visible")
		events = append(events, id)
		return true
	}))
	errRollback := errors.New("rollback")
	var kept *models.Word
	require.NoError(t, db.WordTransaction(db.DB, func(tx *gorm.DB) error {
		var err error
		kept, err = services.GetOrCreateWord(tx, "kept", enums.LanguageEn)
		require.NoError(t, err)
		assert.Empty(t, events)
		err = db.WordTransaction(tx, func(nested *gorm.DB) error {
			_, err := services.GetOrCreateWord(nested, "rolled back savepoint", enums.LanguageEn)
			require.NoError(t, err)
			return errRollback
		})
		require.ErrorIs(t, err, errRollback)
		assert.Empty(t, events)
		return nil
	}))
	assert.Equal(t, []uuid.UUID{kept.ID}, events)
	err := db.WordTransaction(db.DB, func(tx *gorm.DB) error {
		_, err := services.GetOrCreateWord(tx, "outer rollback", enums.LanguageEn)
		require.NoError(t, err)
		return errRollback
	})
	require.ErrorIs(t, err, errRollback)
	assert.Len(t, events, 1)
	_, err = services.GetOrCreateWord(db.DB, "kept", enums.LanguageEn)
	require.NoError(t, err)
	assert.Len(t, events, 1, "existing words do not emit insertion events")
	require.Panics(t, func() {
		_ = db.WordTransaction(db.DB, func(tx *gorm.DB) error {
			_, err := services.GetOrCreateWord(tx, "panic rollback", enums.LanguageEn)
			require.NoError(t, err)
			panic("rollback")
		})
	})
	assert.Len(t, events, 1)
	assert.Error(t, db.DB.Transaction(func(tx *gorm.DB) error {
		_, err := services.GetOrCreateWord(tx, "untracked", enums.LanguageEn)
		return err
	}))
}

func TestCategorizationAdminAuthorizationValidationAndSharedUpdates(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	user := testkit.CreateUser(t)
	unknown := enums.PartOfSpeechUnknown
	word := categoryWord(t, "shared", &unknown)
	require.NoError(t, db.DB.Model(&word).Update("type", enums.TypeIdiom).Error)
	path := "/api/admin/words/" + word.ID.String() + "/part-of-speech"
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/admin/categorization/unknown"},
		{http.MethodGet, "/api/admin/categorization/mismatches"},
		{http.MethodPut, path},
	} {
		testkit.RequireStatus(t, testkit.Request(t, route.method, route.path, nil), http.StatusUnauthorized)
		testkit.RequireStatus(t, testkit.AuthedRequest(t, user, route.method, route.path, nil), http.StatusForbidden)
	}
	for _, value := range []any{"article", "", nil, 42} {
		testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPut, path, map[string]any{"part_of_speech": value}), http.StatusBadRequest)
	}
	for _, query := range []string{"page=0", "page=-1", "page=abc", "page_size=0", "page_size=101"} {
		for _, list := range []string{"unknown", "mismatches"} {
			testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/categorization/"+list+"?"+query, nil), http.StatusBadRequest)
		}
	}
	pending := categoryWord(t, "pending", nil)
	noun := enums.PartOfSpeechNoun
	target := categoryWord(t, "target", &noun)
	translation := models.Translation{OriginalID: word.ID, TranslationID: target.ID, Source: enums.TranslationSourceGoogle}
	require.NoError(t, db.DB.Create(&translation).Error)
	for _, owner := range []models.User{admin, user} {
		require.NoError(t, db.DB.Create(&models.Vocabulary{UserID: owner.ID, TranslationID: translation.ID}).Error)
	}
	response := testkit.AuthedRequest(t, admin, http.MethodPut, path, map[string]any{"part_of_speech": "noun"})
	testkit.RequireStatus(t, response, http.StatusOK)
	var saved models.Word
	testkit.DecodeJSON(t, response, &saved)
	assert.Equal(t, word.ID, saved.ID)
	require.NotNil(t, saved.PartOfSpeech)
	assert.Equal(t, noun, *saved.PartOfSpeech)
	assert.Equal(t, enums.TypeIdiom, saved.Type)
	for _, owner := range []models.User{admin, user} {
		response := testkit.AuthedRequest(t, owner, http.MethodGet, "/api/vocabulary", nil)
		testkit.RequireStatus(t, response, http.StatusOK)
		var list services.VocabularyListResponse
		testkit.DecodeJSON(t, response, &list)
		require.Len(t, list.Data, 1)
		assert.Equal(t, &noun, list.Data[0].Translation.Original.PartOfSpeech)
		assert.Equal(t, &noun, list.Data[0].Translation.Translation.PartOfSpeech)
	}
	for _, value := range []string{"noun", "verb", "unknown"} {
		testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPut, path, map[string]any{"part_of_speech": value}), http.StatusConflict)
	}
	testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPut, "/api/admin/words/"+pending.ID.String()+"/part-of-speech", map[string]any{"part_of_speech": "verb"}), http.StatusOK)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPut, "/api/admin/words/"+uuid.NewString()+"/part-of-speech", map[string]any{"part_of_speech": "verb"}), http.StatusNotFound)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPut, "/api/admin/words/no-id/part-of-speech", map[string]any{"part_of_speech": "verb"}), http.StatusBadRequest)
}

func TestCategorizationListsUnknownsAndEveryCompletedMismatch(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	other := testkit.CreateUser(t)
	noun, verb, unknown := enums.PartOfSpeechNoun, enums.PartOfSpeechVerb, enums.PartOfSpeechUnknown
	a := categoryWord(t, "noun", &noun)
	b := categoryWord(t, "verb", &verb)
	u := categoryWord(t, "unknown", &unknown)
	p := categoryWord(t, "pending", nil)
	aTwin := categoryWord(t, "another noun", &noun)
	uTwin := categoryWord(t, "another unknown", &unknown)
	var expected []uuid.UUID
	for _, pair := range []struct {
		left, right models.Word
		owners      []models.User
		mismatch    bool
	}{
		{a, b, []models.User{admin, other}, true},
		{a, u, []models.User{admin}, true},
		{u, b, []models.User{admin}, true},
		{a, p, []models.User{admin}, false},
		{p, u, []models.User{admin}, false},
		{a, aTwin, []models.User{admin}, false},
		{u, uTwin, []models.User{admin}, false},
	} {
		translation := models.Translation{OriginalID: pair.left.ID, TranslationID: pair.right.ID, Source: enums.TranslationSourceGoogle}
		require.NoError(t, db.DB.Create(&translation).Error)
		for _, owner := range pair.owners {
			vocab := models.Vocabulary{UserID: owner.ID, TranslationID: translation.ID}
			require.NoError(t, db.DB.Create(&vocab).Error)
			if pair.mismatch {
				expected = append(expected, vocab.ID)
			}
		}
	}
	response := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/categorization/unknown", nil)
	testkit.RequireStatus(t, response, http.StatusOK)
	var words services.UnknownWordsResponse
	testkit.DecodeJSON(t, response, &words)
	assert.Equal(t, int64(2), words.Pagination.Total)
	require.Len(t, words.Data, 2)
	assert.ElementsMatch(t, []uuid.UUID{u.ID, uTwin.ID}, []uuid.UUID{words.Data[0].ID, words.Data[1].ID})
	var ids []uuid.UUID
	for _, page := range []string{"1", "2"} {
		response = testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/categorization/mismatches?page_size=2&page="+page, nil)
		testkit.RequireStatus(t, response, http.StatusOK)
		var list services.VocabularyListResponse
		testkit.DecodeJSON(t, response, &list)
		assert.Equal(t, int64(4), list.Pagination.Total)
		assert.Equal(t, 2, list.Pagination.TotalPages)
		require.Len(t, list.Data, 2)
		for _, row := range list.Data {
			ids = append(ids, row.ID)
			require.NotNil(t, row.Translation.Original.PartOfSpeech)
			require.NotNil(t, row.Translation.Translation.PartOfSpeech)
			assert.NotEqual(t, *row.Translation.Original.PartOfSpeech, *row.Translation.Translation.PartOfSpeech)
		}
	}
	assert.ElementsMatch(t, expected, ids)
}

func TestCategorizationMismatchesExcludeDeletedVocabulary(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	other := testkit.CreateUser(t)
	noun, verb := enums.PartOfSpeechNoun, enums.PartOfSpeechVerb
	original := categoryWord(t, "decision", &noun)
	translated := categoryWord(t, "decide", &verb)
	translation := models.Translation{OriginalID: original.ID, TranslationID: translated.ID, Source: enums.TranslationSourceGoogle}
	require.NoError(t, db.DB.Create(&translation).Error)
	active := models.Vocabulary{UserID: admin.ID, TranslationID: translation.ID}
	deleted := models.Vocabulary{UserID: other.ID, TranslationID: translation.ID}
	require.NoError(t, db.DB.Create(&active).Error)
	require.NoError(t, db.DB.Create(&deleted).Error)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, other, http.MethodDelete, "/api/vocabulary/"+deleted.ID.String(), nil), http.StatusOK)

	response := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/categorization/mismatches?page_size=1", nil)
	testkit.RequireStatus(t, response, http.StatusOK)
	var list services.VocabularyListResponse
	testkit.DecodeJSON(t, response, &list)
	assert.Equal(t, int64(1), list.Pagination.Total)
	assert.Equal(t, 1, list.Pagination.TotalPages)
	require.Len(t, list.Data, 1)
	assert.Equal(t, active.ID, list.Data[0].ID)
	assert.Nil(t, list.Data[0].DeletedAt)
}

type observedClassificationStore struct {
	classification.WordStore
	saved chan struct{}
}

func (s observedClassificationStore) Save(ctx context.Context, word models.Word, category enums.PartOfSpeech) error {
	err := s.WordStore.Save(ctx, word, category)
	s.saved <- struct{}{}
	return err
}

func TestConditionalClassificationSavesAndManualRaces(t *testing.T) {
	for _, change := range []string{"manual", "text", "language", "deleted"} {
		t.Run(change, func(t *testing.T) {
			testkit.Truncate(t)
			word := categoryWord(t, "word", nil)
			started, release := make(chan struct{}), make(chan struct{})
			saved := make(chan struct{}, 1)
			worker := classification.NewWorker(observedClassificationStore{WordStore: classification.WordStore{DB: db.DB}, saved: saved}, func(ctx context.Context, text string, language enums.Language) (enums.PartOfSpeech, error) {
				assert.Equal(t, "word", text)
				assert.Equal(t, enums.LanguageEn, language)
				close(started)
				select {
				case <-release:
				case <-ctx.Done():
					return "", ctx.Err()
				}
				return enums.PartOfSpeechNoun, nil
			})
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() { defer close(done); worker.Run(ctx) }()
			t.Cleanup(func() { cancel(); <-done })
			worker.Enqueue(word.ID)
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("classification did not start")
			}
			switch change {
			case "manual":
				_, err := services.SetWordPartOfSpeech(context.Background(), word.ID, enums.PartOfSpeechVerb)
				require.NoError(t, err)
			case "text":
				require.NoError(t, db.DB.Model(&word).Update("word", "changed").Error)
			case "language":
				require.NoError(t, db.DB.Model(&word).Update("language", enums.LanguageIt).Error)
			case "deleted":
				require.NoError(t, db.DB.Delete(&word).Error)
			}
			close(release)
			select {
			case <-saved:
			case <-time.After(time.Second):
				t.Fatal("classification did not finish saving")
			}
			cancel()
			<-done
			var stored models.Word
			err := db.DB.First(&stored, "id = ?", word.ID).Error
			if change == "deleted" {
				assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
			} else {
				require.NoError(t, err)
				if change == "manual" {
					require.NotNil(t, stored.PartOfSpeech)
					assert.Equal(t, enums.PartOfSpeechVerb, *stored.PartOfSpeech)
				} else {
					assert.Nil(t, stored.PartOfSpeech)
				}
			}
		})
	}
}

func TestConcurrentManualChoicesOnlyOneCanWin(t *testing.T) {
	testkit.Truncate(t)
	word := categoryWord(t, "race", nil)
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, category := range []enums.PartOfSpeech{enums.PartOfSpeechNoun, enums.PartOfSpeechVerb} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := services.SetWordPartOfSpeech(context.Background(), word.ID, category)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for err := range results {
		if err == nil {
			wins++
		} else if errors.Is(err, services.ErrPartOfSpeechPermanent) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	assert.Equal(t, 1, wins)
	assert.Equal(t, 1, conflicts)
}

func TestStandaloneClassificationSweepPersistsAllPendingWords(t *testing.T) {
	testkit.Truncate(t)
	pending := make([]models.Word, 70)
	for i := range pending {
		pending[i] = models.Word{Word: uuid.NewString(), Language: enums.LanguageEn}
	}
	require.NoError(t, db.DB.Create(&pending).Error)
	for _, category := range []enums.PartOfSpeech{enums.PartOfSpeechNoun, enums.PartOfSpeechUnknown} {
		categoryWord(t, string(category), &category)
	}
	calls := 0
	worker := classification.NewWorker(classification.WordStore{DB: db.DB}, func(_ context.Context, text string, language enums.Language) (enums.PartOfSpeech, error) {
		calls++
		assert.Equal(t, enums.LanguageEn, language)
		assert.NotContains(t, []string{"noun", "unknown"}, text)
		return enums.PartOfSpeechVerb, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, worker.RunSweep(ctx))
	assert.Equal(t, len(pending), calls)
	var categorized int64
	require.NoError(t, db.DB.Model(&models.Word{}).Where("part_of_speech = ?", enums.PartOfSpeechVerb).Count(&categorized).Error)
	assert.Equal(t, int64(len(pending)), categorized)
	require.NoError(t, classification.NewWorker(classification.WordStore{DB: db.DB}, nil).RunSweep(ctx), "a second process must skip all completed words")
}

func TestStandaloneSweepAndBackendWorkerCannotOverwriteEachOther(t *testing.T) {
	for _, winner := range []string{"backend", "hourly job"} {
		t.Run(winner, func(t *testing.T) {
			testkit.Truncate(t)
			word := categoryWord(t, "overlap", nil)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			started := make(chan struct{}, 2)
			releaseBackend, releaseSweep := make(chan struct{}), make(chan struct{})
			decide := func(release <-chan struct{}, category enums.PartOfSpeech) classification.Decide {
				return func(ctx context.Context, _ string, _ enums.Language) (enums.PartOfSpeech, error) {
					started <- struct{}{}
					select {
					case <-release:
						return category, nil
					case <-ctx.Done():
						return "", ctx.Err()
					}
				}
			}
			saved := make(chan struct{}, 2)
			store := observedClassificationStore{WordStore: classification.WordStore{DB: db.DB}, saved: saved}
			backend := classification.NewWorker(store, decide(releaseBackend, enums.PartOfSpeechNoun))
			sweep := classification.NewWorker(store, decide(releaseSweep, enums.PartOfSpeechUnknown))
			backend.Enqueue(word.ID)
			backendDone, sweepDone := make(chan struct{}), make(chan error, 1)
			go func() { defer close(backendDone); backend.Run(ctx) }()
			go func() { defer close(sweepDone); sweepDone <- sweep.RunSweep(ctx) }()
			t.Cleanup(func() { cancel(); <-backendDone; <-sweepDone })
			for i := 0; i < 2; i++ {
				select {
				case <-started:
				case <-ctx.Done():
					t.Fatal("both classifiers must start before either saves")
				}
			}
			first, second := releaseBackend, releaseSweep
			expected := enums.PartOfSpeechNoun
			if winner == "hourly job" {
				first, second = releaseSweep, releaseBackend
				expected = enums.PartOfSpeechUnknown
			}
			close(first)
			select {
			case <-saved:
			case <-ctx.Done():
				t.Fatal("first classification did not save")
			}
			close(second)
			select {
			case <-saved:
			case <-ctx.Done():
				t.Fatal("second classification did not finish")
			}
			require.NoError(t, <-sweepDone)
			require.NoError(t, db.DB.First(&word, "id = ?", word.ID).Error)
			require.NotNil(t, word.PartOfSpeech)
			assert.Equal(t, expected, *word.PartOfSpeech)
		})
	}
}
