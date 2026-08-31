package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func authContext() (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("GET", "/", nil)
	return context, recorder
}

func TestGetAuthTokenUsesBearerToken(t *testing.T) {
	context, _ := authContext()
	context.Request.Header.Set("Authorization", "Bearer extension-session")

	token, err := GetAuthToken(context)

	require.NoError(t, err)
	assert.Equal(t, "extension-session", token)
}

func TestGetAuthTokenAcceptsCaseInsensitiveScheme(t *testing.T) {
	context, _ := authContext()
	context.Request.Header.Set("Authorization", "bearer session-token")

	token, err := GetAuthToken(context)

	require.NoError(t, err)
	assert.Equal(t, "session-token", token)
}

func TestGetAuthTokenFallsBackToCookie(t *testing.T) {
	context, _ := authContext()
	context.Request.AddCookie(&http.Cookie{Name: authCookieName, Value: "cookie-session"})

	token, err := GetAuthToken(context)

	require.NoError(t, err)
	assert.Equal(t, "cookie-session", token)
}

func TestGetAuthTokenRejectsMalformedAuthorization(t *testing.T) {
	context, _ := authContext()
	context.Request.Header.Set("Authorization", "Basic credentials")
	context.Request.AddCookie(&http.Cookie{Name: authCookieName, Value: "cookie-session"})

	_, err := GetAuthToken(context)

	require.Error(t, err)
}
