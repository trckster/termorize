package middlewares

import (
	"errors"
	"net/http"
	"termorize/src/auth"
	"termorize/src/services"
	"time"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := auth.GetAuthToken(c)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		userID, err := auth.DecodeJWT(tokenString)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		user, err := services.GetUserByID(userID)
		if err != nil {
			if errors.Is(err, services.ErrUserNotFound) {
				c.AbortWithStatus(http.StatusUnauthorized)
			} else {
				c.AbortWithStatus(http.StatusInternalServerError)
			}
			return
		}
		if user.GuestExpiresAt != nil && !user.GuestExpiresAt.After(time.Now().UTC()) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set("userID", userID)

		c.Next()
	}
}
