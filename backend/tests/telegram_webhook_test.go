package tests

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/telegram"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const telegramWebhookPath = "/api/telegram/webhook"
const telegramSecretHeader = "X-Telegram-Bot-Api-Secret-Token"

// telegramRequest issues a raw-body request to the webhook endpoint with the given
// secret-token header. The body is sent verbatim (the handler reads the raw body),
// so callers control exactly what JSON — or non-JSON — is delivered.
func telegramRequest(t *testing.T, rawBody, secret string) *httptest.ResponseRecorder {
	t.Helper()

	headers := make(http.Header)
	if secret != "" {
		headers.Set(telegramSecretHeader, secret)
	}

	return testkit.RawRequest(t, http.MethodPost, telegramWebhookPath, rawBody, headers)
}

// telegramUpdate marshals an update payload and posts it with the correct secret.
func telegramUpdate(t *testing.T, update map[string]any) *httptest.ResponseRecorder {
	t.Helper()

	encoded, err := json.Marshal(update)
	require.NoError(t, err)

	return telegramRequest(t, string(encoded), telegram.BuildWebhookSecret())
}

func telegramCompactUUID(id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString(id[:])
}

// telegramPrivateMessage builds a realistic private-chat message update for the
// given telegram id and text.
func telegramPrivateMessage(telegramID int64, text string) map[string]any {
	return map[string]any{
		"update_id": 1,
		"message": map[string]any{
			"message_id": 10,
			"date":       1700000000,
			"text":       text,
			"chat": map[string]any{
				"id":         telegramID,
				"first_name": "Ada",
				"username":   "ada",
				"type":       "private",
			},
			"from": map[string]any{
				"id":         telegramID,
				"is_bot":     false,
				"first_name": "Ada",
				"username":   "ada",
			},
		},
	}
}

// -----------------------------------------------------------------------------
// Middleware auth
// -----------------------------------------------------------------------------

func TestTelegramWebhookAuthenticatesSecretToken(t *testing.T) {
	tests := []struct {
		name       string
		secret     string
		wantStatus int
	}{
		{name: "missing", secret: "", wantStatus: http.StatusUnauthorized},
		{name: "incorrect", secret: "definitely-not-the-secret", wantStatus: http.StatusUnauthorized},
		{name: "valid", secret: telegram.BuildWebhookSecret(), wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.Truncate(t)
			tg := testkit.MockTelegramAPI(t)

			rec := telegramRequest(t, `{"update_id":1}`, tt.secret)
			testkit.RequireStatus(t, rec, tt.wantStatus)
			assert.Empty(t, tg.Requests(), "authentication checks must not call the Telegram API")
		})
	}
}

// -----------------------------------------------------------------------------
// Payload handling
// -----------------------------------------------------------------------------

func TestTelegramWebhookMalformedBody(t *testing.T) {
	testkit.Truncate(t)
	testkit.MockTelegramAPI(t)

	rec := telegramRequest(t, "this is not json", telegram.BuildWebhookSecret())
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid payload", body["error"])
}

func TestTelegramWebhookUnknownUpdateNoOp(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	// A well-formed update with none of message/callback_query/my_chat_member set.
	rec := telegramUpdate(t, map[string]any{"update_id": 42})
	testkit.RequireStatus(t, rec, http.StatusOK)
	assert.Empty(t, tg.Requests(), "no outbound telegram call expected for an empty update")
}

func TestTelegramWebhookIgnoredExerciseReportsDeletedVocabulary(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555099
	const exerciseMessageID int64 = 501
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusIgnored, vocabulary.ID)
	require.NoError(t, db.DB.Model(&models.Exercise{}).
		Where("id = ?", exercise.ID).
		Update("telegram_message_id", exerciseMessageID).Error)
	require.NoError(t, db.DB.Model(&models.ExerciseVocabulary{}).
		Where("exercise_id = ? AND vocabulary_id = ?", exercise.ID, vocabulary.ID).
		Updates(map[string]any{
			"result":        services.ExerciseVocabularyResultIgnored,
			"result_reason": services.ExerciseVocabularyResultReasonDeletedVocabulary,
		}).Error)

	update := telegramPrivateMessage(telegramID, "Hund")
	message := update["message"].(map[string]any)
	message["reply_to_message"] = map[string]any{
		"message_id": exerciseMessageID,
		"chat": map[string]any{
			"id":   telegramID,
			"type": "private",
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	requests := tg.RequestsFor("sendMessage")
	require.Len(t, requests, 1)
	assert.Contains(t, string(requests[0].Body), "нельзя выполнить")
	assert.NotContains(t, string(requests[0].Body), "устарело")
	require.Len(t, tg.RequestsFor("editMessageReplyMarkup"), 1)
}

func TestTelegramWebhookKeyboardCleanupFailureStillReportsDeletedVocabulary(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)
	tg.Fail("editMessageReplyMarkup")

	const telegramID int64 = 555098
	const exerciseMessageID int64 = 500
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusIgnored, vocabulary.ID)
	require.NoError(t, db.DB.Model(&models.Exercise{}).
		Where("id = ?", exercise.ID).
		Update("telegram_message_id", exerciseMessageID).Error)
	require.NoError(t, db.DB.Model(&models.ExerciseVocabulary{}).
		Where("exercise_id = ? AND vocabulary_id = ?", exercise.ID, vocabulary.ID).
		Updates(map[string]any{
			"result":        services.ExerciseVocabularyResultIgnored,
			"result_reason": services.ExerciseVocabularyResultReasonDeletedVocabulary,
		}).Error)

	update := telegramPrivateMessage(telegramID, "Hund")
	message := update["message"].(map[string]any)
	message["reply_to_message"] = map[string]any{
		"message_id": exerciseMessageID,
		"chat":       map[string]any{"id": telegramID, "type": "private"},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.Len(t, tg.RequestsFor("editMessageReplyMarkup"), 1)
	requests := tg.RequestsFor("sendMessage")
	require.Len(t, requests, 1)
	assert.Contains(t, string(requests[0].Body), "нельзя выполнить")
}

func TestTelegramReplyToCancelledAudioDoesNotScoreOrTranslate(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555097
	const exerciseMessageID int64 = 499
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID), testkit.WithSettings(models.UserSettings{
		SystemLanguage: enums.LanguageEn,
	}))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeAudioDirect, enums.ExerciseStatusInProgress, vocabulary.ID)
	require.NoError(t, db.DB.Model(&models.Exercise{}).
		Where("id = ?", exercise.ID).
		Update("telegram_message_id", exerciseMessageID).Error)
	beforeProgress := exerciseReloadVocabulary(t, vocabulary.ID).Progress

	_, err := services.IgnoreAudioLanguageForExercise(exercise.ID, user.ID)
	require.NoError(t, err)

	update := telegramPrivateMessage(telegramID, "carta")
	message := update["message"].(map[string]any)
	message["reply_to_message"] = map[string]any{
		"message_id": exerciseMessageID,
		"chat":       map[string]any{"id": telegramID, "type": "private"},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	requests := tg.RequestsFor("sendMessage")
	require.Len(t, requests, 1)
	assert.Contains(t, string(requests[0].Body), "cancelled")
	assert.Empty(t, tg.RequestsFor("editMessageReplyMarkup"))

	var deleted models.Exercise
	require.NoError(t, db.DB.Unscoped().Where("id = ?", exercise.ID).Take(&deleted).Error)
	assert.Equal(t, enums.ExerciseStatusInProgress, deleted.Status)
	link := exerciseLink(t, exercise.ID, vocabulary.ID)
	assert.Nil(t, link.Result)
	assert.Nil(t, link.ProgressDelta)
	assert.Equal(t, beforeProgress, exerciseReloadVocabulary(t, vocabulary.ID).Progress)
}

