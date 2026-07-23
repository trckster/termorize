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

const AuthCookieName = "auth"

type UserOption func(*models.User)

func WithName(name string) UserOption {
	return func(u *models.User) { u.Name = name }
}

func WithUsername(username string) UserOption {
	return func(u *models.User) { u.Username = username }
}

func WithTelegramID(id int64) UserOption {
	return func(u *models.User) { u.TelegramID = id }
}

func WithAdmin() UserOption {
	return func(u *models.User) { u.IsAdmin = true }
}

func WithSettings(settings models.UserSettings) UserOption {
	return func(u *models.User) { u.Settings = settings }
}

var telegramIDSeq int64 = 100000

func CreateUser(t *testing.T, opts ...UserOption) models.User {
	t.Helper()

	telegramIDSeq++
	user := models.User{
		Username:   fmt.Sprintf("user_%d", telegramIDSeq),
		Name:       fmt.Sprintf("Test User %d", telegramIDSeq),
		TelegramID: telegramIDSeq,
	}

	for _, opt := range opts {
		opt(&user)
	}

	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("testkit.CreateUser: failed to insert user: %v", err)
	}

	return user
}

func AuthCookie(user models.User) *http.Cookie {
	return &http.Cookie{
		Name:     AuthCookieName,
		Value:    auth.IssueJWT(user.ID),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func AuthedRequest(t *testing.T, user models.User, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return Request(t, method, path, body, AuthCookie(user))
}
