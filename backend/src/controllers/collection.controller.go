package controllers

import (
	nethttp "net/http"
	"strconv"
	"termorize/src/http/validators"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func respondCollectionError(c *gin.Context, err error) {
	switch {
	case services.CollectionNotFoundError(err):
		c.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	case services.CollectionForbiddenError(err):
		c.JSON(nethttp.StatusForbidden, gin.H{"error": err.Error()})
	case services.InvalidInviteTokenError(err):
		c.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	case services.CollectionTitleRequiredError(err):
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
	case services.AIPromptRequiredError(err):
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
	case services.InvalidPaginationError(err):
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
	case services.AIGenerationUnavailableError(err):
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": err.Error()})
	case services.AIGenerationFailedError(err):
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": "request to OpenRouter failed"})
	default:
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func GetCollections(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	search := c.Query("search")

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

	response, err := services.ListCollections(userID, page, pageSize, search)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, response)
}

func GetCollection(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid collection ID"})
		return
	}

	collection, err := services.GetCollection(userID, collectionID)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, collection)
}

func CreateCollection(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req services.CreateCollectionRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	collection, err := services.CreateCollection(userID, req)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusCreated, collection)
}

func DeleteCollection(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid collection ID"})
		return
	}

	if err := services.DeleteCollection(userID, collectionID); err != nil {
		respondCollectionError(c, err)
		return
	}

	c.Status(nethttp.StatusOK)
}

func AddCollectionTranslation(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid collection ID"})
		return
	}

	var req services.AddCollectionTranslationRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	collection, err := services.AddTranslationToCollection(userID, collectionID, req)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusCreated, collection)
}

func RemoveCollectionTranslation(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid collection ID"})
		return
	}

	translationID, err := uuid.Parse(c.Param("translationId"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid translation ID"})
		return
	}

	if err := services.RemoveTranslationFromCollection(userID, collectionID, translationID); err != nil {
		respondCollectionError(c, err)
		return
	}

	c.Status(nethttp.StatusOK)
}

func AddCollectionToVocabulary(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid collection ID"})
		return
	}

	result, err := services.AddCollectionToVocabulary(userID, collectionID)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, result)
}

func GenerateCollection(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req services.GenerateCollectionRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	collection, err := services.GenerateCollection(userID, req.Prompt)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusCreated, collection)
}

func PublishCollection(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid collection ID"})
		return
	}

	var req services.SetCollectionIsPublishedRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	collection, err := services.SetCollectionIsPublished(userID, collectionID, req.IsPublished)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, collection)
}

func JoinCollection(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collection, err := services.JoinCollectionByToken(userID, c.Param("token"))
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, collection)
}
