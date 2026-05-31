package controllers

import (
	"errors"
	nethttp "net/http"
	"strconv"
	"strings"
	"termorize/src/enums"
	"termorize/src/http/validators"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func parseUUIDParam(c *gin.Context, name, errMsg string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": errMsg})
		return uuid.Nil, false
	}
	return id, true
}

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
		ServerError(c, errors.New("request to OpenRouter failed"))
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

	var languages []enums.Language
	if langParam := c.Query("languages"); langParam != "" {
		for _, code := range strings.Split(langParam, ",") {
			code = strings.TrimSpace(code)
			if code != "" {
				languages = append(languages, enums.Language(code))
			}
		}
	}

	response, err := services.ListCollections(userID, page, pageSize, search, languages)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, response)
}

func GetCollection(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, ok := parseUUIDParam(c, "id", "invalid collection ID")
	if !ok {
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

	collectionID, ok := parseUUIDParam(c, "id", "invalid collection ID")
	if !ok {
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

	collectionID, ok := parseUUIDParam(c, "id", "invalid collection ID")
	if !ok {
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

	collectionID, ok := parseUUIDParam(c, "id", "invalid collection ID")
	if !ok {
		return
	}

	translationID, ok := parseUUIDParam(c, "translationId", "invalid translation ID")
	if !ok {
		return
	}

	if err := services.RemoveTranslationFromCollection(userID, collectionID, translationID); err != nil {
		respondCollectionError(c, err)
		return
	}

	c.Status(nethttp.StatusOK)
}

func ReorderCollectionTranslations(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, ok := parseUUIDParam(c, "id", "invalid collection ID")
	if !ok {
		return
	}

	var req services.ReorderCollectionTranslationsRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	collection, err := services.ReorderCollectionTranslations(userID, collectionID, req.TranslationIDs)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, collection)
}

func AddCollectionToVocabulary(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, ok := parseUUIDParam(c, "id", "invalid collection ID")
	if !ok {
		return
	}

	// Optional body: absent means "add all", a translation_ids list restricts to those.
	var req services.AddCollectionToVocabularyRequest
	_ = c.ShouldBindJSON(&req)

	result, err := services.AddCollectionToVocabulary(userID, collectionID, req.TranslationIDs)
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

	collectionID, ok := parseUUIDParam(c, "id", "invalid collection ID")
	if !ok {
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

func UpdateCollection(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, ok := parseUUIDParam(c, "id", "invalid collection ID")
	if !ok {
		return
	}

	var req services.UpdateCollectionTitleRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	collection, err := services.UpdateCollectionTitle(userID, collectionID, req)
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
