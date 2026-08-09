package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

func GetAdminWordPronunciations(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}

	page := 1
	pageSize := 20
	if value := c.Query("page"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			page = parsed
		}
	}
	if value := c.Query("page_size"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			pageSize = parsed
		}
	}

	response, err := services.GetWordPronunciationsForAdmin(page, pageSize, c.Query("search"))
	if err != nil {
		if services.InvalidPaginationError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func GetAdminWordPronunciationAudio(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid word pronunciation ID"})
		return
	}

	audio, mimeType, err := services.GetWordPronunciationAudioForAdmin(id)
	if err != nil {
		if errors.Is(err, services.ErrWordPronunciationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ServerError(c, err)
		return
	}

	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("Content-Disposition", `inline; filename="pronunciation.mp3"`)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, mimeType, audio)
}

func RegenerateAdminWordPronunciation(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid word pronunciation ID"})
		return
	}

	pronunciation, err := services.RegenerateWordPronunciationForAdmin(id)
	if err != nil {
		if errors.Is(err, services.ErrWordPronunciationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, pronunciation)
}

func authorizeAdmin(c *gin.Context) bool {
	err := services.RequireAdmin(c.MustGet("userID").(uint))
	if err == nil {
		return true
	}
	if errors.Is(err, services.ErrUserNotFound) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return false
	}
	if errors.Is(err, services.ErrAdminRequired) {
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}

	ServerError(c, err)
	return false
}
