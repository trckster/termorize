package middlewares

import (
	"net/http"
	"termorize/src/config"

	"github.com/gin-gonic/gin"
)

func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := config.GetPublicURL()
		if config.IsLocal() {
			origin = c.Request.Header.Get("Origin")
			if origin == "" {
				origin = "*"
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, X-Timezone")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
