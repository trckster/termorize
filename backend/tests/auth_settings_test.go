package tests

import (
	"net/http"
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
	testkit.RequireStatus(t, rec, http.StatusOK)

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
	rec := testkit.RequestWithHeaders(
		t,
		http.MethodPost,
		"/api/telegram/login/start",
		nil,
		http.Header{"Origin": []string{"https://app.example.com"}},
	)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	authURL, ok := body["auth_url"].(string)
	require.True(t, ok)
	// redirect_uri is URL-encoded inside the query string.
	assert.Contains(t, authURL, "redirect_uri=https%3A%2F%2Fapp.example.com%2Flogin%2Ftelegram%2Fcallback")
}

// ---------------------------------------------------------------------------
// POST /api/telegram/login/callback (CompleteTelegramLogin) — public.
// Local rejection paths live here; successful OAuth and WebApp login protocol
// tests use loopback Telegram endpoints in telegram_login_test.go.
// ---------------------------------------------------------------------------

func TestLoginCallbackInvalidJSON(t *testing.T) {
	testkit.Truncate(t)

	// Send a raw non-JSON body to trigger the bind error (non-validation).
	rec := testkit.RawRequest(t, http.MethodPost, "/api/telegram/login/callback", "not-json", nil)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	// Non-validation bind errors are reported under "error".
	assert.Contains(t, body, "error")
}

func TestLoginCallbackRejectsInvalidLocalPayload(t *testing.T) {
	tests := []struct {
		name        string
		payload     map[string]any
		wantStatus  int
		wantError   string
		wantDetails bool
	}{
		{
			name:       "empty payload",
			payload:    map[string]any{},
			wantStatus: http.StatusBadRequest,
			wantError:  "telegram login payload is invalid",
		},
		{
			name:       "authorization code without state",
			payload:    map[string]any{"code": "some-code"},
			wantStatus: http.StatusBadRequest,
			wantError:  "telegram login payload is invalid",
		},
		{
			name: "invalid state token",
			payload: map[string]any{
				"code":  "some-code",
				"state": "not-a-valid-jwt",
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "telegram login session is invalid",
		},
		{
			name: "invalid WebApp signature",
			payload: map[string]any{
				"init_data": "auth_date=1700000000&hash=deadbeef&user=%7B%22id%22%3A1%7D",
			},
			wantStatus:  http.StatusUnauthorized,
			wantError:   "telegram login failed",
			wantDetails: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.Truncate(t)

			rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", tt.payload)
			testkit.RequireStatus(t, rec, tt.wantStatus)

			var body struct {
				Error   string `json:"error"`
				Details string `json:"details"`
			}
			testkit.DecodeJSON(t, rec, &body)
			assert.Equal(t, tt.wantError, body.Error)
			if tt.wantDetails {
				assert.NotEmpty(t, body.Details)
			} else {
				assert.Empty(t, body.Details)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// POST /api/logout (Logout) — public; clears the auth cookie.
// ---------------------------------------------------------------------------

func TestLogout(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/logout", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

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
	testkit.RequireStatus(t, rec, http.StatusOK)

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
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestMeReturnsSettings(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t,
		testkit.WithName("Grace"),
		testkit.WithUsername("grace"),
	)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/me", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

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
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestUpdateSettingsHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t, testkit.WithName("Ada"))

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", authSettingsValidPayload())
	testkit.RequireStatus(t, rec, http.StatusOK)

	var got models.User
	testkit.DecodeJSON(t, rec, &got)
	assert.Equal(t, user.ID, got.ID)
	assert.Equal(t, enums.Language("en"), got.Settings.SystemLanguage)
	assert.Equal(t, enums.Language("de"), got.Settings.MainLearningLanguage)
	assert.Equal(t, enums.Language("en"), got.Settings.TranslationSourceLanguage)
	assert.Equal(t, enums.Language("ru"), got.Settings.TranslationTargetLanguage)
	assert.NotNil(t, got.Settings.IgnoredAudioLanguages)
	assert.Empty(t, got.Settings.IgnoredAudioLanguages)
	assert.Equal(t, "Europe/Berlin", got.Settings.TimeZone)
	assert.True(t, got.Settings.Telegram.DailyQuestionsEnabled)
	assert.Equal(t, uint(3), got.Settings.Telegram.DailyQuestionsCount)
	require.Len(t, got.Settings.Telegram.DailyQuestionsSchedule, 1)
	assert.Equal(t, "09:00", got.Settings.Telegram.DailyQuestionsSchedule[0].From)
	assert.Equal(t, "18:00", got.Settings.Telegram.DailyQuestionsSchedule[0].To)

	// Verify persistence via a fresh /api/me read.
	meRec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/me", nil)
	testkit.RequireStatus(t, meRec, http.StatusOK)
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

	rec := testkit.AuthedRawRequest(t, user, http.MethodPut, "/api/settings", "}{ not json", nil)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Contains(t, body, "error")
}

func TestUpdateSettingsMissingRequiredFields(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", map[string]any{})
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	require.NotEmpty(t, body.Errors)
	assert.Equal(t, "required", body.Errors["SystemLanguage"])
}

func TestUpdateSettingsRejectsInvalidFieldValues(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(map[string]any)
		wantField string
		wantTag   string
	}{
		{
			name: "unknown system language",
			mutate: func(payload map[string]any) {
				payload["system_language"] = "xx"
			},
			wantField: "SystemLanguage",
			wantTag:   "enum",
		},
		{
			name: "unknown timezone",
			mutate: func(payload map[string]any) {
				payload["time_zone"] = "Not/AZone"
			},
			wantField: "TimeZone",
			wantTag:   "timezone",
		},
		{
			name: "daily question count above maximum",
			mutate: func(payload map[string]any) {
				payload["telegram"].(map[string]any)["daily_questions_count"] = 101
			},
			wantField: "DailyQuestionsCount",
			wantTag:   "max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.Truncate(t)
			user := testkit.CreateUser(t)
			payload := authSettingsValidPayload()
			tt.mutate(payload)

			rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
			testkit.RequireStatus(t, rec, http.StatusBadRequest)

			var body struct {
				Errors map[string]string `json:"errors"`
			}
			testkit.DecodeJSON(t, rec, &body)
			assert.Equal(t, tt.wantTag, body.Errors[tt.wantField])
		})
	}
}

func TestUpdateSettingsSameSourceAndTargetLanguage(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	payload := authSettingsValidPayload()
	payload["translation_source_language"] = "en"
	payload["translation_target_language"] = "en" // violates nefield constraint

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

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
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	require.NotEmpty(t, body.Errors)
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
	testkit.RequireStatus(t, rec, http.StatusOK)

	var got models.User
	testkit.DecodeJSON(t, rec, &got)
	assert.True(t, got.Settings.Telegram.BotEnabled, "BotEnabled must be preserved across settings update")
}
