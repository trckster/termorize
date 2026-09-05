package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetAdminDescriptionModels(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}
	c.JSON(http.StatusOK, services.AdminDescriptionModels)
}

func GetAdminWordDescriptions(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	response, err := services.GetWordDescriptionsForAdmin(page, pageSize, c.Query("search"))
	if err != nil {
		adminDescriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func PreviewAdminWordDescription(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	var request struct {
		Model string `json:"model" binding:"required"`
	}
	if c.ShouldBindJSON(&request) != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	preview, err := services.PreviewWordDescriptionForAdmin(id, c.MustGet("userID").(uint), request.Model)
	if err != nil {
		adminDescriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
}

func ApproveAdminWordDescription(c *gin.Context) {
	if !authorizeAdmin(c) {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	var request struct {
		PreviewID uuid.UUID `json:"preview_id" binding:"required"`
	}
	if c.ShouldBindJSON(&request) != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := services.ApproveWordDescriptionForAdmin(id, request.PreviewID, c.MustGet("userID").(uint)); err != nil {
		adminDescriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"approved": true})
}

func adminDescriptionError(c *gin.Context, err error) {
	switch {
	case services.InvalidPaginationError(err), errors.Is(err, services.ErrInvalidDescriptionModel):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.AbortWithStatus(http.StatusNotFound)
	case errors.Is(err, services.ErrDescriptionPreviewConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		ServerError(c, err)
	}
}
