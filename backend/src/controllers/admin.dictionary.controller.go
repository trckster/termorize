package controllers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"termorize/src/services"
)

func GetAdminDictionaries(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}
	result, err := services.ListDictionaries()
	if err != nil {
		ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func StartAdminDictionaryImport(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}
	id, ok := dictionaryID(c)
	if !ok {
		return
	}
	result, err := services.StartDictionaryImport(id)
	if err != nil {
		dictionaryError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, result)
}

func GetAdminDictionaryImports(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}
	id, ok := dictionaryID(c)
	if !ok {
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		dictionaryError(c, services.ErrInvalidPage)
		return
	}
	result, err := services.ListDictionaryImports(id, page)
	if err != nil {
		dictionaryError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func dictionaryID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dictionary ID"})
		return uuid.Nil, false
	}
	return id, true
}

func dictionaryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "dictionary not found"})
	case errors.Is(err, services.ErrDictionaryImportActive):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case services.InvalidPaginationError(err), errors.Is(err, services.ErrDictionaryEdition):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		ServerError(c, err)
	}
}
