package tests

import (
	"errors"
	"net/http"
	"sync/atomic"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWordPronunciationRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/words/"+uuid.NewString()+"/pronunciation", nil)

	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestWordPronunciationCacheMissGeneratesStoresAndReturnsMP3(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	word := createPronunciationWord(t, "buongiorno")
	audio := []byte{0xff, 0xfb, 0x01, 0x02, 0x03}
	generated := 0
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{
		Usage: openrouter.Usage{GenerationID: "gen-tts", Model: "tts/model", Cost: 0.002, TotalTokens: 10},
		GenerateFunc: func(input string) ([]byte, error) {
			generated++
			assert.Equal(t, "Synthesize speech in Italian. Speak only the transcript exactly as written.\nTranscript: \"buongiorno\"", input)
			return audio, nil
		}})

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/words/"+word.ID.String()+"/pronunciation", nil)

	testkit.RequireStatus(t, rec, http.StatusOK)
	assert.Equal(t, models.PronunciationMIMETypeMP3, rec.Header().Get("Content-Type"))
	assert.Equal(t, `inline; filename="pronunciation.mp3"`, rec.Header().Get("Content-Disposition"))
	assert.Equal(t, "private, max-age=86400", rec.Header().Get("Cache-Control"))
	assert.Equal(t, audio, rec.Body.Bytes())
	assert.Equal(t, 1, generated)

	var stored models.WordPronunciation
	require.NoError(t, db.DB.Where("word_id = ?", word.ID).First(&stored).Error)
	assert.Equal(t, audio, stored.Audio)
	assert.Equal(t, config.GetOpenRouterTTSModel(), stored.Model)
	assert.Equal(t, config.GetOpenRouterTTSVoice(), stored.Voice)
	var usage models.OpenRouterUsage
	require.NoError(t, db.DB.Where("user_id = ?", user.ID).First(&usage).Error)
	assert.Equal(t, "gen-tts", *usage.GenerationID)
	assert.InDelta(t, 0.002, usage.Cost, 0.0000000001)
}

func TestWordPronunciationFallsBackToSecondaryModel(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	word := createPronunciationWord(t, "radio")
	audio := []byte("fallback-mp3")
	generated := 0
	var inputs []string
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(input string) ([]byte, error) {
		generated++
		inputs = append(inputs, input)
		if generated == 1 {
			return nil, errors.New("empty audio")
		}
		return audio, nil
	}})

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/words/"+word.ID.String()+"/pronunciation", nil)

	testkit.RequireStatus(t, rec, http.StatusOK)
	assert.Equal(t, audio, rec.Body.Bytes())
	assert.Equal(t, 2, generated)
	assert.Equal(t, []string{
		"Synthesize speech in Italian. Speak only the transcript exactly as written.\nTranscript: \"radio\"",
		"radio",
	}, inputs)

	var stored models.WordPronunciation
	require.NoError(t, db.DB.Where("word_id = ?", word.ID).First(&stored).Error)
	fallback := config.GetOpenRouterTTSConfigs(string(word.Language))[1]
	assert.Equal(t, fallback.Model, stored.Model)
	assert.Equal(t, fallback.Voice, stored.Voice)
}

func TestWordPronunciationCacheHitReturnsStoredAudioWithoutGenerating(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	word := createPronunciationWord(t, "ciao")
	audio := []byte("stored-mp3")
	_, err := services.StoreWordPronunciation(
		word.ID,
		config.GetOpenRouterTTSModel(),
		config.GetOpenRouterTTSVoice(),
		audio,
	)
	require.NoError(t, err)
	require.NoError(t, db.DB.Create(&models.OpenRouterUsage{
		UserID: user.ID, Model: "test/model", Cost: 1, CreatedAt: time.Now().UTC(),
	}).Error)
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(string) ([]byte, error) {
		return nil, errors.New("unexpected TTS generation")
	}})

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/words/"+word.ID.String()+"/pronunciation", nil)

	testkit.RequireStatus(t, rec, http.StatusOK)
	assert.Equal(t, audio, rec.Body.Bytes())
}

func TestWordPronunciationRejectsGenerationAtSpendingLimit(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	word := createPronunciationWord(t, "limite")
	createdAt := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, db.DB.Create(&models.OpenRouterUsage{
		UserID: user.ID, Model: "test/model", Cost: 1, CreatedAt: createdAt,
	}).Error)
	called := false
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(string) ([]byte, error) {
		called = true
		return []byte("unexpected"), nil
	}})

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/words/"+word.ID.String()+"/pronunciation", nil)

	testkit.RequireStatus(t, rec, http.StatusTooManyRequests)
	assert.False(t, called)
	var body struct {
		Limit   float64   `json:"limit"`
		RetryAt time.Time `json:"retry_at"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, 1.0, body.Limit)
	assert.WithinDuration(t, createdAt.Add(24*time.Hour), body.RetryAt, time.Second)
}

func TestWordPronunciationCoalescesConcurrentGeneration(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	word := createPronunciationWord(t, "arrivederci")
	audio := []byte("shared-mp3")
	started := make(chan struct{})
	release := make(chan struct{})
	var generated atomic.Int32
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(string) ([]byte, error) {
		if generated.Add(1) == 1 {
			close(started)
		}
		<-release
		return audio, nil
	}})

	type result struct {
		pronunciation *models.WordPronunciation
		err           error
	}
	results := make(chan result, 2)
	request := func() {
		pronunciation, err := services.GetOrCreateWordPronunciation(user.ID, word.ID)
		results <- result{pronunciation: pronunciation, err: err}
	}

	go request()
	<-started
	go request()
	time.Sleep(20 * time.Millisecond)
	close(release)

	for range 2 {
		got := <-results
		require.NoError(t, got.err)
		assert.Equal(t, audio, got.pronunciation.Audio)
	}
	assert.Equal(t, int32(1), generated.Load())
}

func TestWordPronunciationRejectsInvalidAndUnknownWordIDs(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)

	invalid := testkit.AuthedRequest(t, user, http.MethodGet, "/api/words/not-a-uuid/pronunciation", nil)
	testkit.RequireStatus(t, invalid, http.StatusBadRequest)

	missing := testkit.AuthedRequest(t, user, http.MethodGet, "/api/words/"+uuid.NewString()+"/pronunciation", nil)
	testkit.RequireStatus(t, missing, http.StatusNotFound)
}

func createPronunciationWord(t *testing.T, value string) models.Word {
	t.Helper()

	word := models.Word{Word: value, Language: enums.LanguageIt}
	require.NoError(t, db.DB.Create(&word).Error)
	return word
}
