package testkit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"termorize/src/auth"
	"termorize/src/data/db"
	"termorize/src/models"
)

// AuthCookieName is the name of the cookie the auth middleware reads. It mirrors
// the (unexported) name used by auth.SetAuthCookie.
const AuthCookieName = "auth"

// UserOption customizes a user created by CreateUser.
type UserOption func(*models.User)

// WithName sets the user's display name.
func WithName(name string) UserOption {
	return func(u *models.User) { u.Name = name }
}

// WithUsername sets the user's username.
func WithUsername(username string) UserOption {
	return func(u *models.User) { u.Username = username }
}

// WithTelegramID sets the user's Telegram ID (defaults to a unique value).
func WithTelegramID(id int64) UserOption {
	return func(u *models.User) { u.TelegramID = id }
}

// WithAdmin marks the user as an admin.
func WithAdmin() UserOption {
	return func(u *models.User) { u.IsAdmin = true }
}

// WithSettings overrides the user's settings.
func WithSettings(settings models.UserSettings) UserOption {
	return func(u *models.User) { u.Settings = settings }
}

var telegramIDSeq int64 = 100000

// CreateUser inserts a user row directly via the shared DB handle and returns it
// (with its generated ID populated). Sensible defaults are applied; override them
// with the With* options.
//
//	user := testkit.CreateUser(t)
//	admin := testkit.CreateUser(t, testkit.WithAdmin(), testkit.WithName("Boss"))
func CreateUser(t *testing.T, opts ...UserOption) models.User {
	t.Helper()

	telegramIDSeq++
	user := models.User{
		Username:   fmt.Sprintf("user_%d", telegramIDSeq),
		Name:       fmt.Sprintf("Test User %d", telegramIDSeq),
		TelegramID: telegramIDSeq,
		// Settings get sensible defaults applied by UserSettings.Value().
	}

	for _, opt := range opts {
		opt(&user)
	}

	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("testkit.CreateUser: failed to insert user: %v", err)
	}

	return user
}

// AuthCookie issues a real JWT for the given user (via auth.IssueJWT) and returns
// a cookie with the same name the auth middleware expects. Attach it to a request
// to authenticate.
//
//	rec := testkit.Request(t, http.MethodGet, "/api/me", nil, testkit.AuthCookie(user))
func AuthCookie(user models.User) *http.Cookie {
	return &http.Cookie{
		Name:     AuthCookieName,
		Value:    auth.IssueJWT(user.ID),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// AuthedRequest is a convenience wrapper around Request that attaches the auth
// cookie for the given user.
//
//	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/me", nil)
func AuthedRequest(t *testing.T, user models.User, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return Request(t, method, path, body, AuthCookie(user))
}
