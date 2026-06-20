package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/telegram"
	"termorize/src/models"
	"termorize/src/testkit"

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
