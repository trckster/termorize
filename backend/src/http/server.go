package http

import (
	"termorize/src/config"
	"termorize/src/controllers"

	"github.com/gin-gonic/gin"
)

func LaunchServer() {
	r := gin.Default()

	api := r.Group("/api")
	{
		api.POST("/telegram/login", controllers.TelegramLogin)
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "nice"})
		})
	}

	r.Run(":" + config.GetPort())
}
