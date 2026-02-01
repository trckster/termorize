package http

import (
	"net/http"
	"termorize/src/config"

	"github.com/gin-gonic/gin"
)

func LaunchServer() {
	r := gin.Default()

	r.POST("/telegram/login", telegramLogin)
	r.GET("/ping", ping)

	r.Run(":" + config.GetPort())
}

func telegramLogin(c *gin.Context) {
	c.Status(http.StatusOK)
}

func ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "nice"})
}
