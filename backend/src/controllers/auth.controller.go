package controllers

import (
	"net/http"
	"termorize/src/auth"
	"termorize/src/data/db"
	"termorize/src/models"
	"termorize/src/services"

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

	user, err := services.CreateOrUpdateUserByTelegramAuthData(data)
	if err != nil {
		// TODO add zap as logger
		c.Status(http.StatusInternalServerError)
		return
	}

	auth.SetAuthCookie(c, auth.IssueJWT(user.ID))
	c.JSON(http.StatusOK, user)
}

func Me(c *gin.Context) {
	userID := c.MustGet("userID")

	var user models.User
	db.DB.Where("id = ?", userID).First(&user)

	c.JSON(http.StatusOK, user)
}

func Logout(c *gin.Context) {
	auth.DeleteAuthCookie(c)
}
