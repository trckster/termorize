package tests

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"sync"
	"termorize/src/controllers"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/telegram"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"
	"testing"
	"time"
)

// Contract: auth, selection/target validation, isolated LLM cache, no Google
// fallback, and exact preservation through the existing Vocabulary endpoint.
func TestTranslateDailyIdiomEndpointAndVocabulary(t *testing.T) {
	testkit.Truncate(t)
	denyIdiomGoogle(t)
	user := testkit.CreateUser(t)
	word := seedDailyIdiomWord(t, "break the ice", enums.LanguageEn)
	daily, err := services.GetDailyIdiom(context.Background(), user.ID, time.Now())
	require.NoError(t, err)
	path := "/api/daily-idiom/" + daily.Idiom.ID.String() + "/translate"
	literal, err := services.GetOrCreateWord(db.DB, "сломать лёд", enums.LanguageRu)
	require.NoError(t, err)
	require.NoError(t, db.DB.Create(&models.Translation{OriginalID: word.ID, TranslationID: literal.ID, Source: enums.TranslationSourceGoogle}).Error)
	calls := 0
	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{TranslateIdiomFunc: func(context.Context, string, string, string) (string, error) {
		calls++
		return "растопить лёд", nil
	}})
	body := map[string]string{"to_language": "ru"}
	testkit.RequireStatus(t, testkit.Request(t, http.MethodPost, path, body), http.StatusUnauthorized)
	for _, tc := range []struct {
		path   string
		body   any
		status int
	}{
		{"/api/daily-idiom/bad/translate", body, 400},
		{"/api/daily-idiom/" + uuid.NewString() + "/translate", body, 404},
		{path, map[string]string{}, 400},
		{path, map[string]string{"to_language": "xx"}, 400},
		{path, map[string]string{"to_language": "en"}, 400},
	} {
		testkit.RequireStatus(t, testkit.AuthedRequest(t, user, http.MethodPost, tc.path, tc.body), tc.status)
	}
	assert.Zero(t, calls)
	var first controllers.TranslateResponse
	for range 2 {
		rec := testkit.AuthedRequest(t, user, http.MethodPost, path, body)
		testkit.RequireStatus(t, rec, 200)
		var result controllers.TranslateResponse
		testkit.DecodeJSON(t, rec, &result)
		assert.Equal(t, "растопить лёд", result.Translation)
		assert.Equal(t, enums.TranslationSourceIdiomLLM, result.Source)
		if first.ID != uuid.Nil {
			assert.Equal(t, first, result)
		}
		first = result
	}
	assert.Equal(t, 1, calls)
	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary/translation", map[string]any{"translation_id": first.ID})
	testkit.RequireStatus(t, rec, http.StatusCreated)
	var saved models.Vocabulary
	require.NoError(t, db.DB.Preload("Translation.Original").Preload("Translation.Translation").Where("user_id = ?", user.ID).First(&saved).Error)
	assert.Equal(t, first.ID, saved.TranslationID)
	assert.Equal(t, word.Word, saved.Translation.Original.Word)
	assert.Equal(t, enums.LanguageEn, saved.Translation.Original.Language)
	assert.Equal(t, first.Translation, saved.Translation.Translation.Word)
	assert.Equal(t, enums.LanguageRu, saved.Translation.Translation.Language)
	assert.Equal(t, 1, calls)
	other := testkit.CreateUser(t)
	_, err = services.CreateVocabulary(other.ID, services.CreateVocabularyRequest{Original: word.Word, Translation: first.Translation, OriginalLanguage: enums.LanguageEn, TranslationLanguage: enums.LanguageRu})
	require.NoError(t, err)
	_, err = services.CreateVocabularyByTranslation(other.ID, first.ID)
	require.ErrorIs(t, err, services.ErrVocabularyAlreadyExists)
	third := testkit.CreateUser(t)
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := services.CreateVocabularyByTranslation(third.ID, first.ID)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	success := 0
	for err := range errs {
		if err == nil {
			success++
		} else {
			require.ErrorIs(t, err, services.ErrVocabularyAlreadyExists)
		}
	}
	assert.Equal(t, 1, success)
}

func TestTranslateDailyIdiomFailureDoesNotFallBack(t *testing.T) {
	testkit.Truncate(t)
	denyIdiomGoogle(t)
	user := testkit.CreateUser(t)
	seedDailyIdiomWord(t, "break the ice", enums.LanguageEn)
	daily, err := services.GetDailyIdiom(context.Background(), user.ID, time.Now())
	require.NoError(t, err)
	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{TranslateIdiomFunc: func(context.Context, string, string, string) (string, error) { return "", errors.New("unavailable") }})
	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/daily-idiom/"+daily.Idiom.ID.String()+"/translate", map[string]string{"to_language": "ru"})
	testkit.RequireStatus(t, rec, 503)
	var count int64
	require.NoError(t, db.DB.Model(&models.Translation{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestIdiomReplayDoesNotRestoreDuplicateCustomPair(t *testing.T) {
	testkit.Truncate(t)
	denyIdiomGoogle(t)
	tg := testkit.MockTelegramAPI(t)
	user := testkit.CreateUser(t)
	seedDailyIdiomWord(t, "break the ice", enums.LanguageEn)
	daily, err := services.GetDailyIdiom(context.Background(), user.ID, time.Now())
	require.NoError(t, err)
	translated, err := services.TranslateDailyIdiom(context.Background(), daily.Idiom.ID, enums.LanguageRu)
	require.NoError(t, err)
	saved, err := services.CreateVocabularyByTranslation(user.ID, translated.TranslationID)
	require.NoError(t, err)
	require.NoError(t, services.DeleteVocabulary(user.ID, saved.ID))
	custom, err := services.CreateVocabulary(user.ID, services.CreateVocabularyRequest{Original: translated.SourceWord, Translation: translated.TranslatedWord, OriginalLanguage: enums.LanguageEn, TranslationLanguage: enums.LanguageRu})
	require.NoError(t, err)
	update := telegramMenuCallback(user.TelegramID, "restore-idiom", "unused")
	update["callback_query"].(map[string]any)["data"] = "idiom:add:" + daily.Idiom.ID.String()
	testkit.RequireStatus(t, telegramUpdate(t, update), http.StatusOK)
	var active []models.Vocabulary
	require.NoError(t, db.DB.Where("user_id = ? AND deleted_at IS NULL", user.ID).Find(&active).Error)
	require.Len(t, active, 1)
	assert.Equal(t, custom.ID, active[0].ID)
	require.Equal(t, 1, tg.Count("editMessageText"))
	assert.Contains(t, string(tg.RequestsFor("editMessageText")[0].Body), telegram.GetBotTexts(enums.LanguageRu).AddVocabularyExists)
	require.NoError(t, services.DeleteVocabulary(user.ID, custom.ID))
	restored, err := services.CreateVocabularyByTranslation(user.ID, translated.TranslationID)
	require.NoError(t, err)
	assert.Equal(t, saved.ID, restored.ID)
}