func TestTelegramAudioIgnoreAndUndoCallbacksAreIdempotent(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555096
	const exerciseMessageID int64 = 498
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID), testkit.WithSettings(models.UserSettings{
		SystemLanguage: enums.LanguageEn,
	}))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeAudioDirect, enums.ExerciseStatusInProgress, vocabulary.ID)
	require.NoError(t, db.DB.Model(&models.Exercise{}).
		Where("id = ?", exercise.ID).
		Update("telegram_message_id", exerciseMessageID).Error)

	callback := func(id, action string) map[string]any {
		return map[string]any{
			"update_id": 1,
			"callback_query": map[string]any{
				"id":   id,
				"data": "exercise:" + action + ":" + telegramCompactUUID(exercise.ID) + ":en",
				"from": map[string]any{"id": telegramID, "is_bot": false},
				"message": map[string]any{
					"message_id": exerciseMessageID,
					"chat":       map[string]any{"id": telegramID, "type": "private"},
				},
			},
		}
	}

	for _, callbackID := range []string{"ignore-1", "ignore-2"} {
		rec := telegramUpdate(t, callback(callbackID, "ai"))
		testkit.RequireStatus(t, rec, http.StatusOK)
	}
	var storedUser models.User
	require.NoError(t, db.DB.Where("id = ?", user.ID).Take(&storedUser).Error)
	assert.Equal(t, []enums.Language{enums.LanguageEn}, storedUser.Settings.IgnoredAudioLanguages)

	for _, callbackID := range []string{"undo-1", "undo-2"} {
		rec := telegramUpdate(t, callback(callbackID, "au"))
		testkit.RequireStatus(t, rec, http.StatusOK)
	}
	require.NoError(t, db.DB.Where("id = ?", user.ID).Take(&storedUser).Error)
	assert.Empty(t, storedUser.Settings.IgnoredAudioLanguages)

	var deleted models.Exercise
	require.NoError(t, db.DB.Unscoped().Where("id = ?", exercise.ID).Take(&deleted).Error)
	assert.True(t, deleted.DeletedAt.Valid)
	assert.Equal(t, enums.ExerciseStatusInProgress, deleted.Status)
	assert.Len(t, tg.RequestsFor("answerCallbackQuery"), 4)
	assert.Len(t, tg.RequestsFor("editMessageCaption"), 2)
	assert.Len(t, tg.RequestsFor("editMessageReplyMarkup"), 2)

	var cancelledCaption struct {
		Caption     string `json:"caption"`
		ReplyMarkup struct {
			InlineKeyboard [][]telegramKeyboardButton `json:"inline_keyboard"`
		} `json:"reply_markup"`
	}
	require.NoError(t, json.Unmarshal(tg.RequestsFor("editMessageCaption")[0].Body, &cancelledCaption))
	assert.Equal(t, "Listen and translate into 🇮🇹 Italian.\n\nExercise was cancelled.", cancelledCaption.Caption)
	_, hasUndo := findCallbackButton(cancelledCaption.ReplyMarkup.InlineKeyboard, "exercise:au:")
	assert.True(t, hasUndo)

	var undoEdit telegramKeyboardRequest
	require.NoError(t, json.Unmarshal(tg.RequestsFor("editMessageReplyMarkup")[0].Body, &undoEdit))
	assert.Empty(t, undoEdit.ReplyMarkup.InlineKeyboard)
}

func TestTelegramAudioIgnoreCallbackUsesActiveLanguage(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555095
	const exerciseMessageID int64 = 497
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID), testkit.WithSettings(models.UserSettings{
		SystemLanguage: enums.LanguageRu,
	}))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeAudioDirect, enums.ExerciseStatusInProgress, vocabulary.ID)
	require.NoError(t, db.DB.Model(&models.Exercise{}).
		Where("id = ?", exercise.ID).
		Update("telegram_message_id", exerciseMessageID).Error)

	update := map[string]any{
		"update_id": 1,
		"callback_query": map[string]any{
			"id":   "ignore-ru",
			"data": "exercise:ai:" + telegramCompactUUID(exercise.ID) + ":en",
			"from": map[string]any{"id": telegramID, "is_bot": false},
			"message": map[string]any{
				"message_id": exerciseMessageID,
				"chat":       map[string]any{"id": telegramID, "type": "private"},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.Len(t, tg.RequestsFor("editMessageCaption"), 1)

	var edited struct {
		Caption string `json:"caption"`
	}
	require.NoError(t, json.Unmarshal(tg.RequestsFor("editMessageCaption")[0].Body, &edited))
	assert.Equal(t, "Прослушай и переведи. Язык ответа: 🇮🇹 Итальянский.\n\nУпражнение отменено.", edited.Caption)
}

func TestTelegramWebhookIgnoredDeletedVocabularyCallbacksRetryKeyboardRemoval(t *testing.T) {
	tests := []struct {
		name         string
		exerciseType enums.ExerciseType
		payload      func(exerciseID, vocabularyID uuid.UUID) string
	}{
		{
			name:         "basic",
			exerciseType: enums.ExerciseTypeBasicDirect,
			payload: func(exerciseID, _ uuid.UUID) string {
				return "exercise:idk:" + exerciseID.String()
			},
		},
		{
			name:         "choice",
			exerciseType: enums.ExerciseTypeChoiceDirect,
			payload: func(exerciseID, vocabularyID uuid.UUID) string {
				return "exercise:answer:" + telegramCompactUUID(exerciseID) + ":" + telegramCompactUUID(vocabularyID)
			},
		},
		{
			name:         "character",
			exerciseType: enums.ExerciseTypeCharactersDirect,
			payload: func(exerciseID, _ uuid.UUID) string {
				return "exercise:ct:" + telegramCompactUUID(exerciseID) + ":0"
			},
		},
		{
			name:         "match pairs",
			exerciseType: enums.ExerciseTypeMatchPairs,
			payload: func(exerciseID, _ uuid.UUID) string {
				return "exercise:mt:" + telegramCompactUUID(exerciseID) + ":0"
			},
		},
	}

	for index, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testkit.Truncate(t)
			tg := testkit.MockTelegramAPI(t)

			telegramID := int64(555100 + index)
			messageID := int64(510 + index)
			user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))
			vocabulary := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
			exercise := exerciseSeedExercise(t, user.ID, testCase.exerciseType, enums.ExerciseStatusIgnored, vocabulary.ID)
			require.NoError(t, db.DB.Model(&models.Exercise{}).
				Where("id = ?", exercise.ID).
				Update("telegram_message_id", messageID).Error)
			require.NoError(t, db.DB.Model(&models.ExerciseVocabulary{}).
				Where("exercise_id = ? AND vocabulary_id = ?", exercise.ID, vocabulary.ID).
				Updates(map[string]any{
					"result":        services.ExerciseVocabularyResultIgnored,
					"result_reason": services.ExerciseVocabularyResultReasonDeletedVocabulary,
				}).Error)

			update := map[string]any{
				"update_id": 110 + index,
				"callback_query": map[string]any{
					"id":   "cb-ignored-deleted-" + strconv.Itoa(index),
					"data": testCase.payload(exercise.ID, vocabulary.ID),
					"from": map[string]any{"id": telegramID, "is_bot": false},
					"message": map[string]any{
						"message_id": messageID,
						"chat":       map[string]any{"id": telegramID, "type": "private"},
					},
				},
			}

			rec := telegramUpdate(t, update)
			testkit.RequireStatus(t, rec, http.StatusOK)
			require.Len(t, tg.RequestsFor("editMessageReplyMarkup"), 1)
			requests := tg.RequestsFor("sendMessage")
			require.Len(t, requests, 1)
			assert.Contains(t, string(requests[0].Body), "нельзя выполнить")
		})
	}
}

