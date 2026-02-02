package middlewares

import (
	"net/http"
	"termorize/src/auth"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := auth.GetAuthCookie(c)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		userID, err := auth.DecodeJWT(tokenString)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
		}

		c.Set("userID", userID)

		c.Next()
	}
}
