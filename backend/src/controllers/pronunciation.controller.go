package controllers

import (
	"errors"
	nethttp "net/http"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetWordPronunciation(c *gin.Context) {
	wordID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid word ID"})
		return
	}

	pronunciation, err := services.GetOrCreateWordPronunciation(wordID)
	if err != nil {
		if errors.Is(err, services.ErrWordNotFound) {
			c.JSON(nethttp.StatusNotFound, gin.H{"error": services.ErrWordNotFound.Error()})
			return
		}

		ServerError(c, err)
		return
	}

	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Content-Disposition", `inline; filename="pronunciation.mp3"`)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(nethttp.StatusOK, pronunciation.MIMEType, pronunciation.Audio)
}