// -----------------------------------------------------------------------------
// Message updates
// -----------------------------------------------------------------------------

func TestTelegramWebhookStartCommandCreatesUserAndReplies(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555001

	rec := telegramUpdate(t, telegramPrivateMessage(telegramID, "/start"))
	testkit.RequireStatus(t, rec, http.StatusOK)

	// Side effect 1: a user row was created for the new telegram id, with the bot
	// marked enabled by ensurePrivateMessageUser.
	var user models.User
	require.NoError(t, db.DB.Where("telegram_id = ?", telegramID).First(&user).Error)
	assert.Equal(t, "ada", user.Username)
	assert.True(t, user.Settings.Telegram.BotEnabled)

	// Side effect 2: a reply (sendMessage) was dispatched to the user.
	require.True(t, tg.Sent("sendMessage"), "expected a sendMessage reply to /start")

	var sent map[string]any
	require.NoError(t, json.Unmarshal(tg.RequestsFor("sendMessage")[0].Body, &sent))
	assert.EqualValues(t, telegramID, sent["chat_id"])
}

func TestTelegramWebhookPlainTextTranslatesAndReplies(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	// A plain text message in a private chat is treated as a word to translate and
	// auto-add to the vocabulary; make the translate result deterministic.
	testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
		TranslateFunc: func(text, src, dst string) (string, error) { return "perro", nil },
		DetectFunc:    func(text string) (string, error) { return "en", nil },
	})

	const telegramID int64 = 555002

	rec := telegramUpdate(t, telegramPrivateMessage(telegramID, "dog"))
	testkit.RequireStatus(t, rec, http.StatusOK)

	// New user ensured.
	var user models.User
	require.NoError(t, db.DB.Where("telegram_id = ?", telegramID).First(&user).Error)

	// A reply carrying the translation was dispatched.
	require.True(t, tg.Sent("sendMessage"))
	var sent map[string]any
	require.NoError(t, json.Unmarshal(tg.RequestsFor("sendMessage")[0].Body, &sent))
	assert.Contains(t, sent["text"], "perro")

	// The translated word was auto-added to the user's vocabulary.
	var vocabCount int64
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).Where("user_id = ?", user.ID).Count(&vocabCount).Error)
	assert.EqualValues(t, 1, vocabCount, "plain text should auto-add a vocabulary entry")
}

func TestTelegramWebhookCancelCommandClearsState(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555003

	// Seed a user already in a non-none telegram state.
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))
	require.NoError(t, db.DB.Model(&models.User{}).Where("id = ?", user.ID).
		Update("telegram_state", enums.TelegramStateAddingVocabulary).Error)

	rec := telegramUpdate(t, telegramPrivateMessage(telegramID, "/cancel"))
	testkit.RequireStatus(t, rec, http.StatusOK)

	// State reset to none.
	var refreshed models.User
	require.NoError(t, db.DB.Where("id = ?", user.ID).First(&refreshed).Error)
	assert.Equal(t, enums.TelegramStateNone, refreshed.TelegramState)

	// A confirmation reply was sent.
	require.True(t, tg.Sent("sendMessage"))
}

// -----------------------------------------------------------------------------
// CallbackQuery updates
// -----------------------------------------------------------------------------

