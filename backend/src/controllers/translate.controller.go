package controllers

import (
	nethttp "net/http"
	"termorize/src/enums"
	"termorize/src/http/validators"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TranslateRequest struct {
	FromWord     string         `json:"from_word" binding:"required,max=5000"`
	FromLanguage enums.Language `json:"from_language" binding:"required,enum=Language"`
	ToLanguage   enums.Language `json:"to_language" binding:"required,enum=Language,nefield=FromLanguage"`
}

type TranslateSelectionRequest struct {
	FromWord   string         `json:"from_word" binding:"required,max=5000"`
	ToLanguage enums.Language `json:"to_language" binding:"required,enum=Language"`
}

type TranslateResponse struct {
	ID                uuid.UUID               `json:"id"`
	OriginalWordID    uuid.UUID               `json:"original_word_id"`
	TranslationWordID uuid.UUID               `json:"translation_word_id"`
	OriginalLanguage  enums.Language          `json:"original_language,omitempty"`
	Translation       string                  `json:"translation"`
	Source            enums.TranslationSource `json:"source"`
}

func Translate(c *gin.Context) {
	var req TranslateRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	translation, err := services.Translate(req.FromWord, req.FromLanguage, req.ToLanguage)
	if err != nil {
		ServerError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, TranslateResponse{
		ID:                translation.TranslationID,
		OriginalWordID:    translation.SourceWordID,
		TranslationWordID: translation.TranslatedWordID,
		Translation:       translation.TranslatedWord,
		Source:            translation.Source,
	})
}

func TranslateSelection(c *gin.Context) {
	var req TranslateSelectionRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	detectedLanguage, supported, err := services.DetectLanguage(req.FromWord)
	if err != nil {
		ServerError(c, err)
		return
	}

	if !supported {
		c.JSON(nethttp.StatusUnprocessableEntity, gin.H{
			"error":             "unsupported source language",
			"detected_language": detectedLanguage,
		})
		return
	}

	if detectedLanguage == req.ToLanguage {
		c.JSON(nethttp.StatusUnprocessableEntity, gin.H{
			"error":             "source language matches target language",
			"detected_language": detectedLanguage,
		})
		return
	}

	translation, err := services.Translate(req.FromWord, detectedLanguage, req.ToLanguage)
	if err != nil {
		ServerError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, TranslateResponse{
		ID:                translation.TranslationID,
		OriginalWordID:    translation.SourceWordID,
		TranslationWordID: translation.TranslatedWordID,
		OriginalLanguage:  detectedLanguage,
		Translation:       translation.TranslatedWord,
		Source:            translation.Source,
	})
}
