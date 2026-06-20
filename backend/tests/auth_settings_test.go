package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authSettingsValidPayload returns a fully-valid UpdateSettings request body.
func authSettingsValidPayload() map[string]any {
	return map[string]any{
		"system_language":             "en",
		"main_learning_language":      "de",
		"translation_source_language": "en",
		"translation_target_language": "ru",
		"time_zone":                   "Europe/Berlin",
		"telegram": map[string]any{
			"daily_questions_enabled": true,
			"daily_questions_count":   3,
			"daily_questions_schedule": []map[string]any{
				{"from": "09:00", "to": "18:00"},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// POST /api/telegram/login/start (StartTelegramLogin) — public, network-free.
// It only validates config, builds a session token and a login URL; no HTTP
// to Telegram is made, so the happy path is testable.
// ---------------------------------------------------------------------------

func TestLoginStartReturnsAuthURL(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/start", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)

	authURL, ok := body["auth_url"].(string)
	require.True(t, ok, "auth_url should be a string, body=%v", body)
	assert.Contains(t, authURL, "https://oauth.telegram.org/auth")
	// The URL is built from the configured client id and carries the PKCE bits.
	assert.Contains(t, authURL, "client_id=test-telegram-client-id")
	assert.Contains(t, authURL, "code_challenge_method=S256")
	assert.Contains(t, authURL, "response_type=code")
	assert.Contains(t, authURL, "state=")
}

func TestLoginStartUsesOriginRedirect(t *testing.T) {
	testkit.Truncate(t)

	// Send an Origin header to exercise the per-origin redirect URL branch.
	rec := buildRequestWithHeader(t, http.MethodPost, "/api/telegram/login/start",
		"Origin", "https://app.example.com")
	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	authURL, ok := body["auth_url"].(string)
	require.True(t, ok)
	// redirect_uri is URL-encoded inside the query string.
	assert.Contains(t, authURL, "redirect_uri=https%3A%2F%2Fapp.example.com%2Flogin%2Ftelegram%2Fcallback")
}

// ---------------------------------------------------------------------------
// POST /api/telegram/login/callback (CompleteTelegramLogin) — public.
// Only the pre-network code paths are exercised; a successful login requires
// real Telegram HTTP (ExchangeTelegramLoginCode) and so is out of scope.
// ---------------------------------------------------------------------------

func TestLoginCallbackInvalidJSON(t *testing.T) {
	testkit.Truncate(t)

	// Send a raw non-JSON body to trigger the bind error (non-validation).
	rec := buildRawRequest(t, http.MethodPost, "/api/telegram/login/callback", "not-json")
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	// Non-validation bind errors are reported under "error".
	assert.Contains(t, body, "error")
}

func TestLoginCallbackEmptyPayload(t *testing.T) {
	testkit.Truncate(t)

	// Valid JSON, but no code/state/init_data => 400 before any network call.
	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "telegram login payload is invalid", body["error"])
}

func TestLoginCallbackMissingState(t *testing.T) {
	testkit.Truncate(t)

	// Code present but no state => still 400 before network.
	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", map[string]any{
		"code": "some-code",
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "telegram login payload is invalid", body["error"])
}

func TestLoginCallbackInvalidStateToken(t *testing.T) {
	testkit.Truncate(t)

	// Code + state present, but the state token is not a valid session JWT, so
	// DecodeTelegramLoginSessionToken fails => 401 before any network call.
	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", map[string]any{
		"code":  "some-code",
		"state": "not-a-valid-jwt",
	})
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "telegram login session is invalid", body["error"])
}

func TestLoginCallbackInvalidInitData(t *testing.T) {
	testkit.Truncate(t)

	// init_data is present but its hash is invalid; ValidateTelegramInitData
	// fails locally (HMAC check) without any network call => 401.
	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", map[string]any{
		"init_data": "auth_date=1700000000&hash=deadbeef&user=%7B%22id%22%3A1%7D",
	})
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "telegram login failed", body["error"])
	assert.Contains(t, body, "details")
}

// ---------------------------------------------------------------------------
// POST /api/logout (Logout) — public; clears the auth cookie.
// ---------------------------------------------------------------------------

