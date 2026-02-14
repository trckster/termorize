package controllers

import (
	nethttp "net/http"
	"termorize/src/enums"
	"termorize/src/http/validators"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
)

type TranslateRequest struct {
	FromWord     string         `json:"from_word" binding:"required"`
	FromLanguage enums.Language `json:"from_language" binding:"required,enum=Language"`
	ToLanguage   enums.Language `json:"to_language" binding:"required,enum=Language,nefield=FromLanguage"`
}

type TranslateResponse struct {
	Translation string `json:"translation"`
}

func Translate(c *gin.Context) {
	var req TranslateRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	translation, err := services.Translate(req.FromWord, req.FromLanguage, req.ToLanguage)
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, TranslateResponse{Translation: translation})
}
