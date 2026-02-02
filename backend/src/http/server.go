package http

import (
	"net/http"
	"termorize/src/auth"
	"termorize/src/config"
	"termorize/src/database"
	"termorize/src/models"

	"github.com/gin-gonic/gin"
)

func LaunchServer() {
	r := gin.Default()

	api := r.Group("/api")
	{
		api.POST("/telegram/login", telegramLogin)
		api.GET("/ping", ping)
	}

	r.Run(":" + config.GetPort())
}

func telegramLogin(c *gin.Context) {
	var data auth.TelegramAuthData
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	botToken := config.GetTelegramBotToken()
	if !auth.ValidateTelegramAuth(data, botToken) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
		return
	}

	var user models.User
	result := database.DB.Where("telegram_id = ?", data.ID).First(&user)

	if result.Error == nil {
		c.Status(http.StatusOK)
		return
	}

	user = models.User{
		TelegramID: data.ID,
		Username:   data.Username,
		Name:       data.FirstName + " " + data.LastName,
		PhotoUrl:   data.PhotoUrl,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.Status(http.StatusCreated)
}

func ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "nice"})
}