func TestLogout(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/logout", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	// The handler deletes the auth cookie: expect a Set-Cookie clearing "auth".
	var authCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "auth" {
			authCookie = c
		}
	}
	require.NotNil(t, authCookie, "logout should set an auth cookie")
	assert.Empty(t, authCookie.Value)
	assert.True(t, authCookie.MaxAge < 0, "auth cookie should be expired")
}

// ---------------------------------------------------------------------------
// GET /api/settings (GetSettings) — public; returns the language list.
// ---------------------------------------------------------------------------

func TestGetSettingsPublic(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/settings", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Languages []string `json:"languages"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, enums.AllLanguages(), body.Languages)
	assert.Contains(t, body.Languages, "en")
	assert.Contains(t, body.Languages, "ru")
}

// ---------------------------------------------------------------------------
// GET /api/me (Me) — protected. Extra coverage beyond the poc tests:
// a stale JWT for a deleted user must be rejected with 401.
// ---------------------------------------------------------------------------

func TestMeRejectsDeletedUser(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t, testkit.WithName("Ghost"))
	cookie := testkit.AuthCookie(user)

	// Remove the user but keep the (still valid) cookie.
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/me", nil, cookie)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMeReturnsSettings(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t,
		testkit.WithName("Grace"),
		testkit.WithUsername("grace"),
	)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/me", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var got models.User
	testkit.DecodeJSON(t, rec, &got)
	assert.Equal(t, user.ID, got.ID)
	assert.Equal(t, "grace", got.Username)
	assert.Equal(t, "Grace", got.Name)
	// TelegramID is intentionally omitted from the JSON (json:"-").
	assert.Equal(t, int64(0), got.TelegramID)
	assert.False(t, got.IsAdmin)
}

// ---------------------------------------------------------------------------
// PUT /api/settings (UpdateSettings) — protected.
// ---------------------------------------------------------------------------

func TestUpdateSettingsRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPut, "/api/settings", authSettingsValidPayload())
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateSettingsHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t, testkit.WithName("Ada"))

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", authSettingsValidPayload())
	require.Equal(t, http.StatusOK, rec.Code)

	var got models.User
	testkit.DecodeJSON(t, rec, &got)
	assert.Equal(t, user.ID, got.ID)
	assert.Equal(t, enums.Language("en"), got.Settings.SystemLanguage)
	assert.Equal(t, enums.Language("de"), got.Settings.MainLearningLanguage)
	assert.Equal(t, enums.Language("en"), got.Settings.TranslationSourceLanguage)
	assert.Equal(t, enums.Language("ru"), got.Settings.TranslationTargetLanguage)
	assert.Equal(t, "Europe/Berlin", got.Settings.TimeZone)
	assert.True(t, got.Settings.Telegram.DailyQuestionsEnabled)
	assert.Equal(t, uint(3), got.Settings.Telegram.DailyQuestionsCount)
	require.Len(t, got.Settings.Telegram.DailyQuestionsSchedule, 1)
	assert.Equal(t, "09:00", got.Settings.Telegram.DailyQuestionsSchedule[0].From)
	assert.Equal(t, "18:00", got.Settings.Telegram.DailyQuestionsSchedule[0].To)

	// Verify persistence via a fresh /api/me read.
	meRec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/me", nil)
	require.Equal(t, http.StatusOK, meRec.Code)
	var me models.User
	testkit.DecodeJSON(t, meRec, &me)
	assert.Equal(t, enums.Language("de"), me.Settings.MainLearningLanguage)
	assert.Equal(t, "Europe/Berlin", me.Settings.TimeZone)
	require.Len(t, me.Settings.Telegram.DailyQuestionsSchedule, 1)
	assert.Equal(t, "09:00", me.Settings.Telegram.DailyQuestionsSchedule[0].From)
}

func TestUpdateSettingsInvalidJSON(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := buildAuthedRawRequest(t, user, http.MethodPut, "/api/settings", "}{ not json")
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Contains(t, body, "error")
}

