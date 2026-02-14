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
		if err.Error() == "translation already exists" {
			c.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
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
		if err.Error() == "page must be greater than 0" || err.Error() == "page size must be between 1 and 1000" {
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
		if err.Error() == "vocabulary item not found" {
			c.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(nethttp.StatusOK)
}

func Translate(c *gin.Context) {
	var req services.TranslateRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	result, err := services.Translate(req)
	if err != nil {
		if err.Error() == "languages must differ" {
			c.JSON(nethttp.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, result)
}
