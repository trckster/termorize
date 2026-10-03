package tests

import (
	"context"
	"net/http"
	"net/url"
	"termorize/src/classification"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func categoryStats(t *testing.T, admin models.User) services.CategorizationStats {
	t.Helper()
	response := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/categorization/stats", nil)
	testkit.RequireStatus(t, response, http.StatusOK)
	var stats services.CategorizationStats
	testkit.DecodeJSON(t, response, &stats)
	return stats
}

func TestCategorizationRetryRunsInBackgroundAndStatsTrackProgress(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	unknown, noun := enums.PartOfSpeechUnknown, enums.PartOfSpeechNoun
	uncertain := categoryWord(t, "uncertain", &unknown)
	pending := categoryWord(t, "pending", nil)
	known := categoryWord(t, "known", &noun)
	started, release := make(chan struct{}), make(chan struct{})
	worker := classification.NewWorker(classification.WordStore{DB: db.DB}, func(ctx context.Context, term string, _ enums.Language) (enums.PartOfSpeech, error) {
		if term == "uncertain" {
			close(started)
			select {
			case <-release:
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		assert.NotEqual(t, "known", term)
		return enums.PartOfSpeechPhrase, nil
	})
	t.Cleanup(services.SetCategorizationWorker(worker))
	stats := categoryStats(t, admin)
	assert.Equal(t, int64(3), stats.Total)
	assert.Equal(t, int64(1), stats.Pending)
	assert.Equal(t, int64(1), stats.Unknown)
	assert.Equal(t, int64(1), stats.Categorized)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, "/api/admin/categorization/restart", nil), http.StatusAccepted)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, "/api/admin/categorization/restart", nil), http.StatusConflict)
	stats = categoryStats(t, admin)
	assert.Equal(t, int64(2), stats.Pending)
	assert.Zero(t, stats.Unknown)
	assert.True(t, stats.Worker.Active)
	require.NoError(t, db.DB.First(&known, "id = ?", known.ID).Error)
	assert.Equal(t, &noun, known.PartOfSpeech)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); worker.Run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("retry did not start")
	}
	assert.True(t, categoryStats(t, admin).Worker.Active)
	close(release)
	require.Eventually(t, func() bool { return !worker.Stats().Active }, 3*time.Second, 10*time.Millisecond)
	stats = categoryStats(t, admin)
	assert.Zero(t, stats.Pending)
	assert.Zero(t, stats.Unknown)
	assert.Equal(t, int64(3), stats.Categorized)
	assert.Equal(t, int64(2), stats.Worker.Processed)
	assert.Zero(t, stats.Worker.Failed)
	for _, word := range []models.Word{uncertain, pending} {
		var stored models.Word
		require.NoError(t, db.DB.First(&stored, "id = ?", word.ID).Error)
		assert.Equal(t, enums.PartOfSpeechPhrase, *stored.PartOfSpeech)
	}
}

func TestCategorizationRetryUnavailableDoesNotResetWords(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	unknown := enums.PartOfSpeechUnknown
	word := categoryWord(t, "uncertain", &unknown)
	t.Cleanup(services.SetCategorizationWorker(nil))
	testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, "/api/admin/categorization/restart", nil), http.StatusServiceUnavailable)
	assert.Nil(t, categoryStats(t, admin).Worker)
	require.NoError(t, db.DB.First(&word, "id = ?", word.ID).Error)
	assert.Equal(t, &unknown, word.PartOfSpeech)
}

func TestAdminCanSearchAndRecategorizeAnyWord(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	noun := enums.PartOfSpeechNoun
	word := categoryWord(t, "100%_Complete", &noun)
	categoryWord(t, "100 percent Complete", &noun)
	categoryWord(t, "pending", nil)
	for _, search := range []string{"complete", "%_"} {
		response := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/categorization/words?page_size=1&search="+url.QueryEscape(search), nil)
		testkit.RequireStatus(t, response, http.StatusOK)
		var result services.UnknownWordsResponse
		testkit.DecodeJSON(t, response, &result)
		if search == "%_" {
			assert.Equal(t, int64(1), result.Pagination.Total)
			assert.Equal(t, word.ID, result.Data[0].ID)
		} else {
			assert.Equal(t, int64(2), result.Pagination.Total)
			assert.Equal(t, 2, result.Pagination.TotalPages)
		}
		require.Len(t, result.Data, 1)
	}
	path := "/api/admin/words/" + word.ID.String() + "/part-of-speech"
	testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPut, path, map[string]any{"part_of_speech": "adjective"}), http.StatusOK)
	require.NoError(t, db.DB.First(&word, "id = ?", word.ID).Error)
	assert.Equal(t, enums.PartOfSpeechAdjective, *word.PartOfSpeech)
}

func TestClassificationCanSaveAfterUnknownIsReset(t *testing.T) {
	testkit.Truncate(t)
	word := categoryWord(t, "word", nil)
	store := classification.WordStore{DB: db.DB}
	stale, err := store.Load(context.Background(), word.ID)
	require.NoError(t, err)
	_, err = services.SetWordPartOfSpeech(context.Background(), word.ID, enums.PartOfSpeechUnknown)
	require.NoError(t, err)
	require.NoError(t, store.Save(context.Background(), *stale, enums.PartOfSpeechNoun))
	require.NoError(t, db.DB.First(&word, "id = ?", word.ID).Error)
	require.NotNil(t, word.PartOfSpeech)
	assert.Equal(t, enums.PartOfSpeechUnknown, *word.PartOfSpeech)
	require.NoError(t, store.ResetUnknown(context.Background()))
	require.NoError(t, db.DB.First(&word, "id = ?", word.ID).Error)
	assert.Nil(t, word.PartOfSpeech)
	require.NoError(t, store.Save(context.Background(), *stale, enums.PartOfSpeechVerb))
	require.NoError(t, db.DB.First(&word, "id = ?", word.ID).Error)
	assert.Equal(t, enums.PartOfSpeechVerb, *word.PartOfSpeech)
}