func TestTelegramWebhookCallbackAnswersAndEdits(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555004
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))

	// "menu:statistics" is a reachable branch that loads the user and edits the
	// message — no extra pre-seeded exercise/vocabulary state is required.
	update := map[string]any{
		"update_id": 2,
		"callback_query": map[string]any{
			"id":   "cb-1",
			"data": "menu:statistics",
			"from": map[string]any{
				"id":         telegramID,
				"is_bot":     false,
				"first_name": "Ada",
				"username":   "ada",
			},
			"message": map[string]any{
				"message_id": 77,
				"date":       1700000000,
				"chat": map[string]any{
					"id":   telegramID,
					"type": "private",
				},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	// Every callback is acknowledged first.
	require.True(t, tg.Sent("answerCallbackQuery"), "callback queries must be answered")
	var answered map[string]any
	require.NoError(t, json.Unmarshal(tg.RequestsFor("answerCallbackQuery")[0].Body, &answered))
	assert.Equal(t, "cb-1", answered["callback_query_id"])

	// The statistics menu edits the existing message in place.
	require.True(t, tg.Sent("editMessageText"), "statistics menu should edit the message")
	var edited map[string]any
	require.NoError(t, json.Unmarshal(tg.RequestsFor("editMessageText")[0].Body, &edited))
	assert.EqualValues(t, telegramID, edited["chat_id"])
	assert.EqualValues(t, 77, edited["message_id"])

	_ = user
}

func TestTelegramWebhookCallbackDeleteTranslationSetsState(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555005
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))

	update := map[string]any{
		"update_id": 3,
		"callback_query": map[string]any{
			"id":   "cb-2",
			"data": "menu:delete_translation",
			"from": map[string]any{
				"id":     telegramID,
				"is_bot": false,
			},
			"message": map[string]any{
				"message_id": 88,
				"chat": map[string]any{
					"id":   telegramID,
					"type": "private",
				},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	// DB side effect: the user's telegram state moves to deletingVocabulary.
	var refreshed models.User
	require.NoError(t, db.DB.Where("id = ?", user.ID).First(&refreshed).Error)
	assert.Equal(t, enums.TelegramStateDeletingVocabulary, refreshed.TelegramState)

	require.True(t, tg.Sent("answerCallbackQuery"))
	require.True(t, tg.Sent("editMessageText"))
}

func TestTelegramWebhookDeleteByWordConfirmsPairAndHandlesRepeat(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555020
	user := testkit.CreateUser(t,
		testkit.WithTelegramID(telegramID),
		testkit.WithSettings(models.UserSettings{SystemLanguage: enums.LanguageEn}),
	)
	seedVocabularyForWordDeletion(t, user.ID, "river", "Fluss")
	require.NoError(t, db.DB.Model(&user).Update("telegram_state", enums.TelegramStateDeletingVocabulary).Error)

	rec := telegramUpdate(t, telegramPrivateMessage(telegramID, "  FLUSS "))
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.Len(t, tg.RequestsFor("sendMessage"), 1)
	var sent struct {
		Text string `json:"text"`
	}
	require.NoError(t, json.Unmarshal(tg.RequestsFor("sendMessage")[0].Body, &sent))
	assert.Equal(t, "Deleted: river — Fluss ✅", sent.Text)
	assert.Equal(t, int64(0), vocabCountForUser(t, user.ID))

	require.NoError(t, db.DB.Model(&user).Update("telegram_state", enums.TelegramStateDeletingVocabulary).Error)
	rec = telegramUpdate(t, telegramPrivateMessage(telegramID, "Fluss"))
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.Len(t, tg.RequestsFor("sendMessage"), 2)
	require.NoError(t, json.Unmarshal(tg.RequestsFor("sendMessage")[1].Body, &sent))
	assert.Equal(t, "Word not found ❌", sent.Text)
}

func TestTelegramWebhookDeleteByWordAsksForAmbiguousPair(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555021
	user := testkit.CreateUser(t,
		testkit.WithTelegramID(telegramID),
		testkit.WithSettings(models.UserSettings{SystemLanguage: enums.LanguageEn}),
	)
	seedVocabularyForWordDeletion(t, user.ID, "bank", "Ufer")
	seedVocabularyForWordDeletion(t, user.ID, "bank", "Bank")
	require.NoError(t, db.DB.Model(&user).Update("telegram_state", enums.TelegramStateDeletingVocabulary).Error)

	rec := telegramUpdate(t, telegramPrivateMessage(telegramID, "bank"))
	testkit.RequireStatus(t, rec, http.StatusOK)
	assert.Equal(t, int64(2), vocabCountForUser(t, user.ID))
	require.Len(t, tg.RequestsFor("sendMessage"), 1)
	var sent struct {
		Text string `json:"text"`
	}
	require.NoError(t, json.Unmarshal(tg.RequestsFor("sendMessage")[0].Body, &sent))
	assert.Equal(t, "Several translations match that word. Send the exact pair as word1:word2.", sent.Text)
}

func TestTelegramWebhookVocabularyAddCallbackIsReplaySafe(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555006
	const chatID int64 = 88006
	const messageID int64 = 99
	const originalText = "dog → Hund"

	user := testkit.CreateUser(t,
		testkit.WithTelegramID(telegramID),
		testkit.WithSettings(models.UserSettings{SystemLanguage: enums.LanguageEn}),
	)
	translationID := vocabSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)

	update := map[string]any{
		"update_id": 4,
		"callback_query": map[string]any{
			"id":   "cb-vocabulary-add",
			"data": "vocabulary:add:" + translationID.String(),
			"from": map[string]any{
				"id":     telegramID,
				"is_bot": false,
			},
			"message": map[string]any{
				"message_id": messageID,
				"text":       originalText,
				"chat": map[string]any{
					"id":   chatID,
					"type": "private",
				},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	assert.Equal(t, int64(1), vocabCountForUser(t, user.ID))
	var stored models.Vocabulary
	require.NoError(t, db.DB.
		Where("user_id = ? AND translation_id = ?", user.ID, translationID).
		First(&stored).Error)

	require.Len(t, tg.RequestsFor("answerCallbackQuery"), 1)
	var answerRequest map[string]any
	require.NoError(t, json.Unmarshal(tg.RequestsFor("answerCallbackQuery")[0].Body, &answerRequest))
	assert.Equal(t, "cb-vocabulary-add", answerRequest["callback_query_id"])

	require.Len(t, tg.RequestsFor("editMessageText"), 1)
	var editRequest struct {
		ChatID      int64  `json:"chat_id"`
		MessageID   int64  `json:"message_id"`
		Text        string `json:"text"`
		ReplyMarkup struct {
			InlineKeyboard [][]struct {
				Text         string `json:"text"`
				CallbackData string `json:"callback_data"`
			} `json:"inline_keyboard"`
		} `json:"reply_markup"`
	}
	require.NoError(t, json.Unmarshal(tg.RequestsFor("editMessageText")[0].Body, &editRequest))
	assert.Equal(t, chatID, editRequest.ChatID)
	assert.Equal(t, messageID, editRequest.MessageID)
	assert.Equal(t, originalText+"\n\nSuccessfully added to your vocabulary", editRequest.Text)
	require.Len(t, editRequest.ReplyMarkup.InlineKeyboard, 1)
	require.Len(t, editRequest.ReplyMarkup.InlineKeyboard[0], 1)
	assert.Equal(t, "🔊 Pronunciation", editRequest.ReplyMarkup.InlineKeyboard[0][0].Text)
	assert.Equal(t, "pronunciation:"+telegramCompactUUID(translationID), editRequest.ReplyMarkup.InlineKeyboard[0][0].CallbackData)
	assert.False(t, tg.Sent("sendMessage"), "the callback should edit, not send, the message")

	// Telegram may redeliver an update. The service treats an existing vocabulary
	// item as success, so replaying this callback must not create a duplicate.
	replayed := telegramUpdate(t, update)
	testkit.RequireStatus(t, replayed, http.StatusOK)
	assert.Equal(t, int64(1), vocabCountForUser(t, user.ID))
	assert.Equal(t, 2, tg.Count("answerCallbackQuery"))
	assert.Equal(t, 2, tg.Count("editMessageText"))
}

func TestTelegramWebhookCharacterExerciseUsesSquareBoardAndCompletes(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555013
	const messageID int64 = 104
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "carta", "letter", enums.LanguageIt, enums.LanguageEn)
	exercise := exerciseSeedExercise(
		t,
		user.ID,
		enums.ExerciseTypeCharactersDirect,
		enums.ExerciseStatusPending,
		vocabulary.ID,
	)
	require.NoError(t, services.StartCharacterExercise(exercise.ID, messageID, []int{5, -1, 0, 4, -1, 1, 3, 2, -1}))

	sendCharacterCallback := func(callbackID string, action string, updateID int) {
		t.Helper()
		update := map[string]any{
			"update_id": updateID,
			"callback_query": map[string]any{
				"id":   callbackID,
				"data": "exercise:" + action,
				"from": map[string]any{
					"id":     telegramID,
					"is_bot": false,
				},
				"message": map[string]any{
					"message_id": messageID,
					"chat": map[string]any{
						"id":   telegramID,
						"type": "private",
					},
				},
			},
		}

		rec := telegramUpdate(t, update)
		testkit.RequireStatus(t, rec, http.StatusOK)
	}

	compactExerciseID := telegramCompactUUID(exercise.ID)
	sendCharacterCallback("cb-character-initial", "ct:"+compactExerciseID+":0", 70)

	editRequests := tg.RequestsFor("editMessageText")
	require.Len(t, editRequests, 1)

	var edited map[string]any
	require.NoError(t, json.Unmarshal(editRequests[0].Body, &edited))
	replyMarkup, ok := edited["reply_markup"].(map[string]any)
	require.True(t, ok)
	keyboard, ok := replyMarkup["inline_keyboard"].([]any)
	require.True(t, ok)
	require.Len(t, keyboard, 4, "the 3x3 character grid should have a separate action row")
	for _, rowValue := range keyboard[:3] {
		row, ok := rowValue.([]any)
		require.True(t, ok)
		require.Len(t, row, 3)
	}
	firstRow := keyboard[0].([]any)
	firstRowMiddle := firstRow[1].(map[string]any)
	assert.Equal(t, "exercise:cn", firstRowMiddle["callback_data"], "empty cells should be distributed inside the board")
	actionRow := keyboard[3].([]any)
	require.Len(t, actionRow, 2)
	backspaceButton := actionRow[0].(map[string]any)
	assert.Equal(t, "⌫", backspaceButton["text"])
	assert.Equal(t, "exercise:cc:"+compactExerciseID, backspaceButton["callback_data"])
	idkButton := actionRow[1].(map[string]any)
	assert.Equal(t, telegram.GetBotTexts(enums.LanguageRu).ButtonExerciseIDK, idkButton["text"])
	assert.Equal(t, "exercise:idk:"+exercise.ID.String(), idkButton["callback_data"])
	assert.Contains(t, edited["text"], "l ＿ ＿ ＿ ＿ ＿")

	sendCharacterCallback("cb-character-backspace", "cc:"+compactExerciseID, 71)
	backspaceEdits := tg.RequestsFor("editMessageText")
	require.Len(t, backspaceEdits, 2)
	var backedUp map[string]any
	require.NoError(t, json.Unmarshal(backspaceEdits[1].Body, &backedUp))
	assert.Contains(t, backedUp["text"], "＿ ＿ ＿ ＿ ＿ ＿")

	for tappedIndex := range []rune("letter") {
		sendCharacterCallback(
			"cb-character-"+strconv.Itoa(tappedIndex),
			"ct:"+compactExerciseID+":"+strconv.Itoa(tappedIndex),
			72+tappedIndex,
		)
	}

	stored := exerciseReload(t, exercise.ID)
	assert.Equal(t, enums.ExerciseStatusCompleted, stored.Status)

	editRequests = tg.RequestsFor("editMessageText")
	require.Len(t, editRequests, len([]rune("letter"))+2)
	var finalEdit map[string]any
	require.NoError(t, json.Unmarshal(editRequests[len(editRequests)-1].Body, &finalEdit))
	replyMarkup, ok = finalEdit["reply_markup"].(map[string]any)
	require.True(t, ok)
	keyboard, ok = replyMarkup["inline_keyboard"].([]any)
	require.True(t, ok)
	assert.Empty(t, keyboard)

	require.True(t, tg.Sent("sendMessage"), "completion should send the exercise result")
	link := exerciseLink(t, exercise.ID, vocabulary.ID)
	require.NotNil(t, link.ResultReason)
	assert.Equal(t, services.ExerciseVocabularyResultReasonCharacterAnswer, *link.ResultReason)

	feedbackCount := len(tg.RequestsFor("sendMessage"))
	retry := map[string]any{
		"update_id": 99,
		"callback_query": map[string]any{
			"id":   "cb-character-retry",
			"data": "exercise:ct:" + telegramCompactUUID(exercise.ID) + ":0",
			"from": map[string]any{
				"id":     telegramID,
				"is_bot": false,
			},
			"message": map[string]any{
				"message_id": messageID,
				"chat": map[string]any{
					"id":   telegramID,
					"type": "private",
				},
			},
		},
	}
	rec := telegramUpdate(t, retry)
	testkit.RequireStatus(t, rec, http.StatusOK)
	assert.Len(t, tg.RequestsFor("sendMessage"), feedbackCount, "a retried callback must not duplicate feedback")

	retryEdits := tg.RequestsFor("editMessageText")
	require.Len(t, retryEdits, len([]rune("letter"))+3)
	var retryEdit map[string]any
	require.NoError(t, json.Unmarshal(retryEdits[len(retryEdits)-1].Body, &retryEdit))
	assert.Contains(t, retryEdit["text"], "l e t t e r")
}

func TestTelegramWebhookCharacterExerciseCanBeFailedWithIDK(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555014
	const messageID int64 = 105
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "carta", "letter", enums.LanguageIt, enums.LanguageEn)
	exercise := exerciseSeedExercise(
		t,
		user.ID,
		enums.ExerciseTypeCharactersDirect,
		enums.ExerciseStatusPending,
		vocabulary.ID,
	)
	require.NoError(t, services.StartCharacterExercise(exercise.ID, messageID, []int{5, -1, 0, 4, -1, 1, 3, 2, -1}))

	update := map[string]any{
		"update_id": 79,
		"callback_query": map[string]any{
			"id":   "cb-character-idk",
			"data": "exercise:idk:" + exercise.ID.String(),
			"from": map[string]any{
				"id":     telegramID,
				"is_bot": false,
			},
			"message": map[string]any{
				"message_id": messageID,
				"chat": map[string]any{
					"id":   telegramID,
					"type": "private",
				},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	stored := exerciseReload(t, exercise.ID)
	assert.Equal(t, enums.ExerciseStatusFailed, stored.Status)
	link := exerciseLink(t, exercise.ID, vocabulary.ID)
	require.NotNil(t, link.Result)
	require.NotNil(t, link.ResultReason)
	assert.Equal(t, services.ExerciseVocabularyResultIgnored, *link.Result)
	assert.Equal(t, services.ExerciseVocabularyResultReasonSkipped, *link.ResultReason)
	require.NotNil(t, link.ProgressDelta)
	assert.Equal(t, services.ExerciseCharacterWrongProgressDelta, *link.ProgressDelta)
	require.Len(t, tg.RequestsFor("editMessageReplyMarkup"), 1)
	require.Len(t, tg.RequestsFor("sendMessage"), 1)
	assert.Contains(t, string(tg.RequestsFor("sendMessage")[0].Body), "carta")
	assert.Contains(t, string(tg.RequestsFor("sendMessage")[0].Body), "letter")
}

func TestTelegramWebhookCharacterTapReportsDeletedVocabulary(t *testing.T) {
	for _, exerciseType := range []enums.ExerciseType{
		enums.ExerciseTypeCharactersDirect,
		enums.ExerciseTypeCharactersReversed,
	} {
		t.Run(string(exerciseType), func(t *testing.T) {
			testkit.Truncate(t)
			tg := testkit.MockTelegramAPI(t)

			const telegramID int64 = 555015
			const messageID int64 = 106
			user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))
			vocabulary := exerciseSeedVocabulary(t, user.ID, "carta", "letter", enums.LanguageIt, enums.LanguageEn)
			exercise := exerciseSeedExercise(t, user.ID, exerciseType, enums.ExerciseStatusPending, vocabulary.ID)
			require.NoError(t, services.StartCharacterExercise(exercise.ID, messageID, []int{5, 4, 3, 2, 1, 0}))
			require.NoError(t, services.DeleteVocabulary(user.ID, vocabulary.ID))

			update := map[string]any{
				"update_id": 80,
				"callback_query": map[string]any{
					"id":   "cb-character-deleted",
					"data": "exercise:ct:" + telegramCompactUUID(exercise.ID) + ":0",
					"from": map[string]any{"id": telegramID, "is_bot": false},
					"message": map[string]any{
						"message_id": messageID,
						"chat":       map[string]any{"id": telegramID, "type": "private"},
					},
				},
			}

			rec := telegramUpdate(t, update)
			testkit.RequireStatus(t, rec, http.StatusOK)

			stored := exerciseReload(t, exercise.ID)
			assert.Equal(t, enums.ExerciseStatusIgnored, stored.Status)
			link := exerciseLink(t, exercise.ID, vocabulary.ID)
			require.NotNil(t, link.ResultReason)
			assert.Equal(t, services.ExerciseVocabularyResultReasonDeletedVocabulary, *link.ResultReason)
			cancelEdits := tg.RequestsFor("editMessageText")
			require.Len(t, cancelEdits, 1)
			assert.Contains(t, string(cancelEdits[0].Body), "отменено")
			requests := tg.RequestsFor("sendMessage")
			require.Len(t, requests, 1)
			assert.Contains(t, string(requests[0].Body), "нельзя выполнить")
			assert.NotContains(t, string(requests[0].Body), "устарело")
		})
	}
}

func TestDeleteVocabularyReplacesActiveTelegramExerciseWithCancellation(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555099
	const messageID int64 = 199
	user := testkit.CreateUser(t,
		testkit.WithTelegramID(telegramID),
		testkit.WithSettings(models.UserSettings{SystemLanguage: enums.LanguageEn}),
	)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocabulary.ID)
	require.NoError(t, db.DB.Model(&models.Exercise{}).Where("id = ?", exercise.ID).Update("telegram_message_id", messageID).Error)

	require.NoError(t, services.DeleteVocabulary(user.ID, vocabulary.ID))

	requests := tg.RequestsFor("editMessageText")
	require.Len(t, requests, 1)
	var edited struct {
		ChatID      int64  `json:"chat_id"`
		MessageID   int64  `json:"message_id"`
		Text        string `json:"text"`
		ReplyMarkup struct {
			InlineKeyboard []any `json:"inline_keyboard"`
		} `json:"reply_markup"`
	}
	require.NoError(t, json.Unmarshal(requests[0].Body, &edited))
	assert.Equal(t, telegramID, edited.ChatID)
	assert.Equal(t, messageID, edited.MessageID)
	assert.Equal(t, "This exercise was cancelled because its vocabulary was deleted 🗑️", edited.Text)
	assert.Empty(t, edited.ReplyMarkup.InlineKeyboard)
	assert.False(t, tg.Sent("sendMessage"))
}

func TestTelegramWebhookCharacterBackspaceRecoversPendingMessage(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555014
	const messageID int64 = 105
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "carta", "letter", enums.LanguageIt, enums.LanguageEn)
	exercise := exerciseSeedExercise(
		t,
		user.ID,
		enums.ExerciseTypeCharactersDirect,
		enums.ExerciseStatusPending,
		vocabulary.ID,
	)

	compactExerciseID := telegramCompactUUID(exercise.ID)
	tap := func(index int) string {
		return "exercise:ct:" + compactExerciseID + ":" + strconv.Itoa(index)
	}
	inlineKeyboard := [][]map[string]string{
		{
			{"text": "r", "callback_data": tap(5)},
			{"text": " ", "callback_data": "exercise:cn"},
			{"text": "l", "callback_data": tap(0)},
		},
		{
			{"text": "e", "callback_data": tap(4)},
			{"text": " ", "callback_data": "exercise:cn"},
			{"text": "e", "callback_data": tap(1)},
		},
		{
			{"text": "t", "callback_data": tap(3)},
			{"text": "t", "callback_data": tap(2)},
			{"text": " ", "callback_data": "exercise:cn"},
		},
		{
			{"text": "⌫", "callback_data": "exercise:cc:" + compactExerciseID},
			{"text": "Не знаю", "callback_data": "exercise:idk:" + exercise.ID.String()},
		},
	}

	update := map[string]any{
		"update_id": 100,
		"callback_query": map[string]any{
			"id":   "cb-character-pending-backspace",
			"data": "exercise:cc:" + compactExerciseID,
			"from": map[string]any{
				"id":     telegramID,
				"is_bot": false,
			},
			"message": map[string]any{
				"message_id": messageID,
				"chat": map[string]any{
					"id":   telegramID,
					"type": "private",
				},
				"reply_markup": map[string]any{
					"inline_keyboard": inlineKeyboard,
				},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.True(t, tg.Sent("editMessageText"))

	stored := exerciseReload(t, exercise.ID)
	assert.Equal(t, enums.ExerciseStatusInProgress, stored.Status)
	require.NotNil(t, stored.TelegramMessageID)
	assert.EqualValues(t, messageID, *stored.TelegramMessageID)
	require.NotNil(t, stored.CharacterState)

	var state struct {
		Order  []int `json:"order"`
		Chosen []int `json:"chosen"`
	}
	require.NoError(t, json.Unmarshal([]byte(*stored.CharacterState), &state))
	assert.Equal(t, []int{5, -1, 0, 4, -1, 1, 3, 2, -1}, state.Order)
	assert.Empty(t, state.Chosen)
}

func TestTelegramWebhookMatchTapEditsBoard(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555008
	const messageID int64 = 99
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))

	vocabularies := []models.Vocabulary{
		exerciseSeedVocabulary(t, user.ID, "release", "rilasciare", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "cell", "la cella", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "sentence", "la condanna", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "prison", "la prigione", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "guard", "la guardia", enums.LanguageEn, enums.LanguageIt),
	}
	vocabularyIDs := make([]uuid.UUID, 0, len(vocabularies))
	for _, vocabulary := range vocabularies {
		vocabularyIDs = append(vocabularyIDs, vocabulary.ID)
	}

	exercise := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusPending, vocabularyIDs)
	require.NoError(t, services.StartMatchExercise(exercise.ID, messageID, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))

	update := map[string]any{
		"update_id": 5,
		"callback_query": map[string]any{
			"id":   "cb-match-1",
			"data": "exercise:mt:" + telegramCompactUUID(exercise.ID) + ":0",
			"from": map[string]any{
				"id":     telegramID,
				"is_bot": false,
			},
			"message": map[string]any{
				"message_id": messageID,
				"chat": map[string]any{
					"id":   telegramID,
					"type": "private",
				},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	require.True(t, tg.Sent("answerCallbackQuery"))
	require.True(t, tg.Sent("editMessageText"), "match tap should re-render the board")

	var edited map[string]any
	require.NoError(t, json.Unmarshal(tg.RequestsFor("editMessageText")[0].Body, &edited))
	assert.EqualValues(t, telegramID, edited["chat_id"])
	assert.EqualValues(t, messageID, edited["message_id"])
	replyMarkup, ok := edited["reply_markup"].(map[string]any)
	require.True(t, ok)
	keyboard, ok := replyMarkup["inline_keyboard"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, keyboard)
	firstRow, ok := keyboard[0].([]any)
	require.True(t, ok)
	require.NotEmpty(t, firstRow)
	firstButton, ok := firstRow[0].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, firstButton["text"], "▸ ")

	var refreshed models.Exercise
	require.NoError(t, db.DB.Where("id = ?", exercise.ID).First(&refreshed).Error)
	require.NotNil(t, refreshed.MatchState)
	var matchState struct {
		Pending int `json:"pending"`
	}
	require.NoError(t, json.Unmarshal([]byte(*refreshed.MatchState), &matchState))
	assert.Equal(t, 0, matchState.Pending)
}

func TestTelegramWebhookMatchTapReportsDeletedVocabulary(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555016
	const messageID int64 = 107
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))
	vocabularyIDs := make([]uuid.UUID, 0, services.MatchPairsVocabularyCount)
	for index := 0; index < services.MatchPairsVocabularyCount; index++ {
		vocabulary := exerciseSeedVocabulary(
			t, user.ID,
			"original-"+strconv.Itoa(index), "translation-"+strconv.Itoa(index),
			enums.LanguageEn, enums.LanguageIt,
		)
		vocabularyIDs = append(vocabularyIDs, vocabulary.ID)
	}
	exercise := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusPending, vocabularyIDs)
	require.NoError(t, services.StartMatchExercise(exercise.ID, messageID, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))
	require.NoError(t, services.DeleteVocabulary(user.ID, vocabularyIDs[0]))

	update := map[string]any{
		"update_id": 81,
		"callback_query": map[string]any{
			"id":   "cb-match-deleted",
			"data": "exercise:mt:" + telegramCompactUUID(exercise.ID) + ":0",
			"from": map[string]any{"id": telegramID, "is_bot": false},
			"message": map[string]any{
				"message_id": messageID,
				"chat":       map[string]any{"id": telegramID, "type": "private"},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	stored := exerciseReload(t, exercise.ID)
	assert.Equal(t, enums.ExerciseStatusIgnored, stored.Status)
	for _, vocabularyID := range vocabularyIDs {
		link := exerciseLink(t, exercise.ID, vocabularyID)
		require.NotNil(t, link.ResultReason)
		assert.Equal(t, services.ExerciseVocabularyResultReasonDeletedVocabulary, *link.ResultReason)
	}
	cancelEdits := tg.RequestsFor("editMessageText")
	require.Len(t, cancelEdits, 1)
	assert.Contains(t, string(cancelEdits[0].Body), "отменено")
	requests := tg.RequestsFor("sendMessage")
	require.Len(t, requests, 1)
	assert.Contains(t, string(requests[0].Body), "нельзя выполнить")
	assert.NotContains(t, string(requests[0].Body), "устарело")
}

func TestTelegramWebhookMatchTapRetriesFinalizationFromPersistedBoard(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555011
	const messageID int64 = 102
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))

	vocabularyIDs := make([]uuid.UUID, 0, services.MatchPairsVocabularyCount)
	for index := 0; index < services.MatchPairsVocabularyCount; index++ {
		vocabulary := exerciseSeedVocabulary(
			t, user.ID,
			"original-"+strconv.Itoa(index), "translation-"+strconv.Itoa(index),
			enums.LanguageEn, enums.LanguageIt,
		)
		vocabularyIDs = append(vocabularyIDs, vocabulary.ID)
	}

	exercise := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusPending, vocabularyIDs)
	require.NoError(t, services.StartMatchExercise(exercise.ID, messageID, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))

	// Persist a fully resolved board without completing the exercise, matching a
	// transient failure between ApplyMatchTap and CompleteMatchPairsExercise.
	for index := 0; index < services.MatchPairsVocabularyCount*2; index++ {
		_, _, _, _, err := services.ApplyMatchTap(exercise.ID, user.ID, index)
		require.NoError(t, err)
	}

	update := map[string]any{
		"update_id": 51,
		"callback_query": map[string]any{
			"id":   "cb-match-retry-finalize",
			"data": "exercise:mt:" + telegramCompactUUID(exercise.ID) + ":0",
			"from": map[string]any{"id": telegramID, "is_bot": false},
			"message": map[string]any{
				"message_id": messageID,
				"chat":       map[string]any{"id": telegramID, "type": "private"},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.True(t, tg.Sent("editMessageText"))

	var refreshed models.Exercise
	require.NoError(t, db.DB.Where("id = ?", exercise.ID).First(&refreshed).Error)
	assert.Equal(t, enums.ExerciseStatusCompleted, refreshed.Status)
}

func TestTelegramWebhookCompletedMatchTapRepairsOriginalMessage(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555012
	const messageID int64 = 103
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))

	vocabularyIDs := make([]uuid.UUID, 0, services.MatchPairsVocabularyCount)
	for index := 0; index < services.MatchPairsVocabularyCount; index++ {
		vocabulary := exerciseSeedVocabulary(
			t, user.ID,
			"original-"+strconv.Itoa(index), "translation-"+strconv.Itoa(index),
			enums.LanguageEn, enums.LanguageIt,
		)
		vocabularyIDs = append(vocabularyIDs, vocabulary.ID)
	}

	exercise := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusPending, vocabularyIDs)
	require.NoError(t, services.StartMatchExercise(exercise.ID, messageID, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))

	var attempts []services.MatchPairAttempt
	for index := 0; index < services.MatchPairsVocabularyCount*2; index++ {
		_, _, _, finalizeAttempts, err := services.ApplyMatchTap(exercise.ID, user.ID, index)
		require.NoError(t, err)
		if len(finalizeAttempts) > 0 {
			attempts = finalizeAttempts
		}
	}
	_, err := services.CompleteMatchPairsExercise(exercise.ID, user.ID, attempts)
	require.NoError(t, err)

	update := map[string]any{
		"update_id": 52,
		"callback_query": map[string]any{
			"id":   "cb-match-repair-completed",
			"data": "exercise:mt:" + telegramCompactUUID(exercise.ID) + ":0",
			"from": map[string]any{"id": telegramID, "is_bot": false},
			"message": map[string]any{
				"message_id": messageID,
				"chat":       map[string]any{"id": telegramID, "type": "private"},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.True(t, tg.Sent("editMessageText"))
	require.False(t, tg.Sent("sendMessage"))

	var edited map[string]any
	require.NoError(t, json.Unmarshal(tg.RequestsFor("editMessageText")[0].Body, &edited))
	replyMarkup, ok := edited["reply_markup"].(map[string]any)
	require.True(t, ok)
	keyboard, ok := replyMarkup["inline_keyboard"].([]any)
	require.True(t, ok)
	assert.Empty(t, keyboard)
}

func TestTelegramWebhookMatchTapMarksWrongCards(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555010
	const messageID int64 = 101
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))

	vocabularies := []models.Vocabulary{
		exerciseSeedVocabulary(t, user.ID, "release", "rilasciare", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "cell", "la cella", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "sentence", "la condanna", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "prison", "la prigione", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "guard", "la guardia", enums.LanguageEn, enums.LanguageIt),
	}
	vocabularyIDs := make([]uuid.UUID, 0, len(vocabularies))
	for _, vocabulary := range vocabularies {
		vocabularyIDs = append(vocabularyIDs, vocabulary.ID)
	}

	exercise := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusPending, vocabularyIDs)
	require.NoError(t, services.StartMatchExercise(exercise.ID, messageID, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))

	for callbackIndex, tappedCard := range []int{0, 2} {
		update := map[string]any{
			"update_id": 50 + callbackIndex,
			"callback_query": map[string]any{
				"id":   "cb-match-wrong-" + string(rune('1'+callbackIndex)),
				"data": "exercise:mt:" + telegramCompactUUID(exercise.ID) + ":" + strconv.Itoa(tappedCard),
				"from": map[string]any{
					"id":     telegramID,
					"is_bot": false,
				},
				"message": map[string]any{
					"message_id": messageID,
					"chat": map[string]any{
						"id":   telegramID,
						"type": "private",
					},
				},
			},
		}

		rec := telegramUpdate(t, update)
		testkit.RequireStatus(t, rec, http.StatusOK)
	}

	editRequests := tg.RequestsFor("editMessageText")
	require.Len(t, editRequests, 2)

	answerRequests := tg.RequestsFor("answerCallbackQuery")
	require.Len(t, answerRequests, 2)
	var wrongAnswer map[string]any
	require.NoError(t, json.Unmarshal(answerRequests[1].Body, &wrongAnswer))
	assert.Equal(t, telegram.GetBotTexts(enums.LanguageRu).MatchNotAMatchToast, wrongAnswer["text"])

	var edited map[string]any
	require.NoError(t, json.Unmarshal(editRequests[1].Body, &edited))
	replyMarkup, ok := edited["reply_markup"].(map[string]any)
	require.True(t, ok)
	keyboard, ok := replyMarkup["inline_keyboard"].([]any)
	require.True(t, ok)
	require.Len(t, keyboard, 5)

	firstRow, ok := keyboard[0].([]any)
	require.True(t, ok)
	secondRow, ok := keyboard[1].([]any)
	require.True(t, ok)
	require.NotEmpty(t, firstRow)
	require.NotEmpty(t, secondRow)

	firstButton, ok := firstRow[0].(map[string]any)
	require.True(t, ok)
	secondButton, ok := secondRow[0].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, firstButton["text"], "⚠️ ")
	assert.Contains(t, secondButton["text"], "⚠️ ")
}

