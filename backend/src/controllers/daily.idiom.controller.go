package controllers

import (
	"errors"
	"net/http"
	"termorize/src/enums"
	"termorize/src/services"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

func GetDailyIdiomDescription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid daily idiom id"})
		return
	}
	description, err := services.GetDailyIdiomDescription(c.Request.Context(), id)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.AbortWithStatus(http.StatusNotFound)
	case errors.Is(err, services.ErrDescriptionGenerationFailed):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not generate idiom description; retry"})
	case err != nil:
		ServerError(c, err)
	default:
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{"description": description.Description})
	}
}
