package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"termorize/src/classification"
	"termorize/src/enums"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetAdminUnknownWords(c *gin.Context)         { getCategorizationList(c, "unknown") }
func GetAdminMismatchedVocabulary(c *gin.Context) { getCategorizationList(c, "mismatches") }
func GetAdminCategoryWords(c *gin.Context)        { getCategorizationList(c, "words") }

func getCategorizationList(c *gin.Context, list string) {
	if !authorizeAdmin(c) {
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page size"})
		return
	}
	var response any
	if list == "mismatches" {
		response, err = services.GetMismatchedVocabulary(c.Request.Context(), page, pageSize)
	} else if list == "words" {
		response, err = services.GetCategoryWords(c.Request.Context(), page, pageSize, c.Query("search"))
	} else {
		response, err = services.GetUnknownWords(c.Request.Context(), page, pageSize)
	}
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

func SetAdminWordPartOfSpeech(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid word ID"})
		return
	}
	var request struct {
		PartOfSpeech enums.PartOfSpeech `json:"part_of_speech" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "part_of_speech is required"})
		return
	}
	word, err := services.SetWordPartOfSpeech(c.Request.Context(), id, request.PartOfSpeech)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidPartOfSpeech):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "word not found"})
		default:
			ServerError(c, err)
		}
		return
	}
	c.JSON(http.StatusOK, word)
}

func GetAdminCategorizationStats(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}
	stats, err := services.GetCategorizationStats(c.Request.Context())
	if err != nil {
		ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

func RestartAdminCategorization(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}
	if err := services.RestartCategorization(c.Request.Context()); err != nil {
		switch {
		case errors.Is(err, classification.ErrBusy):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, classification.ErrUnavailable):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		default:
			ServerError(c, err)
		}
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "started"})
}
