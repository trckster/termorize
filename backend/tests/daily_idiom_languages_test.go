package tests

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"termorize/src/controllers"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/integrations/telegram"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"
	"testing"
	"time"
)

// Contract: compare coverage to the actual application registry, not a second
// hand-maintained language list. Exercise the same selection in web and Telegram.
func TestDailyIdiomEverySupportedLanguage(t *testing.T) {
	testkit.Truncate(t)
	migration, err := os.ReadFile("src/data/migrations/0028_seed_daily_idioms_all_languages.sql")
	require.NoError(t, err)
	for range 2 {
		require.NoError(t, db.DB.Exec(string(migration)).Error)
	}
	var languages []string
	require.NoError(t, db.DB.Model(&models.Word{}).Distinct("language").Where("type = ?", enums.TypeIdiom).Pluck("language", &languages).Error)
	assert.ElementsMatch(t, enums.AllLanguages(), languages)
	now := time.Date(2026, 9, 24, 11, 0, 0, 0, time.UTC)
	for _, language := range enums.AllLanguageValues() {
		t.Run(string(language), func(t *testing.T) {
			tg := testkit.MockTelegramAPI(t)
			target := enums.LanguageEn
			if language == target {
				target = enums.LanguageRu
			}
			settings := models.UserSettings{SystemLanguage: enums.LanguageEn, MainLearningLanguage: language, TranslationSourceLanguage: language, TranslationTargetLanguage: target, TimeZone: "UTC",
				Telegram: models.UserTelegramSettings{BotEnabled: true, DailyIdiomEnabled: true}}
			user := testkit.CreateUser(t, testkit.WithSettings(settings))
			other := testkit.CreateUser(t, testkit.WithSettings(settings))
			var daily services.DailyIdiomResponse
			rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/daily-idiom", nil)
			testkit.RequireStatus(t, rec, 200)
			testkit.DecodeJSON(t, rec, &daily)
			require.NotNil(t, daily.Idiom)
			assert.Equal(t, language, daily.Language)
			previousWord := ""
			for range 12 {
				next, err := services.GetDailyIdiom(context.Background(), user.ID, now)
				require.NoError(t, err)
				require.NotNil(t, next.Idiom)
				assert.NotEqual(t, previousWord, next.Idiom.Word)
				previousWord = next.Idiom.Word
				shared, err := services.GetDailyIdiom(context.Background(), other.ID, now)
				require.NoError(t, err)
				assert.Equal(t, next, shared)
				now = now.AddDate(0, 0, 1)
			}
			// Description generation supports each language independently of translation.
			testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{TranslateFunc: func(string, string, string) (string, error) { t.Error("Google translation called"); return "", nil }, DetectFunc: func(string) (string, error) { return string(language), nil }})
			translationCalls := 0
			testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
				GenerateIdiomDescriptionFunc: func(_ context.Context, idiom, from string) (*openrouter.GeneratedDescription, error) {
					assert.Equal(t, daily.Idiom.Word, idiom)
					assert.Equal(t, language.DisplayName(), from)
					return &openrouter.GeneratedDescription{Description: "a figurative meaning"}, nil
				},
				TranslateIdiomFunc: func(_ context.Context, idiom, from, to string) (string, error) {
					translationCalls++
					assert.Equal(t, language.DisplayName(), from)
					assert.Equal(t, target.DisplayName(), to)
					return "natural equivalent", nil
				},
			})
			rec = testkit.AuthedRequest(t, user, http.MethodGet, "/api/daily-idiom/"+daily.Idiom.ID.String()+"/description", nil)
			testkit.RequireStatus(t, rec, 200)
			denyIdiomGoogle(t)
			// Use today's real selected date for the fixed 11:00 dispatch.
			dispatch, err := time.Parse(time.DateOnly, daily.Date)
			require.NoError(t, err)
			dispatch = dispatch.Add(11 * time.Hour)
			require.NoError(t, services.DeliverDailyIdioms(context.Background(), dispatch, telegram.SendDailyIdiom))
			require.Equal(t, 2, tg.Count("sendMessage"))
			assert.Contains(t, string(tg.RequestsFor("sendMessage")[0].Body), daily.Idiom.ID.String())
			var translated controllers.TranslateResponse
			rec = testkit.AuthedRequest(t, user, http.MethodPost, "/api/daily-idiom/"+daily.Idiom.ID.String()+"/translate", map[string]string{"to_language": string(target)})
			testkit.RequireStatus(t, rec, 200)
			testkit.DecodeJSON(t, rec, &translated)
			rec = testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary/translation", map[string]any{"translation_id": translated.ID})
			testkit.RequireStatus(t, rec, 201)
			update := telegramMenuCallback(other.TelegramID, "all-languages", "unused")
			update["callback_query"].(map[string]any)["data"] = "idiom:add:" + daily.Idiom.ID.String()
			testkit.RequireStatus(t, telegramUpdate(t, update), 200)
			assert.Equal(t, 1, translationCalls)
			var saved []models.Vocabulary
			require.NoError(t, db.DB.Preload("Translation.Original").Preload("Translation.Translation").Where("user_id IN ?", []uint{user.ID, other.ID}).Find(&saved).Error)
			require.Len(t, saved, 2)
			for _, item := range saved {
				assert.Equal(t, daily.Idiom.Word, item.Translation.Original.Word)
				assert.Equal(t, language, item.Translation.Original.Language)
				assert.Equal(t, target, item.Translation.Translation.Language)
				assert.Equal(t, translated.Translation, item.Translation.Translation.Word)
			}
			require.NoError(t, db.DB.Model(&models.User{}).Where("id IN ?", []uint{user.ID, other.ID}).Update("settings", models.UserSettings{}).Error)
		})
	}
	user := testkit.CreateUser(t)
	daily, err := services.GetDailyIdiom(context.Background(), user.ID, time.Now())
	require.NoError(t, err)
	denyIdiomGoogle(t)
	for _, target := range enums.AllLanguageValues() {
		if target == enums.LanguageEn {
			continue
		}
		testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{TranslateIdiomFunc: func(_ context.Context, _, _, to string) (string, error) {
			assert.Equal(t, target.DisplayName(), to)
			return "natural equivalent", nil
		}})
		result, err := services.TranslateDailyIdiom(context.Background(), daily.Idiom.ID, target)
		require.NoError(t, err)
		assert.Equal(t, enums.TranslationSourceIdiomLLM, result.Source)
	}
}
