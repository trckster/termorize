package auth

import (
	"net/http"
	"termorize/src/config"

	"github.com/gin-gonic/gin"
)

const authCookieName = "auth"

func SetAuthCookie(c *gin.Context, token string) {
	setAuthCookie(c, token, int(config.GetJWTExpirationTime().Seconds()))
}

func GetAuthCookie(c *gin.Context) (string, error) {
	return c.Cookie(authCookieName)
}

func DeleteAuthCookie(c *gin.Context) {
	setAuthCookie(c, "", -1)
}

func setAuthCookie(c *gin.Context, token string, time int) {
	sameSite := http.SameSiteStrictMode
	if config.IsLocal() {
		sameSite = http.SameSiteNoneMode
	}

	cookie := &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		Domain:   config.GetDomain(),
		MaxAge:   time,
		Secure:   true,
		HttpOnly: true,
		SameSite: sameSite,
	}

	http.SetCookie(c.Writer, cookie)
}
