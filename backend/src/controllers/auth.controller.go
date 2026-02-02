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
		auth.SetAuthCookie(c, auth.IssueJWT(user.ID))
		c.Status(http.StatusOK)
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

	auth.SetAuthCookie(c, auth.IssueJWT(user.ID))
	c.Status(http.StatusCreated)
}

func Me(c *gin.Context) {
	userID := c.MustGet("userID")

	var user models.User
	database.DB.Where("id = ?", userID).First(&user)

	c.JSON(http.StatusOK, user)
}

func Logout(c *gin.Context) {
	auth.DeleteAuthCookie(c)
}
