package tests

import (
	"bytes"
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

	req := httptest.NewRequest(http.MethodPost, telegramWebhookPath, bytes.NewReader([]byte(rawBody)))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set(telegramSecretHeader, secret)
	}

	rec := httptest.NewRecorder()
	testkit.Router().ServeHTTP(rec, req)
	return rec
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

func TestTelegramWebhookMissingSecret(t *testing.T) {
	testkit.Truncate(t)
	testkit.MockTelegramAPI(t)

	rec := telegramRequest(t, `{"update_id":1}`, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestTelegramWebhookWrongSecret(t *testing.T) {
	testkit.Truncate(t)
	testkit.MockTelegramAPI(t)

	rec := telegramRequest(t, `{"update_id":1}`, "definitely-not-the-secret")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestTelegramWebhookCorrectSecretPassesMiddleware(t *testing.T) {
	testkit.Truncate(t)
	testkit.MockTelegramAPI(t)

	// An empty update (no message/callback/chat_member) is a valid no-op; it proves
	// the middleware let the request through with the computed secret.
	rec := telegramRequest(t, `{"update_id":1}`, telegram.BuildWebhookSecret())
	assert.Equal(t, http.StatusOK, rec.Code)
}

// -----------------------------------------------------------------------------
// Payload handling
// -----------------------------------------------------------------------------

func TestTelegramWebhookMalformedBody(t *testing.T) {
	testkit.Truncate(t)
	testkit.MockTelegramAPI(t)

	rec := telegramRequest(t, "this is not json", telegram.BuildWebhookSecret())
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid payload", body["error"])
}

func TestTelegramWebhookUnknownUpdateNoOp(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	// A well-formed update with none of message/callback_query/my_chat_member set.
	rec := telegramUpdate(t, map[string]any{"update_id": 42})
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, tg.Requests(), "no outbound telegram call expected for an empty update")
}

// -----------------------------------------------------------------------------
// Message updates
// -----------------------------------------------------------------------------

func TestTelegramWebhookStartCommandCreatesUserAndReplies(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)

	const telegramID int64 = 555001

	rec := telegramUpdate(t, telegramPrivateMessage(telegramID, "/start"))
	require.Equal(t, http.StatusOK, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code)

	// DB side effect: the user's telegram state moves to deletingVocabulary.
	var refreshed models.User
	require.NoError(t, db.DB.Where("id = ?", user.ID).First(&refreshed).Error)
	assert.Equal(t, enums.TelegramStateDeletingVocabulary, refreshed.TelegramState)

	require.True(t, tg.Sent("answerCallbackQuery"))
	require.True(t, tg.Sent("editMessageText"))
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
	require.Equal(t, http.StatusOK, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code)
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
	require.Equal(t, http.StatusOK, rec.Code)
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
		require.Equal(t, http.StatusOK, rec.Code)
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
	require.Equal(t, http.StatusOK, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code)

	var refreshed models.User
	require.NoError(t, db.DB.Where("id = ?", user.ID).First(&refreshed).Error)
	assert.True(t, refreshed.Settings.Telegram.BotEnabled, "unblocking the bot should enable it")
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
