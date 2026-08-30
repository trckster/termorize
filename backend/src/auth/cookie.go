package auth

import (
	"errors"
	"net/http"
	"strings"
	"termorize/src/config"

	"github.com/gin-gonic/gin"
)

const authCookieName = "auth"

var errAuthTokenMissing = errors.New("auth token missing")

func SetAuthCookie(c *gin.Context, token string) {
	setCookie(c, authCookieName, token, int(config.GetJWTExpirationTime().Seconds()), authCookieSameSite())
}

func GetAuthCookie(c *gin.Context) (string, error) {
	return c.Cookie(authCookieName)
}

// GetAuthToken accepts the existing browser session cookie and a Bearer copy of
// the same JWT. The latter lets the browser extension reuse the user's session
// even when Chromium blocks third-party cookies for extension requests.
func GetAuthToken(c *gin.Context) (string, error) {
	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	if authorization != "" {
		scheme, token, found := strings.Cut(authorization, " ")
		token = strings.TrimSpace(token)
		if found && strings.EqualFold(scheme, "Bearer") && token != "" {
			return token, nil
		}

		return "", errAuthTokenMissing
	}

	return GetAuthCookie(c)
}

func DeleteAuthCookie(c *gin.Context) {
	setCookie(c, authCookieName, "", -1, authCookieSameSite())
}

func authCookieSameSite() http.SameSite {
	sameSite := http.SameSiteStrictMode
	if config.IsLocal() {
		sameSite = http.SameSiteLaxMode
	}

	return sameSite
}

func setCookie(c *gin.Context, name string, token string, time int, sameSite http.SameSite) {
	secure := true
	domain := config.GetDomain()

	if config.IsLocal() {
		secure = false
		domain = ""
	}

	cookie := &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		Domain:   domain,
		MaxAge:   time,
		Secure:   secure,
		HttpOnly: true,
		SameSite: sameSite,
	}

	http.SetCookie(c.Writer, cookie)
}
