package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"net/http"
	"sync"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/telegram"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"
	"testing"
	"time"
)

// Contract: opt-in, saved main language, local 11:00 including DST and fractional
// offsets, one delivery per local date across workers, retry after API failure.
func TestDailyIdiomDelivery(t *testing.T) {
	for _, zone := range []string{"Europe/Rome", "America/New_York", "Asia/Kathmandu", "Pacific/Kiritimati", "Pacific/Pago_Pago"} {
		for _, month := range []time.Month{time.January, time.July} {
			t.Run(fmt.Sprintf("%s/%d", zone, month), func(t *testing.T) {
				testkit.Truncate(t)
				tg := testkit.MockTelegramAPI(t)
				location, err := time.LoadLocation(zone)
				require.NoError(t, err)
				now := time.Date(2026, month, 15, 11, 0, 0, 0, location).UTC()
				user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{TimeZone: zone, MainLearningLanguage: enums.LanguageIt,
					Telegram: models.UserTelegramSettings{BotEnabled: true, DailyIdiomEnabled: true}}))
				word := seedDailyIdiomWord(t, "rompere il ghiaccio", enums.LanguageIt)
				testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{TimeZone: zone, Telegram: models.UserTelegramSettings{BotEnabled: true}}))
				testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{TimeZone: zone, Telegram: models.UserTelegramSettings{DailyIdiomEnabled: true}}))
				require.NoError(t, services.DeliverDailyIdioms(context.Background(), now.Add(-time.Minute), telegram.SendDailyIdiom))
				assert.Zero(t, tg.Count("sendMessage"))
				tg.FailNext("sendMessage")
				require.Error(t, services.DeliverDailyIdioms(context.Background(), now, telegram.SendDailyIdiom))
				var count int64
				require.NoError(t, db.DB.Table("daily_idiom_deliveries").Count(&count).Error)
				assert.Zero(t, count)
				var wg sync.WaitGroup
				errs := make(chan error, 3)
				for range 3 {
					wg.Add(1)
					go func() {
						defer wg.Done()
						errs <- services.DeliverDailyIdioms(context.Background(), now, telegram.SendDailyIdiom)
					}()
				}
				wg.Wait()
				close(errs)
				for err := range errs {
					require.NoError(t, err)
				}
				require.Equal(t, 2, tg.Count("sendMessage")) // failed attempt, successful retry
				var sent struct {
					ChatID      int64  `json:"chat_id"`
					Text        string `json:"text"`
					ReplyMarkup struct {
						InlineKeyboard [][]struct {
							Text string `json:"text"`
							Data string `json:"callback_data"`
						} `json:"inline_keyboard"`
					} `json:"reply_markup"`
				}
				require.NoError(t, json.Unmarshal(tg.RequestsFor("sendMessage")[1].Body, &sent))
				assert.Equal(t, user.TelegramID, sent.ChatID)
				assert.Equal(t, "Идиома дня\n\n🇮🇹 "+word.Word, sent.Text)
				daily, err := services.GetDailyIdiom(context.Background(), user.ID, now)
				require.NoError(t, err)
				require.Len(t, sent.ReplyMarkup.InlineKeyboard, 1)
				assert.Equal(t, "idiom:add:"+daily.Idiom.ID.String(), sent.ReplyMarkup.InlineKeyboard[0][0].Data)
				require.NoError(t, services.DeliverDailyIdioms(context.Background(), now.Add(time.Minute), telegram.SendDailyIdiom))
				assert.Equal(t, 2, tg.Count("sendMessage"))
				user.Settings.Telegram.DailyIdiomEnabled = false
				require.NoError(t, db.DB.Model(&user).Update("settings", user.Settings).Error)
				require.NoError(t, services.DeliverDailyIdioms(context.Background(), now.AddDate(0, 0, 1), telegram.SendDailyIdiom))
				assert.Equal(t, 2, tg.Count("sendMessage"))
			})
		}
	}
}

func TestDailyIdiomSettingDefaultsAndCanBeToggled(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	var loaded models.User
	require.NoError(t, db.DB.First(&loaded, user.ID).Error)
	assert.False(t, loaded.Settings.Telegram.DailyIdiomEnabled)
	loaded.Settings.TimeZone = "Europe/Rome"
	loaded.Settings.Telegram.DailyQuestionsSchedule = []models.UserTelegramQuestionsScheduleItem{}
	for _, enabled := range []bool{true, false} {
		loaded.Settings.Telegram.DailyIdiomEnabled = enabled
		rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", loaded.Settings)
		testkit.RequireStatus(t, rec, http.StatusOK)
		require.NoError(t, db.DB.First(&loaded, user.ID).Error)
		assert.Equal(t, enabled, loaded.Settings.Telegram.DailyIdiomEnabled)
	}
}

func denyIdiomGoogle(t *testing.T) {
	t.Helper()
	testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{TranslateFunc: func(string, string, string) (string, error) {
		t.Error("idiom reached Google Translate")
		return "", errors.New("forbidden")
	}, DetectFunc: func(string) (string, error) {
		t.Error("idiom reached Google detection")
		return "", errors.New("forbidden")
	}})
}

