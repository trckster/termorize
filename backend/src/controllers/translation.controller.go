package controllers

import (
	nethttp "net/http"
	"strconv"
	"termorize/src/http/validators"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
)

func CreateTranslation(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req services.CreateTranslationRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	vocabulary, err := services.CreateTranslation(userID, req)
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
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, response)
}