func TestTelegramWebhookMatchTapRecoversPendingMessage(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555009
	const messageID int64 = 100
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))

	vocabularies := []models.Vocabulary{
		exerciseSeedVocabulary(t, user.ID, "release", "rilasciare", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "cell", "la cella", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "sentence", "la condanna", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "prison", "la prigione", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "guard", "la guardia", enums.LanguageEn, enums.LanguageIt),
	}
	vocabularyIDs := make([]uuid.UUID, 0, len(vocabularies))
	for _, vocabulary := range vocabularies {
		vocabularyIDs = append(vocabularyIDs, vocabulary.ID)
	}

	exercise := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusPending, vocabularyIDs)
	compactExerciseID := telegramCompactUUID(exercise.ID)
	inlineKeyboard := [][]map[string]string{
		{
			{"text": "release", "callback_data": "exercise:mt:" + compactExerciseID + ":0"},
			{"text": "rilasciare", "callback_data": "exercise:mt:" + compactExerciseID + ":1"},
		},
		{
			{"text": "cell", "callback_data": "exercise:mt:" + compactExerciseID + ":2"},
			{"text": "la cella", "callback_data": "exercise:mt:" + compactExerciseID + ":3"},
		},
		{
			{"text": "sentence", "callback_data": "exercise:mt:" + compactExerciseID + ":4"},
			{"text": "la condanna", "callback_data": "exercise:mt:" + compactExerciseID + ":5"},
		},
		{
			{"text": "prison", "callback_data": "exercise:mt:" + compactExerciseID + ":6"},
			{"text": "la prigione", "callback_data": "exercise:mt:" + compactExerciseID + ":7"},
		},
		{
			{"text": "guard", "callback_data": "exercise:mt:" + compactExerciseID + ":8"},
			{"text": "la guardia", "callback_data": "exercise:mt:" + compactExerciseID + ":9"},
		},
	}

	update := map[string]any{
		"update_id": 6,
		"callback_query": map[string]any{
			"id":   "cb-match-2",
			"data": "exercise:mt:" + compactExerciseID + ":0",
			"from": map[string]any{
				"id":     telegramID,
				"is_bot": false,
			},
			"message": map[string]any{
				"message_id": messageID,
				"chat": map[string]any{
					"id":   telegramID,
					"type": "private",
				},
				"reply_markup": map[string]any{
					"inline_keyboard": inlineKeyboard,
				},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	require.True(t, tg.Sent("answerCallbackQuery"))
	require.True(t, tg.Sent("editMessageText"), "pending match callback should be recovered and re-rendered")

	var refreshed models.Exercise
	require.NoError(t, db.DB.Where("id = ?", exercise.ID).First(&refreshed).Error)
	assert.Equal(t, enums.ExerciseStatusInProgress, refreshed.Status)
	require.NotNil(t, refreshed.TelegramMessageID)
	assert.EqualValues(t, messageID, *refreshed.TelegramMessageID)
	require.NotNil(t, refreshed.MatchState)
	var matchState struct {
		Order   []int `json:"order"`
		Pending int   `json:"pending"`
	}
	require.NoError(t, json.Unmarshal([]byte(*refreshed.MatchState), &matchState))
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, matchState.Order)
	assert.Equal(t, 0, matchState.Pending)
}

// -----------------------------------------------------------------------------
// MyChatMember updates
// -----------------------------------------------------------------------------

func TestTelegramWebhookBlockBotDisablesUser(t *testing.T) {
	testkit.Truncate(t)
	testkit.MockTelegramAPI(t)

	const telegramID int64 = 555006

	// Seed an enabled user.
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID),
		testkit.WithSettings(models.UserSettings{
			Telegram: models.UserTelegramSettings{BotEnabled: true},
		}))

	update := telegramMyChatMember(telegramID, telegram.Member, telegram.Kicked)

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var refreshed models.User
	require.NoError(t, db.DB.Where("id = ?", user.ID).First(&refreshed).Error)
	assert.False(t, refreshed.Settings.Telegram.BotEnabled, "blocking the bot should disable it")
}

