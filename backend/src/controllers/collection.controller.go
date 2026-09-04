package controllers

import (
	"errors"
	"fmt"
	"html"
	nethttp "net/http"
	"net/url"
	"strconv"
	"strings"
	"termorize/src/config"
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
	case errors.Is(err, services.ErrNoCollectionPracticeVocabulary):
		c.JSON(nethttp.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrCollectionPracticeVocabularyUnavailable):
		c.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func respondPublicCollectionError(c *gin.Context, err error) {
	switch {
	case services.CollectionNotFoundError(err), services.InvalidInviteTokenError(err):
		c.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	default:
		ServerError(c, err)
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

func GetPublicCollection(c *gin.Context) {
	collectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "collection not found"})
		return
	}

	collection, err := services.GetPublicCollectionByID(collectionID)
	if err != nil {
		respondPublicCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, collection)
}

func GetPublicCollectionByShareIdentifier(c *gin.Context) {
	collection, err := services.GetPublicCollectionByShareIdentifier(c.Param("identifier"))
	if err != nil {
		respondPublicCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, collection)
}

func GetCollectionShareMetadata(c *gin.Context) {
	requestPath := strings.SplitN(strings.TrimSpace(c.Query("path")), "?", 2)[0]
	decodedPath, err := url.PathUnescape(requestPath)
	if err != nil {
		decodedPath = ""
	}

	var collection *services.PublicCollectionDetail
	switch {
	case strings.HasPrefix(decodedPath, "/collections/join/"):
		identifier := strings.TrimPrefix(decodedPath, "/collections/join/")
		if identifier != "" && !strings.Contains(identifier, "/") {
			collection, _ = services.GetPublicCollectionByShareIdentifier(identifier)
		}
	case strings.HasPrefix(decodedPath, "/collections/"):
		identifier := strings.TrimPrefix(decodedPath, "/collections/")
		if collectionID, parseErr := uuid.Parse(identifier); parseErr == nil {
			collection, _ = services.GetPublicCollectionByID(collectionID)
		}
	}

	publicURL := strings.TrimRight(config.GetPublicURL(), "/")
	if collection == nil {
		writeCollectionMetadata(c, "Termorize", "Translate, collect, and practice the words you want to remember.", publicURL)
		return
	}

	wordPairLabel := "word pairs"
	if collection.TranslationCount == 1 {
		wordPairLabel = "word pair"
	}
	description := fmt.Sprintf(
		"%d %s in “%s”. Preview this published collection and use it in Termorize.",
		collection.TranslationCount,
		wordPairLabel,
		collection.Title,
	)
	writeCollectionMetadata(c, collection.Title+" · Termorize", description, publicURL+decodedPath)
}

func writeCollectionMetadata(c *gin.Context, title, description, canonicalURL string) {
	publicURL := strings.TrimRight(config.GetPublicURL(), "/")
	imageURL := publicURL + "/collection-share.png"
	escapedTitle := html.EscapeString(title)
	escapedDescription := html.EscapeString(description)
	escapedCanonicalURL := html.EscapeString(canonicalURL)
	escapedImageURL := html.EscapeString(imageURL)

	metadata := fmt.Sprintf(`<link rel="canonical" href="%s" />
<meta property="og:type" content="website" />
<meta property="og:site_name" content="Termorize" />
<meta property="og:title" content="%s" />
<meta property="og:description" content="%s" />
<meta property="og:url" content="%s" />
<meta property="og:image" content="%s" />
<meta property="og:image:width" content="1731" />
<meta property="og:image:height" content="909" />
<meta property="og:image:alt" content="A precise arrangement of paired vocabulary slips" />
<meta name="twitter:card" content="summary_large_image" />
<meta name="twitter:title" content="%s" />
<meta name="twitter:description" content="%s" />
<meta name="twitter:image" content="%s" />`,
		escapedCanonicalURL,
		escapedTitle,
		escapedDescription,
		escapedCanonicalURL,
		escapedImageURL,
		escapedTitle,
		escapedDescription,
		escapedImageURL,
	)

	c.Header("Cache-Control", "public, max-age=300")
	c.Data(nethttp.StatusOK, "text/html; charset=utf-8", []byte(metadata))
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

	var req services.AddCollectionToVocabularyRequest
	_ = c.ShouldBindJSON(&req)

	result, err := services.AddCollectionToVocabulary(userID, collectionID, req.TranslationIDs)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, result)
}

func StartCollectionPractice(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, ok := parseUUIDParam(c, "id", "invalid collection ID")
	if !ok {
		return
	}

	round, err := services.StartCollectionPracticeRound(userID, collectionID)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, round)
}

func CreateCollectionPracticeExercise(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	collectionID, ok := parseUUIDParam(c, "id", "invalid collection ID")
	if !ok {
		return
	}

	var req struct {
		TargetVocabularyID uuid.UUID `json:"target_vocabulary_id" binding:"required"`
		Matching           bool      `json:"matching"`
		ExcludeAudio       bool      `json:"exclude_audio"`
	}
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	result, err := services.CreateCollectionPracticeExerciseWithOptions(
		userID,
		collectionID,
		req.TargetVocabularyID,
		req.Matching,
		req.ExcludeAudio,
	)
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, gin.H{
		"exercise_id":                     result.ExerciseID,
		"type":                            result.Type,
		"question_word":                   result.QuestionWord,
		"language":                        result.Language,
		"answer_language":                 result.AnswerLanguage,
		"audio_word_id":                   result.AudioWordID,
		"options":                         result.Options,
		"cards":                           result.Cards,
		"description":                     result.Description,
		"show_ignore_language_suggestion": result.ShowIgnoreLanguageSuggestion,
	})
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

	collection, err := services.JoinCollectionByShareIdentifier(userID, c.Param("identifier"))
	if err != nil {
		respondCollectionError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, collection)
}
