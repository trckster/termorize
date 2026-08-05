package controllers

import (
	"errors"
	nethttp "net/http"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetWordPronunciation(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	wordID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid word ID"})
		return
	}

	pronunciation, err := services.GetOrCreateWordPronunciation(userID, wordID)
	if err != nil {
		if errors.Is(err, services.ErrWordNotFound) {
			c.JSON(nethttp.StatusNotFound, gin.H{"error": services.ErrWordNotFound.Error()})
			return
		}
		if limitErr, ok := services.AsOpenRouterSpendingLimitError(err); ok {
			c.JSON(nethttp.StatusTooManyRequests, gin.H{
				"error":    "AI spending limit reached",
				"limit":    limitErr.Limit,
				"retry_at": limitErr.RetryAt,
			})
			return
		}

		ServerError(c, err)
		return
	}

	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("Content-Disposition", `inline; filename="pronunciation.mp3"`)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(nethttp.StatusOK, pronunciation.MIMEType, pronunciation.Audio)
}