func TestTelegramWebhookIdiomAddTranslationThenSaveAndReplay(t *testing.T) {
	for _, source := range []enums.Language{enums.LanguageEn, enums.LanguageIt} {
		t.Run(string(source), func(t *testing.T) {
			testkit.Truncate(t)
			tg := testkit.MockTelegramAPI(t)
			denyIdiomGoogle(t)
			user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{MainLearningLanguage: enums.LanguageEn, TranslationSourceLanguage: source, TranslationTargetLanguage: enums.LanguageRu}))
			seedDailyIdiomWord(t, "break the ice", enums.LanguageEn)
			daily, err := services.GetDailyIdiom(context.Background(), user.ID, time.Now())
			require.NoError(t, err)
			calls := 0
			testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{TranslateIdiomFunc: func(ctx context.Context, idiom, from, to string) (string, error) {
				calls++
				assert.Equal(t, "break the ice", idiom)
				assert.Equal(t, "English", from)
				expected := "Russian"
				if source == enums.LanguageIt {
					expected = "Italian"
				}
				assert.Equal(t, expected, to)
				var count int64
				require.NoError(t, db.DB.Model(&models.Vocabulary{}).Count(&count).Error)
				assert.Zero(t, count, "translate before saving")
				return "rompere il ghiaccio", nil
			}})
			update := telegramMenuCallback(user.TelegramID, "idiom-callback", "unused")
			update["callback_query"].(map[string]any)["data"] = "idiom:add:" + daily.Idiom.ID.String()
			for range 2 {
				testkit.RequireStatus(t, telegramUpdate(t, update), http.StatusOK)
			}
			assert.Equal(t, 1, calls)
			var saved []models.Vocabulary
			require.NoError(t, db.DB.Preload("Translation.Original").Preload("Translation.Translation").Find(&saved).Error)
			require.Len(t, saved, 1)
			assert.Equal(t, user.ID, saved[0].UserID)
			assert.Equal(t, "break the ice", saved[0].Translation.Original.Word)
			assert.Equal(t, "rompere il ghiaccio", saved[0].Translation.Translation.Word)
			assert.Equal(t, services.IdiomTargetLanguage(user.Settings, enums.LanguageEn), saved[0].Translation.Translation.Language)
			assert.Equal(t, enums.TranslationSourceIdiomLLM, saved[0].Translation.Source)
			assert.Equal(t, 2, tg.Count("answerCallbackQuery"))
			assert.Zero(t, tg.Count("sendMessage"))
			require.Equal(t, 2, tg.Count("editMessageText"))
			for _, req := range tg.RequestsFor("editMessageText") {
				var body struct {
					Text        string `json:"text"`
					ChatID      int64  `json:"chat_id"`
					MessageID   int64  `json:"message_id"`
					ReplyMarkup struct {
						Keyboard []any `json:"inline_keyboard"`
					} `json:"reply_markup"`
				}
				require.NoError(t, json.Unmarshal(req.Body, &body))
				assert.Equal(t, user.TelegramID, body.ChatID)
				assert.Equal(t, int64(77), body.MessageID)
				assert.Contains(t, body.Text, "break the ice")
				assert.Contains(t, body.Text, "rompere il ghiaccio")
				assert.Empty(t, body.ReplyMarkup.Keyboard)
			}
		})
	}
}

func TestTelegramWebhookIdiomFailuresDoNotConfirmOrSave(t *testing.T) {
	for _, failure := range []string{"translation", "empty", "save"} {
		t.Run(failure, func(t *testing.T) {
			testkit.Truncate(t)
			tg := testkit.MockTelegramAPI(t)
			denyIdiomGoogle(t)
			user := testkit.CreateUser(t)
			seedDailyIdiomWord(t, "break the ice", enums.LanguageEn)
			daily, err := services.GetDailyIdiom(context.Background(), user.ID, time.Now())
			require.NoError(t, err)
			testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{TranslateIdiomFunc: func(context.Context, string, string, string) (string, error) {
				if failure == "translation" {
					return "", errors.New("provider failed")
				}
				if failure == "empty" {
					return " ", nil
				}
				return "растопить лёд", nil
			}})
			if failure == "save" {
				require.NoError(t, db.DB.Callback().Create().Before("gorm:create").Register("idiom-save-failure", func(tx *gorm.DB) {
					if tx.Statement.Table == "vocabulary" {
						tx.AddError(errors.New("write failed"))
					}
				}))
				t.Cleanup(func() { db.DB.Callback().Create().Remove("idiom-save-failure") })
			}
			update := telegramMenuCallback(user.TelegramID, "failed-idiom", "unused")
			update["callback_query"].(map[string]any)["data"] = "idiom:add:" + daily.Idiom.ID.String()
			testkit.RequireStatus(t, telegramUpdate(t, update), http.StatusOK)
			assert.Equal(t, 1, tg.Count("answerCallbackQuery"))
			assert.Zero(t, tg.Count("editMessageText"))
			assert.Equal(t, 1, tg.Count("sendMessage"))
			var count int64
			require.NoError(t, db.DB.Model(&models.Vocabulary{}).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}
