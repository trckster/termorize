package controllers

import (
	"errors"
	"net/http"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
)

func GetAdminUsers(c *gin.Context) {
	response, err := services.GetRecentUsersForAdmin(c.MustGet("userID").(uint))
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if errors.Is(err, services.ErrAdminRequired) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		ServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}
