package controllers

import (
	nethttp "net/http"
	"strconv"
	"termorize/src/http/validators"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateVocabulary(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req services.CreateVocabularyRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	vocabulary, err := services.CreateVocabulary(userID, req)
	if err != nil {
		if services.TranslationAlreadyExistsError(err) {
			c.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusCreated, vocabulary)
}

func CreateVocabularyByTranslation(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req services.CreateVocabularyByTranslationRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	vocabulary, err := services.CreateVocabularyByTranslation(userID, req.TranslationID)
	if err != nil {
		if services.TranslationAlreadyExistsError(err) {
			c.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		if services.TranslationNotFoundError(err) {
			c.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusCreated, vocabulary)
}

func GetVocabulary(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	page := 1
	pageSize := 50

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil {
			pageSize = parsed
		}
	}

	response, err := services.GetVocabulary(userID, page, pageSize)
	if err != nil {
		if services.InvalidPaginationError(err) {
			c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, response)
}

func DeleteVocabulary(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	vocabIDStr := c.Param("id")
	vocabID, err := uuid.Parse(vocabIDStr)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid vocabulary ID"})
		return
	}

	if err := services.DeleteVocabulary(userID, vocabID); err != nil {
		if services.VocabularyNotFoundError(err) {
			c.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(nethttp.StatusOK)
}
