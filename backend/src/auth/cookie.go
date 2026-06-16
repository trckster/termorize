package auth

import (
	"net/http"
	"termorize/src/config"

	"github.com/gin-gonic/gin"
)

const authCookieName = "auth"

func SetAuthCookie(c *gin.Context, token string) {
	setCookie(c, authCookieName, token, int(config.GetJWTExpirationTime().Seconds()), authCookieSameSite())
}

func GetAuthCookie(c *gin.Context) (string, error) {
	return c.Cookie(authCookieName)
}

func DeleteAuthCookie(c *gin.Context) {
	setCookie(c, authCookieName, "", -1, authCookieSameSite())
}

func authCookieSameSite() http.SameSite {
	sameSite := http.SameSiteNoneMode
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
