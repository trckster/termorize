package controllers

import (
	"net/http"
	"termorize/src/enums"
	"termorize/src/services"
	"time"

	"github.com/gin-gonic/gin"
)

func GetDailyIdiom(c *gin.Context) {
	language := enums.Language(c.Query("language"))
	if !enums.IsSupportedLanguage(language) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language must be a supported language"})
		return
	}

	result, err := services.GetDailyIdiom(c.Request.Context(), c.GetUint("userID"), language, time.Now())
	if err != nil {
		ServerError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, result)
}
