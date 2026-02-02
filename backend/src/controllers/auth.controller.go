package controllers

import (
	"net/http"
	"strings"
	"termorize/src/auth"
	"termorize/src/database"
	"termorize/src/models"

	"github.com/gin-gonic/gin"
)

func TelegramLogin(c *gin.Context) {
	var data auth.TelegramAuthData
	if err := c.BindJSON(&data); err != nil {
		return
	}

	if !auth.ValidateTelegramAuth(data) {
		c.Status(http.StatusUnauthorized)
		return
	}

	var user models.User
	result := database.DB.Where("telegram_id = ?", data.ID).First(&user)

	if result.Error == nil {
		// TODO update user data on login
		c.JSON(http.StatusOK, gin.H{"token": auth.IssueJWT(user.ID)})
		return
	}

	name := data.FirstName + " " + data.LastName
	user = models.User{
		TelegramID: data.ID,
		Username:   data.Username,
		Name:       strings.TrimSpace(name),
		PhotoUrl:   data.PhotoUrl,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": auth.IssueJWT(user.ID)})
}
