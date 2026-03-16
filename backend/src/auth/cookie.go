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
	sameSite := http.SameSiteStrictMode
	if config.IsLocal() {
		sameSite = http.SameSiteNoneMode
	}

	return sameSite
}

func setCookie(c *gin.Context, name string, token string, time int, sameSite http.SameSite) {
	cookie := &http.Cookie{
		Name:     name,
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
