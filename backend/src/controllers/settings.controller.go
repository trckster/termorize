package controllers

import (
	"net/http"
	"termorize/src/enums"

	"github.com/gin-gonic/gin"
)

func GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"languages": enums.AllLanguages(),
	})
}
