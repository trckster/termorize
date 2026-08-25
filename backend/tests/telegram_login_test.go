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

func telegramLoginAuthCookie(rec *http.Response) *http.Cookie {
	for _, cookie := range rec.Cookies() {
		if cookie.Name == testkit.AuthCookieName {
			return cookie
		}
	}
	return nil
}

// TestTelegramLoginOAuthCodeSuccess drives the authorization-code branch of the
// callback end-to-end against the faked Telegram OAuth endpoints, asserting a
// user is created, the auth cookie is issued, and the response carries the user.
func TestTelegramLoginOAuthCodeSuccess(t *testing.T) {
	testkit.Truncate(t)

	profile := testkit.TelegramLoginProfile{ID: 5551234, Username: "ada_lovelace", Name: "Ada Lovelace"}
	telegramOAuth := testkit.MockTelegramLogin(t, profile)

	session, err := auth.NewTelegramLoginSession()
	require.NoError(t, err)
	state, err := auth.IssueTelegramLoginSessionToken(*session)
	require.NoError(t, err)

	rec := testkit.RequestWithHeaders(
		t,
		http.MethodPost,
		"/api/telegram/login/callback",
		map[string]any{
			"code":  "any-authorization-code",
			"state": state,
		},
		http.Header{"X-Timezone": []string{"Europe/Rome"}},
	)

	testkit.RequireStatus(t, rec, http.StatusOK)

	authCookie := telegramLoginAuthCookie(rec.Result())
	require.NotNil(t, authCookie, "expected an auth cookie to be set")
	assert.NotEmpty(t, authCookie.Value)
	assert.True(t, authCookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, authCookie.SameSite)

	tokenRequests := telegramOAuth.TokenRequests()
	require.Len(t, tokenRequests, 1)
	assert.Equal(t, "authorization_code", tokenRequests[0].Form.Get("grant_type"))
	assert.Equal(t, "any-authorization-code", tokenRequests[0].Form.Get("code"))
	assert.Equal(t, session.CodeVerifier, tokenRequests[0].Form.Get("code_verifier"))
	assert.Equal(t, "test-telegram-client-id", tokenRequests[0].Form.Get("client_id"))
	assert.NotEmpty(t, tokenRequests[0].Form.Get("redirect_uri"))
	assert.Equal(t, "test-telegram-client-id", tokenRequests[0].BasicUsername)
	assert.Equal(t, "test-telegram-client-secret", tokenRequests[0].BasicPassword)
	assert.Equal(t, "application/x-www-form-urlencoded", tokenRequests[0].ContentType)
	assert.Equal(t, 1, telegramOAuth.JWKSRequestCount())

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
	assert.Equal(t, "Europe/Rome", stored.Settings.TimeZone)

	meRec := testkit.Request(t, http.MethodGet, "/api/me", nil, authCookie)
	testkit.RequireStatus(t, meRec, http.StatusOK)
	var me models.User
	testkit.DecodeJSON(t, meRec, &me)
	assert.Equal(t, stored.ID, me.ID)
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

	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestTelegramLoginOAuthFailureDoesNotCreateSession(t *testing.T) {
	testkit.Truncate(t)

	telegramOAuth := testkit.MockTelegramLogin(t, testkit.TelegramLoginProfile{ID: 5559999})
	telegramOAuth.FailTokenExchange(http.StatusBadGateway)

	session, err := auth.NewTelegramLoginSession()
	require.NoError(t, err)
	state, err := auth.IssueTelegramLoginSessionToken(*session)
	require.NoError(t, err)

	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", map[string]any{
		"code":  "rejected-code",
		"state": state,
	})
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)

	var body struct {
		Error string `json:"error"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "telegram login failed", body.Error)
	assert.Nil(t, telegramLoginAuthCookie(rec.Result()))
	assert.Len(t, telegramOAuth.TokenRequests(), 1)
	assert.Equal(t, 0, telegramOAuth.JWKSRequestCount())

	var userCount int64
	require.NoError(t, db.DB.Model(&models.User{}).Count(&userCount).Error)
	assert.Equal(t, int64(0), userCount)
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

	testkit.RequireStatus(t, rec, http.StatusOK)
	authCookie := telegramLoginAuthCookie(rec.Result())
	require.NotNil(t, authCookie, "expected an auth cookie to be set")
	assert.NotEmpty(t, authCookie.Value)

	var body models.User
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "grace_hopper", body.Username)
	assert.Equal(t, "Grace", body.Name)

	var stored models.User
	require.NoError(t, db.DB.Where("telegram_id = ?", telegramID).First(&stored).Error)
	assert.Equal(t, "grace_hopper", stored.Username)
}

func TestTelegramLoginRejectsDeletedUser(t *testing.T) {
	testkit.Truncate(t)

	const telegramID int64 = 9090910
	user := testkit.CreateUser(t,
		testkit.WithTelegramID(telegramID),
		testkit.WithUsername("deleted_user"),
	)
	require.NoError(t, db.DB.Delete(&user).Error)

	initData := testkit.BuildTelegramInitData(telegramID, "deleted_user", "Deleted")
	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", map[string]any{
		"init_data": initData,
	})

	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
	assert.Nil(t, telegramLoginAuthCookie(rec.Result()))

	var storedUsers []models.User
	require.NoError(t, db.DB.Unscoped().Where("telegram_id = ?", telegramID).Find(&storedUsers).Error)
	require.Len(t, storedUsers, 1)
	assert.Equal(t, user.ID, storedUsers[0].ID)
	assert.True(t, storedUsers[0].DeletedAt.Valid)
}

// TestTelegramLoginInitDataTamperedRejected ensures a tampered init_data payload
// (valid signature, then mutated) fails HMAC validation.
func TestTelegramLoginInitDataTamperedRejected(t *testing.T) {
	testkit.Truncate(t)

	initData := testkit.BuildTelegramInitData(123, "user", "User") + "&extra=tampered"

	rec := testkit.Request(t, http.MethodPost, "/api/telegram/login/callback", map[string]any{
		"init_data": initData,
	})

	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}
