package middlewares

import (
	nethttp "net/http"
	"termorize/src/integrations/telegram"

	"github.com/gin-gonic/gin"
)

func TelegramWebhookMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		providedSecret := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
		expectedSecret := telegram.BuildWebhookSecret()

		if providedSecret != expectedSecret {
			c.Status(nethttp.StatusUnauthorized)
			c.Abort()
			return
		}

		c.Next()
	}
}
