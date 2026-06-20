package tests

import (
	"net/http"
	"testing"

	"termorize/src/auth"
	"termorize/src/data/db"
	"termorize/src/models"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// telegramLoginAuthCookie returns the value of the auth cookie set on the
// response, or "" if none was set.
func telegramLoginAuthCookie(rec *http.Response) string {
	for _, cookie := range rec.Cookies() {
		if cookie.Name == testkit.AuthCookieName {
			return cookie.Value
		}
	}
	return ""
}

// TestTelegramLoginOAuthCodeSuccess drives the authorization-code branch of the
// callback end-to-end against the faked Telegram OAuth endpoints, asserting a
// user is created, the auth cookie is issued, and the response carries the user.
func TestTelegramLoginOAuthCodeSuccess(t *testing.T) {
	testkit.Truncate(t)

	profile := testkit.TelegramLoginProfile{ID: 5551234, Username: "ada_lovelace", Name: "Ada Lovelace"}
	testkit.MockTelegramLogin(t, profile)

	session, err := auth.NewTelegramLoginSession()
	require.NoError(t, err)
	state, err := auth.IssueTelegramLoginSessionToken(*session)
	require.NoError(t, err)

	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", map[string]any{
		"code":  "any-authorization-code",
		"state": state,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	// Auth cookie issued.
	assert.NotEmpty(t, telegramLoginAuthCookie(rec.Result()), "expected an auth cookie to be set")

	// Response carries the logged-in user.
	var body models.User
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "ada_lovelace", body.Username)
	assert.Equal(t, "Ada Lovelace", body.Name)
	assert.NotZero(t, body.ID)

	// User persisted with the Telegram id from the id_token.
	var stored models.User
	require.NoError(t, db.DB.Where("telegram_id = ?", profile.ID).First(&stored).Error)
	assert.Equal(t, "ada_lovelace", stored.Username)
}

// TestTelegramLoginOAuthCodeInvalidState ensures an undecodable state is rejected
// before any (faked) network call.
func TestTelegramLoginOAuthCodeInvalidState(t *testing.T) {
	testkit.Truncate(t)
	testkit.MockTelegramLogin(t, testkit.TelegramLoginProfile{ID: 1})

	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", map[string]any{
		"code":  "any-code",
		"state": "not-a-valid-jwt",
	})

	require.Equal(t, http.StatusUnauthorized, rec.Code, "body=%s", rec.Body.String())
}

// TestTelegramLoginInitDataSuccess drives the WebApp init_data branch, which is
// validated locally (HMAC over the bot token) with no network call.
func TestTelegramLoginInitDataSuccess(t *testing.T) {
	testkit.Truncate(t)

	const telegramID int64 = 9090909
	initData := testkit.BuildTelegramInitData(telegramID, "grace_hopper", "Grace")

	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", map[string]any{
		"init_data": initData,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.NotEmpty(t, telegramLoginAuthCookie(rec.Result()), "expected an auth cookie to be set")

	var body models.User
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "grace_hopper", body.Username)
	assert.Equal(t, "Grace", body.Name)

	var stored models.User
	require.NoError(t, db.DB.Where("telegram_id = ?", telegramID).First(&stored).Error)
	assert.Equal(t, "grace_hopper", stored.Username)
}

// TestTelegramLoginInitDataTamperedRejected ensures a tampered init_data payload
// (valid signature, then mutated) fails HMAC validation.
func TestTelegramLoginInitDataTamperedRejected(t *testing.T) {
	testkit.Truncate(t)

	initData := testkit.BuildTelegramInitData(123, "user", "User") + "&extra=tampered"

	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", map[string]any{
		"init_data": initData,
	})

	require.Equal(t, http.StatusUnauthorized, rec.Code, "body=%s", rec.Body.String())
}
