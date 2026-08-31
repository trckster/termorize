package middlewares

import (
	"net/http"
	"strings"
	"termorize/src/config"

	"github.com/gin-gonic/gin"
)

const googleTranslateOrigin = "https://translate.google.com"

func resolveAllowedCorsOrigin(publicURL, requestOrigin string, isLocal bool) string {
	if isLocal {
		if requestOrigin == "" {
			return "*"
		}
		return requestOrigin
	}

	publicOrigin := strings.TrimRight(publicURL, "/")
	if requestOrigin == publicOrigin || requestOrigin == googleTranslateOrigin {
		return requestOrigin
	}

	return ""
}

func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestOrigin := c.Request.Header.Get("Origin")
		origin := resolveAllowedCorsOrigin(config.GetPublicURL(), requestOrigin, config.IsLocal())
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Add("Vary", "Origin")
			if origin != "*" {
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, X-Timezone, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