func TestUpdateSettingsMissingRequiredFields(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	require.NotEmpty(t, body.Errors)
	assert.Equal(t, "required", body.Errors["SystemLanguage"])
}

func TestUpdateSettingsInvalidLanguageEnum(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	payload := authSettingsValidPayload()
	payload["system_language"] = "xx" // not a valid Language enum value

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "enum", body.Errors["SystemLanguage"])
}

func TestUpdateSettingsInvalidTimezone(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	payload := authSettingsValidPayload()
	payload["time_zone"] = "Not/AZone"

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "timezone", body.Errors["TimeZone"])
}

func TestUpdateSettingsSameSourceAndTargetLanguage(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	payload := authSettingsValidPayload()
	payload["translation_source_language"] = "en"
	payload["translation_target_language"] = "en" // violates nefield constraint

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	require.NotEmpty(t, body.Errors)
	// Either of the two mutually-exclusive fields fails the nefield check.
	_, sourceFailed := body.Errors["TranslationSourceLanguage"]
	_, targetFailed := body.Errors["TranslationTargetLanguage"]
	assert.True(t, sourceFailed || targetFailed, "expected nefield violation, got %v", body.Errors)
}

func TestUpdateSettingsInvalidScheduleTime(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	payload := authSettingsValidPayload()
	payload["telegram"].(map[string]any)["daily_questions_schedule"] = []map[string]any{
		{"from": "9am", "to": "25:00"}, // invalid hhmm values
	}

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	require.NotEmpty(t, body.Errors)
}

func TestUpdateSettingsDailyQuestionsCountTooHigh(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	payload := authSettingsValidPayload()
	payload["telegram"].(map[string]any)["daily_questions_count"] = 101 // max=100

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "max", body.Errors["DailyQuestionsCount"])
}

func TestUpdateSettingsPreservesBotEnabled(t *testing.T) {
	testkit.Truncate(t)

	// Seed a user whose bot is enabled; UpdateUserSettings must keep BotEnabled
	// regardless of the request payload (it is server-managed).
	settings := models.UserSettings{
		TimeZone: "UTC",
		Telegram: models.UserTelegramSettings{BotEnabled: true},
	}
	user := testkit.CreateUser(t, testkit.WithSettings(settings))

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", authSettingsValidPayload())
	require.Equal(t, http.StatusOK, rec.Code)

	var got models.User
	testkit.DecodeJSON(t, rec, &got)
	assert.True(t, got.Settings.Telegram.BotEnabled, "BotEnabled must be preserved across settings update")
}

// ---------------------------------------------------------------------------
// Local request helpers (unexported, prefixed authSettings/build* to avoid
// clashes with other test files in the package).
// ---------------------------------------------------------------------------

// authSettingsRawRequest issues an in-process request against the shared router
// with a raw (possibly invalid) body and/or custom headers — capabilities the
// JSON-encoding testkit.Request helper does not offer.
func authSettingsRawRequest(t *testing.T, method, path, rawBody string, cookies []*http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if rawBody != "" {
		reader = strings.NewReader(rawBody)
	}

	var req *http.Request
	if reader != nil {
		req = httptest.NewRequest(method, path, reader)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, c := range cookies {
		if c != nil {
			req.AddCookie(c)
		}
	}

	rec := httptest.NewRecorder()
	testkit.Router().ServeHTTP(rec, req)
	return rec
}

// buildRawRequest issues an in-process request with a raw (possibly invalid)
// JSON string body so bind-level (non-validation) errors can be exercised.
func buildRawRequest(t *testing.T, method, path, rawBody string) *httptest.ResponseRecorder {
	return authSettingsRawRequest(t, method, path, rawBody, nil, nil)
}

func buildAuthedRawRequest(t *testing.T, user models.User, method, path, rawBody string) *httptest.ResponseRecorder {
	return authSettingsRawRequest(t, method, path, rawBody, []*http.Cookie{testkit.AuthCookie(user)}, nil)
}

func buildRequestWithHeader(t *testing.T, method, path, headerKey, headerValue string) *httptest.ResponseRecorder {
	return authSettingsRawRequest(t, method, path, "", nil, map[string]string{headerKey: headerValue})
}
