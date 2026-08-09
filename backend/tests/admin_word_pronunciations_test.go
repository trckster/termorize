package tests

import (
	"errors"
	"net/http"
	"net/url"
	"termorize/src/config"
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

func TestAdminWordPronunciationsRequiresAuthentication(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/admin/word-pronunciations", nil)
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestAdminWordPronunciationsRejectsNonAdmins(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/admin/word-pronunciations", nil)
	testkit.RequireStatus(t, rec, http.StatusForbidden)
}

func TestAdminWordPronunciationsSupportsSearchAndPagination(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	older := seedAdminPronunciation(t, "buongiorno", enums.LanguageIt, "model-old", "voice-old", []byte("old"))
	newer := seedAdminPronunciation(t, "Buonasera", enums.LanguageIt, "model-new", "voice-new", []byte("new-audio"))
	seedAdminPronunciation(t, "hello", enums.LanguageEn, "model-en", "voice-en", []byte("hello"))
	referenceTime := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	require.NoError(t, db.DB.Model(&models.WordPronunciation{}).Where("id = ?", older.ID).Update("created_at", referenceTime).Error)
	require.NoError(t, db.DB.Model(&models.WordPronunciation{}).Where("id = ?", newer.ID).Updates(map[string]any{
		"created_at":       referenceTime.Add(time.Hour),
		"telegram_file_id": "telegram-file",
	}).Error)

	path := "/api/admin/word-pronunciations?page=1&page_size=1&search=" + url.QueryEscape("BUON")
	rec := testkit.AuthedRequest(t, admin, http.MethodGet, path, nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var response services.AdminWordPronunciationsResponse
	testkit.DecodeJSON(t, rec, &response)
	assert.Equal(t, int64(2), response.Pagination.Total)
	assert.Equal(t, 2, response.Pagination.TotalPages)
	assert.Equal(t, 1, response.Pagination.Page)
	assert.Equal(t, 1, response.Pagination.PageSize)
	require.Len(t, response.Data, 1)
	assert.Equal(t, newer.ID, response.Data[0].ID)
	assert.Equal(t, newer.WordID, response.Data[0].WordID)
	assert.Equal(t, "Buonasera", response.Data[0].Word)
	assert.Equal(t, enums.LanguageIt, response.Data[0].Language)
	assert.Equal(t, "model-new", response.Data[0].Model)
	assert.Equal(t, "voice-new", response.Data[0].Voice)
	assert.Equal(t, int64(len("new-audio")), response.Data[0].SizeBytes)
	assert.True(t, response.Data[0].HasTelegramFile)
}

func TestAdminWordPronunciationAudioIsAdminOnly(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	user := testkit.CreateUser(t)
	pronunciation := seedAdminPronunciation(t, "ciao", enums.LanguageIt, "model", "voice", []byte("audio-bytes"))
	path := "/api/admin/word-pronunciations/" + pronunciation.ID.String() + "/audio"

	forbidden := testkit.AuthedRequest(t, user, http.MethodGet, path, nil)
	testkit.RequireStatus(t, forbidden, http.StatusForbidden)

	rec := testkit.AuthedRequest(t, admin, http.MethodGet, path, nil)
	testkit.RequireStatus(t, rec, http.StatusOK)
	assert.Equal(t, models.PronunciationMIMETypeMP3, rec.Header().Get("Content-Type"))
	assert.Equal(t, []byte("audio-bytes"), rec.Body.Bytes())
}

func TestAdminWordPronunciationRegeneratesAndReplacesAudio(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	pronunciation := seedAdminPronunciation(t, "buongiorno", enums.LanguageIt, "old-model", "old-voice", []byte("old-audio"))
	require.NoError(t, db.DB.Model(&pronunciation).Update("telegram_file_id", "stale-telegram-file").Error)

	generatedAudio := []byte("new-audio")
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(input string) ([]byte, error) {
		assert.Equal(t, "Synthesize speech in Italian. Speak only the transcript exactly as written.\nTranscript: \"buongiorno\"", input)
		return generatedAudio, nil
	}})

	rec := testkit.AuthedRequest(t, admin, http.MethodDelete, "/api/admin/word-pronunciations/"+pronunciation.ID.String(), nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var replacement services.AdminWordPronunciation
	testkit.DecodeJSON(t, rec, &replacement)
	assert.NotEqual(t, pronunciation.ID, replacement.ID)
	assert.Equal(t, pronunciation.WordID, replacement.WordID)
	assert.Equal(t, config.GetOpenRouterTTSModel(), replacement.Model)
	assert.Equal(t, config.GetOpenRouterTTSVoice(), replacement.Voice)
	assert.False(t, replacement.HasTelegramFile)
	assert.Equal(t, int64(len(generatedAudio)), replacement.SizeBytes)

	var removed models.WordPronunciation
	assert.ErrorIs(t, db.DB.First(&removed, "id = ?", pronunciation.ID).Error, gorm.ErrRecordNotFound)
	var stored models.WordPronunciation
	require.NoError(t, db.DB.First(&stored, "id = ?", replacement.ID).Error)
	assert.Equal(t, generatedAudio, stored.Audio)
	assert.Nil(t, stored.TelegramFileID)
}

func TestAdminWordPronunciationFailedRegenerationKeepsExistingAudio(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	pronunciation := seedAdminPronunciation(t, "ciao", enums.LanguageIt, "old-model", "old-voice", []byte("old-audio"))
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(string) ([]byte, error) {
		return nil, errors.New("provider unavailable")
	}})

	rec := testkit.AuthedRequest(t, admin, http.MethodDelete, "/api/admin/word-pronunciations/"+pronunciation.ID.String(), nil)
	testkit.RequireStatus(t, rec, http.StatusInternalServerError)

	var stored models.WordPronunciation
	require.NoError(t, db.DB.First(&stored, "id = ?", pronunciation.ID).Error)
	assert.Equal(t, []byte("old-audio"), stored.Audio)
}

func TestAdminWordPronunciationRegenerationRejectsNonAdmins(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	pronunciation := seedAdminPronunciation(t, "ciao", enums.LanguageIt, "model", "voice", []byte("audio"))

	rec := testkit.AuthedRequest(t, user, http.MethodDelete, "/api/admin/word-pronunciations/"+pronunciation.ID.String(), nil)
	testkit.RequireStatus(t, rec, http.StatusForbidden)
}

func seedAdminPronunciation(
	t *testing.T,
	wordText string,
	language enums.Language,
	model string,
	voice string,
	audio []byte,
) models.WordPronunciation {
	t.Helper()
	word := models.Word{Word: wordText, Language: language}
	require.NoError(t, db.DB.Create(&word).Error)
	pronunciation := models.WordPronunciation{
		ID:       uuid.New(),
		WordID:   word.ID,
		Model:    model,
		Voice:    voice,
		Audio:    audio,
		MIMEType: models.PronunciationMIMETypeMP3,
	}
	require.NoError(t, db.DB.Create(&pronunciation).Error)
	return pronunciation
}
