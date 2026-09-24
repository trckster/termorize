package controllers

import (
	"errors"
	"net/http"
	"termorize/src/enums"
	"termorize/src/http/validators"
	"termorize/src/services"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetDailyIdiom(c *gin.Context) {
	result, err := services.GetDailyIdiom(c.Request.Context(), c.GetUint("userID"), time.Now())
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

func TranslateDailyIdiom(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid daily idiom id"})
		return
	}
	var req struct {
		ToLanguage enums.Language `json:"to_language" binding:"required,enum=Language"`
	}
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}
	result, err := services.TranslateDailyIdiom(c.Request.Context(), id, req.ToLanguage)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.AbortWithStatus(http.StatusNotFound)
	case errors.Is(err, services.ErrInvalidIdiomTarget):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case err != nil:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not translate idiom; retry"})
	default:
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, TranslateResponse{ID: result.TranslationID, OriginalWordID: result.SourceWordID, TranslationWordID: result.TranslatedWordID, Translation: result.TranslatedWord, Source: result.Source})
	}
}
