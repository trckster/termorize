package auth

import (
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
	c.SetCookie(
		authCookieName,
		token,
		time,
		"/",
		config.GetDomain(),
		!config.IsLocal(),
		true,
	)
}
