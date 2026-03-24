package http

import (
	"net/http"
	"termorize/src/controllers"
	"termorize/src/http/middlewares"
	"termorize/src/integrations/telegram"

	"github.com/gin-gonic/gin"
)

/** Defines routes protected by authentication */
func defineProtectedRoutes(group *gin.RouterGroup) {
	group.GET("/me", controllers.Me)
	group.PUT("/settings", controllers.UpdateSettings)

	group.GET("/vocabulary", controllers.GetVocabulary)
	group.POST("/vocabulary", controllers.CreateVocabulary)
	group.POST("/vocabulary/translation", controllers.CreateVocabularyByTranslation)
	group.DELETE("/vocabulary/:id", controllers.DeleteVocabulary)
	group.GET("/exercises", controllers.GetExercises)
	group.GET("/exercises/statistics", controllers.GetExerciseStatistics)

	group.POST("/translate", controllers.Translate)
}

func definePublicRoutes(group *gin.RouterGroup) {
	group.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "nice"})
	})

	group.POST("/telegram/login/start", controllers.StartTelegramLogin)
	group.POST("/telegram/login/callback", controllers.CompleteTelegramLogin)
	group.POST("/logout", controllers.Logout)

	group.GET("/settings", controllers.GetSettings)

	group.POST("/telegram/webhook", middlewares.TelegramWebhookMiddleware(), telegram.HandleWebhook)
}