func TestTelegramWebhookUnblockBotEnablesUser(t *testing.T) {
	testkit.Truncate(t)
	testkit.MockTelegramAPI(t)

	const telegramID int64 = 555007

	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID),
		testkit.WithSettings(models.UserSettings{
			Telegram: models.UserTelegramSettings{BotEnabled: false},
		}))

	update := telegramMyChatMember(telegramID, telegram.Kicked, telegram.Member)

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var refreshed models.User
	require.NoError(t, db.DB.Where("id = ?", user.ID).First(&refreshed).Error)
	assert.True(t, refreshed.Settings.Telegram.BotEnabled, "unblocking the bot should enable it")
}

// A text reply to a match-pairs board must be rejected with a hint while the
// board keyboard stays intact. Previously the reply stripped the keyboard and
// then failed verification silently, leaving the exercise stuck until expiry.
func TestTelegramWebhookMatchPairsTextReplyKeepsBoard(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555021
	const messageID int64 = 110
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))

	vocabularyIDs := make([]uuid.UUID, 0, services.MatchPairsVocabularyCount)
	for index := 0; index < services.MatchPairsVocabularyCount; index++ {
		vocabulary := exerciseSeedVocabulary(
			t, user.ID,
			"original-"+strconv.Itoa(index), "translation-"+strconv.Itoa(index),
			enums.LanguageEn, enums.LanguageIt,
		)
		vocabularyIDs = append(vocabularyIDs, vocabulary.ID)
	}
	exercise := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusPending, vocabularyIDs)
	require.NoError(t, services.StartMatchExercise(exercise.ID, messageID, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))

	update := telegramPrivateMessage(telegramID, "rilasciare")
	update["update_id"] = 91
	update["message"].(map[string]any)["reply_to_message"] = map[string]any{
		"message_id": messageID,
		"chat":       map[string]any{"id": telegramID, "type": "private"},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)

	assert.Empty(
		t,
		tg.RequestsFor("editMessageReplyMarkup"),
		"a text reply must not strip the match board keyboard",
	)
	replies := tg.RequestsFor("sendMessage")
	require.Len(t, replies, 1, "the user must get an explicit use-buttons hint")
	assert.Contains(t, string(replies[0].Body), "кнопок")

	stored := exerciseReload(t, exercise.ID)
	assert.Equal(t, enums.ExerciseStatusInProgress, stored.Status, "the exercise must stay answerable via its board")
	for _, vocabularyID := range vocabularyIDs {
		link := exerciseLink(t, exercise.ID, vocabularyID)
		assert.Nil(t, link.Result, "a text reply must not score the exercise")
	}
}

// telegramMyChatMember builds a my_chat_member update transitioning between the
// given old/new statuses in a private chat.
func telegramMyChatMember(telegramID int64, oldStatus, newStatus string) map[string]any {
	return map[string]any{
		"update_id": 4,
		"my_chat_member": map[string]any{
			"chat": map[string]any{
				"id":   telegramID,
				"type": "private",
			},
			"from": map[string]any{
				"id":     telegramID,
				"is_bot": false,
			},
			"old_chat_member": map[string]any{
				"status": oldStatus,
				"user":   map[string]any{"id": telegramID, "is_bot": false},
			},
			"new_chat_member": map[string]any{
				"status": newStatus,
				"user":   map[string]any{"id": telegramID, "is_bot": false},
			},
		},
	}
}
