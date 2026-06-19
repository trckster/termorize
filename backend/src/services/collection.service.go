package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/logger"
	"termorize/src/models"
	"termorize/src/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CreateCollectionRequest struct {
	Title   string `json:"title" binding:"required"`
	IsAdmin bool   `json:"is_admin"`
}

type AddCollectionTranslationRequest struct {
	Original            string         `json:"original" binding:"required"`
	Translation         string         `json:"translation" binding:"required"`
	OriginalLanguage    enums.Language `json:"original_language" binding:"required,enum=Language"`
	TranslationLanguage enums.Language `json:"translation_language" binding:"required,enum=Language,nefield=OriginalLanguage"`
}

type CollectionSummary struct {
	ID               uuid.UUID        `json:"id"`
	Title            string           `json:"title"`
	IsAdmin          bool             `json:"is_admin"`
	IsOwner          bool             `json:"is_owner"`
	IsPublished      bool             `json:"is_published"`
	OwnerUsername    string           `json:"owner_username,omitempty"`
	Languages        []enums.Language `json:"languages"`
	TranslationCount int              `json:"translation_count"`
	UserAddCount     int              `json:"user_add_count"`
	CreatedAt        time.Time        `json:"created_at"`
}

type CollectionListResponse struct {
	Data       []CollectionSummary `json:"data"`
	Pagination Pagination          `json:"pagination"`
}

type CollectionDetail struct {
	ID               uuid.UUID            `json:"id"`
	Title            string               `json:"title"`
	IsAdmin          bool                 `json:"is_admin"`
	IsOwner          bool                 `json:"is_owner"`
	IsPublished      bool                 `json:"is_published"`
	OwnerUsername    string               `json:"owner_username,omitempty"`
	Languages        []enums.Language     `json:"languages"`
	TranslationCount int                  `json:"translation_count"`
	UserAddCount     int                  `json:"user_add_count"`
	CreatedAt        time.Time            `json:"created_at"`
	InviteToken      string               `json:"invite_token,omitempty"`
	Translations     []models.Translation `json:"translations"`
}

type GenerateCollectionRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

type SetCollectionIsPublishedRequest struct {
	IsPublished bool `json:"is_published"`
}

type AddCollectionToVocabularyRequest struct {
	TranslationIDs []uuid.UUID `json:"translation_ids"`
}

type AddCollectionToVocabularyResult struct {
	Added        int `json:"added"`
	Skipped      int `json:"skipped"`
	Total        int `json:"total"`
	UserAddCount int `json:"user_add_count"`
}

const collectionNotFoundError = "collection not found"
const collectionForbiddenError = "you don't have access to this collection"
const collectionTitleRequiredError = "collection title can't be empty"
const collectionAdminForbiddenError = "only admins can manage global collections"
const invalidInviteTokenError = "invalid invite link"
const aiPromptRequiredError = "prompt can't be empty"
const aiGenerationUnavailableError = "ai generation is not configured"
const aiGenerationFailedError = "request to OpenRouter failed"

var (
	ErrCollectionNotFound       = errors.New(collectionNotFoundError)
	ErrCollectionForbidden      = errors.New(collectionForbiddenError)
	ErrCollectionTitleRequired  = errors.New(collectionTitleRequiredError)
	ErrCollectionAdminForbidden = errors.New(collectionAdminForbiddenError)
	ErrInvalidInviteToken       = errors.New(invalidInviteTokenError)
	ErrAIPromptRequired         = errors.New(aiPromptRequiredError)
	ErrAIGenerationUnavailable  = errors.New(aiGenerationUnavailableError)
	ErrAIGenerationFailed       = errors.New(aiGenerationFailedError)
)

func CollectionNotFoundError(err error) bool {
	return errors.Is(err, ErrCollectionNotFound)
}

func CollectionForbiddenError(err error) bool {
	return errors.Is(err, ErrCollectionForbidden) || errors.Is(err, ErrCollectionAdminForbidden)
}

func CollectionTitleRequiredError(err error) bool {
	return errors.Is(err, ErrCollectionTitleRequired)
}

func InvalidInviteTokenError(err error) bool {
	return errors.Is(err, ErrInvalidInviteToken)
}

func AIPromptRequiredError(err error) bool {
	return errors.Is(err, ErrAIPromptRequired)
}

func AIGenerationUnavailableError(err error) bool {
	return errors.Is(err, ErrAIGenerationUnavailable)
}

func AIGenerationFailedError(err error) bool {
	return errors.Is(err, ErrAIGenerationFailed)
}

const collectionTitleMaxLength = 255

func truncateTitle(title string) string {
	if runes := []rune(title); len(runes) > collectionTitleMaxLength {
		return string(runes[:collectionTitleMaxLength])
	}
	return title
}

func GenerateInviteToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func userIsAdmin(conn *gorm.DB, userID uint) (bool, error) {
	var user models.User
	if err := conn.Select("is_admin").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return user.IsAdmin, nil
}

func getAccessibleCollection(conn *gorm.DB, userID uint, collectionID uuid.UUID) (*models.Collection, error) {
	var collection models.Collection
	err := conn.Where("id = ? AND deleted_at IS NULL", collectionID).First(&collection).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCollectionNotFound
		}
		return nil, err
	}

	if (collection.IsAdmin && collection.IsPublished) || collection.IsOwnedBy(userID) {
		return &collection, nil
	}

	var memberCount int64
	if err := conn.Model(&models.CollectionMember{}).
		Where("collection_id = ? AND user_id = ?", collectionID, userID).
		Count(&memberCount).Error; err != nil {
		return nil, err
	}
	if memberCount > 0 {
		return &collection, nil
	}

	isAdmin, err := userIsAdmin(conn, userID)
	if err != nil {
		return nil, err
	}
	if isAdmin {
		return &collection, nil
	}

	return nil, ErrCollectionNotFound
}

func canEditCollection(conn *gorm.DB, userID uint, collection *models.Collection) (bool, error) {
	if collection.IsOwnedBy(userID) {
		return true, nil
	}
	if collection.IsAdmin {
		return userIsAdmin(conn, userID)
	}
	return false, nil
}

func getEditableCollection(conn *gorm.DB, userID uint, collectionID uuid.UUID) (*models.Collection, error) {
	collection, err := getAccessibleCollection(conn, userID, collectionID)
	if err != nil {
		return nil, err
	}

	canEdit, err := canEditCollection(conn, userID, collection)
	if err != nil {
		return nil, err
	}
	if !canEdit {
		return nil, ErrCollectionForbidden
	}

	return collection, nil
}

func collectionLanguages(conn *gorm.DB, collectionIDs []uuid.UUID) (map[uuid.UUID][]enums.Language, error) {
	result := make(map[uuid.UUID][]enums.Language)
	if len(collectionIDs) == 0 {
		return result, nil
	}

	type row struct {
		CollectionID uuid.UUID
		Language     enums.Language
	}

	var rows []row
	err := conn.
		Table("collection_translations AS ct").
		Select("ct.collection_id AS collection_id, words.language AS language").
		Joins("JOIN translations ON translations.id = ct.translation_id").
		Joins("JOIN words ON words.id = translations.original_id OR words.id = translations.translation_id").
		Where("ct.collection_id IN ?", collectionIDs).
		Group("ct.collection_id, words.language").
		Order("ct.collection_id, words.language").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		result[r.CollectionID] = append(result[r.CollectionID], r.Language)
	}
	return result, nil
}

func countByCollection(conn *gorm.DB, table string, collectionIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	result := make(map[uuid.UUID]int)
	if len(collectionIDs) == 0 {
		return result, nil
	}

	type row struct {
		CollectionID uuid.UUID
		Count        int
	}

	var rows []row
	err := conn.
		Table(table).
		Select("collection_id, COUNT(*) AS count").
		Where("collection_id IN ?", collectionIDs).
		Group("collection_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		result[r.CollectionID] = r.Count
	}
	return result, nil
}

func ListCollections(userID uint, page, pageSize int, search string, languageFilter []enums.Language) (*CollectionListResponse, error) {
	if page <= 0 {
		return nil, ErrInvalidPage
	}

	if pageSize < 1 || pageSize > 1000 {
		return nil, ErrInvalidPageSize
	}

	normalizedSearch := strings.TrimSpace(search)
	searchPattern := "%" + normalizedSearch + "%"

	viewerIsAdmin, err := userIsAdmin(db.DB, userID)
	if err != nil {
		return nil, err
	}

	baseQuery := func() *gorm.DB {
		memberSubquery := db.DB.Model(&models.CollectionMember{}).
			Select("collection_id").
			Where("user_id = ?", userID)

		userAddSubquery := db.DB.Model(&models.CollectionUserAdd{}).
			Select("collection_id, COUNT(*) AS add_count").
			Group("collection_id")

		query := db.DB.Model(&models.Collection{}).
			Joins("LEFT JOIN (?) AS add_counts ON add_counts.collection_id = collections.id", userAddSubquery).
			Where("collections.deleted_at IS NULL")

		if !viewerIsAdmin {
			query = query.Where(
				"(collections.is_admin = ? AND collections.is_published = ?) OR collections.owner_id = ? OR collections.id IN (?)",
				true, true, userID, memberSubquery,
			)
		}

		if normalizedSearch != "" {
			query = query.Where("collections.title ILIKE ?", searchPattern)
		}

		if len(languageFilter) > 0 {
			languageSubquery := db.DB.
				Table("collection_translations AS ct").
				Select("ct.collection_id").
				Joins("JOIN translations ON translations.id = ct.translation_id").
				Joins("JOIN words ON words.id = translations.original_id OR words.id = translations.translation_id").
				Where("words.language IN ?", languageFilter)

			query = query.Where("collections.id IN (?)", languageSubquery)
		}

		return query
	}

	var total int64
	if err := baseQuery().Count(&total).Error; err != nil {
		return nil, err
	}

	type collectionWithOwner struct {
		models.Collection
		OwnerUsername string `gorm:"column:owner_username"`
	}

	var collections []collectionWithOwner
	offset := (page - 1) * pageSize
	if err := baseQuery().
		Select("collections.*, users.username as owner_username").
		Joins("LEFT JOIN users ON users.id = collections.owner_id").
		Order("COALESCE(add_counts.add_count, 0) DESC, collections.title ASC, collections.id ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&collections).Error; err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(collections))
	for i := range collections {
		ids = append(ids, collections[i].ID)
	}

	languages, err := collectionLanguages(db.DB, ids)
	if err != nil {
		return nil, err
	}

	counts, err := countByCollection(db.DB, "collection_translations", ids)
	if err != nil {
		return nil, err
	}

	userAddCounts, err := countByCollection(db.DB, "collection_user_adds", ids)
	if err != nil {
		return nil, err
	}

	summaries := make([]CollectionSummary, 0, len(collections))
	for i := range collections {
		collection := collections[i]
		langs := languages[collection.Collection.ID]
		if langs == nil {
			langs = []enums.Language{}
		}

		summary := CollectionSummary{
			ID:               collection.ID,
			Title:            collection.Title,
			IsAdmin:          collection.IsAdmin,
			IsOwner:          collection.IsOwnedBy(userID),
			IsPublished:      collection.IsPublished,
			Languages:        langs,
			TranslationCount: counts[collection.Collection.ID],
			UserAddCount:     userAddCounts[collection.Collection.ID],
			CreatedAt:        collection.CreatedAt,
		}
		if !collection.IsAdmin && collection.OwnerUsername != "" {
			summary.OwnerUsername = collection.OwnerUsername
		}
		summaries = append(summaries, summary)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return &CollectionListResponse{
		Data: summaries,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func GetCollection(userID uint, collectionID uuid.UUID) (*CollectionDetail, error) {
	collection, err := getAccessibleCollection(db.DB, userID, collectionID)
	if err != nil {
		return nil, err
	}

	var translations []models.Translation
	if err := db.DB.
		Model(&models.Translation{}).
		Joins("JOIN collection_translations ct ON ct.translation_id = translations.id").
		Where("ct.collection_id = ?", collectionID).
		Preload("Original").
		Preload("Translation").
		Order("ct.position ASC, translations.id ASC").
		Find(&translations).Error; err != nil {
		return nil, err
	}

	languageSet := make(map[enums.Language]bool)
	languages := make([]enums.Language, 0)
	for i := range translations {
		for _, word := range []*models.Word{translations[i].Original, translations[i].Translation} {
			if word == nil {
				continue
			}
			if !languageSet[word.Language] {
				languageSet[word.Language] = true
				languages = append(languages, word.Language)
			}
		}
	}

	isOwner := collection.IsOwnedBy(userID)

	var userAddCount int64
	db.DB.Model(&models.CollectionUserAdd{}).Where("collection_id = ?", collectionID).Count(&userAddCount)

	detail := &CollectionDetail{
		ID:               collection.ID,
		Title:            collection.Title,
		IsAdmin:          collection.IsAdmin,
		IsOwner:          isOwner,
		IsPublished:      collection.IsPublished,
		Languages:        languages,
		TranslationCount: len(translations),
		UserAddCount:     int(userAddCount),
		CreatedAt:        collection.CreatedAt,
		Translations:     translations,
	}

	if !collection.IsAdmin && collection.OwnerID != nil {
		var ownerUsername string
		if err := db.DB.Model(&models.User{}).Select("username").Where("id = ?", *collection.OwnerID).Scan(&ownerUsername).Error; err == nil && ownerUsername != "" {
			detail.OwnerUsername = ownerUsername
		}
	}

	if isOwner && !collection.IsAdmin {
		detail.InviteToken = collection.InviteToken
	}

	return detail, nil
}

func CreateCollection(userID uint, req CreateCollectionRequest) (*CollectionDetail, error) {
	title := truncateTitle(strings.TrimSpace(req.Title))
	if title == "" {
		return nil, ErrCollectionTitleRequired
	}

	if req.IsAdmin {
		isAdmin, err := userIsAdmin(db.DB, userID)
		if err != nil {
			return nil, err
		}
		if !isAdmin {
			return nil, ErrCollectionAdminForbidden
		}
	}

	token, err := GenerateInviteToken()
	if err != nil {
		return nil, err
	}

	owner := userID
	collection := models.Collection{
		Title:       title,
		OwnerID:     &owner,
		IsAdmin:     req.IsAdmin,
		IsPublished: true,
		InviteToken: token,
	}

	if err := db.DB.Create(&collection).Error; err != nil {
		return nil, err
	}

	return GetCollection(userID, collection.ID)
}

func DeleteCollection(userID uint, collectionID uuid.UUID) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		collection, err := getEditableCollection(tx, userID, collectionID)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		return tx.Model(collection).Update("deleted_at", now).Error
	})
}

func addInlineTranslation(
	tx *gorm.DB,
	collectionID uuid.UUID,
	ownerID uint,
	source enums.TranslationSource,
	original string,
	originalLanguage enums.Language,
	translation string,
	translationLanguage enums.Language,
) error {
	original, translation = utils.NormalizeTranslationPairCasing(
		original,
		string(originalLanguage),
		translation,
		string(translationLanguage),
	)

	originalWord, err := GetOrCreateWord(tx, original, originalLanguage)
	if err != nil {
		return err
	}

	translatedWord, err := GetOrCreateWord(tx, translation, translationLanguage)
	if err != nil {
		return err
	}

	var existing models.Translation
	result := tx.
		Where("(original_id = ? AND translation_id = ?) OR (original_id = ? AND translation_id = ?)",
			originalWord.ID, translatedWord.ID, translatedWord.ID, originalWord.ID).
		Where("source = ?", source).
		Where("user_id = ?", ownerID).
		First(&existing)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		existing = models.Translation{
			OriginalID:    originalWord.ID,
			TranslationID: translatedWord.ID,
			Source:        source,
			UserID:        &ownerID,
		}
		if err := tx.Create(&existing).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	}

	// NullInt64 because MAX(position) is NULL for an empty collection, distinct from position 0.
	var maxPosition sql.NullInt64
	if err := tx.Model(&models.CollectionTranslation{}).
		Where("collection_id = ?", collectionID).
		Select("MAX(position)").
		Scan(&maxPosition).Error; err != nil {
		return err
	}
	nextPosition := 0
	if maxPosition.Valid {
		nextPosition = int(maxPosition.Int64) + 1
	}

	link := models.CollectionTranslation{
		CollectionID:  collectionID,
		TranslationID: existing.ID,
		Position:      nextPosition,
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error
}

func AddTranslationToCollection(userID uint, collectionID uuid.UUID, req AddCollectionTranslationRequest) (*CollectionDetail, error) {
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		collection, err := getEditableCollection(tx, userID, collectionID)
		if err != nil {
			return err
		}

		ownerID := userID
		if collection.OwnerID != nil {
			ownerID = *collection.OwnerID
		}

		return addInlineTranslation(
			tx, collectionID, ownerID, enums.TranslationSourceUser,
			req.Original, req.OriginalLanguage,
			req.Translation, req.TranslationLanguage,
		)
	})
	if err != nil {
		return nil, err
	}

	return GetCollection(userID, collectionID)
}

func GenerateCollection(userID uint, prompt string) (*CollectionDetail, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, ErrAIPromptRequired
	}

	isAdmin, err := userIsAdmin(db.DB, userID)
	if err != nil {
		return nil, err
	}

	generated, err := openrouter.NewClient().GenerateCollection(prompt, enums.AllLanguages())
	if err != nil {
		if errors.Is(err, openrouter.ErrNotConfigured) {
			return nil, ErrAIGenerationUnavailable
		}
		// Sanitize the API key before logging; never leak the raw OpenRouter error to the client.
		logMsg := err.Error()
		if key := config.GetOpenRouterApiKey(); key != "" {
			logMsg = strings.ReplaceAll(logMsg, key, "***")
		}
		logger.L().Errorw("openrouter request failed", "error", logMsg, "model", config.GetOpenRouterModel())
		return nil, ErrAIGenerationFailed
	}

	allowed := make(map[string]bool)
	for _, language := range enums.AllLanguages() {
		allowed[language] = true
	}

	type pair struct {
		original            string
		originalLanguage    enums.Language
		translation         string
		translationLanguage enums.Language
	}

	pairs := make([]pair, 0, len(generated.Translations))
	for _, item := range generated.Translations {
		original := strings.TrimSpace(item.Original)
		translation := strings.TrimSpace(item.Translation)
		originalLanguage := strings.ToLower(strings.TrimSpace(item.OriginalLanguage))
		translationLanguage := strings.ToLower(strings.TrimSpace(item.TranslationLanguage))

		if original == "" || translation == "" {
			continue
		}
		if originalLanguage == translationLanguage {
			continue
		}
		if !allowed[originalLanguage] || !allowed[translationLanguage] {
			continue
		}

		pairs = append(pairs, pair{
			original:            original,
			originalLanguage:    enums.Language(originalLanguage),
			translation:         translation,
			translationLanguage: enums.Language(translationLanguage),
		})
	}

	if len(pairs) == 0 {
		return nil, ErrAIGenerationFailed
	}

	title := truncateTitle(strings.TrimSpace(generated.Title))
	if title == "" {
		title = "AI Collection"
	}

	token, err := GenerateInviteToken()
	if err != nil {
		return nil, err
	}

	owner := userID
	collection := models.Collection{
		Title:       title,
		OwnerID:     &owner,
		IsAdmin:     isAdmin,
		IsPublished: !isAdmin,
		InviteToken: token,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&collection).Error; err != nil {
			return err
		}

		for _, p := range pairs {
			if err := addInlineTranslation(tx, collection.ID, owner, enums.TranslationSourceLLM, p.original, p.originalLanguage, p.translation, p.translationLanguage); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return GetCollection(userID, collection.ID)
}

func SetCollectionIsPublished(userID uint, collectionID uuid.UUID, isPublished bool) (*CollectionDetail, error) {
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		collection, err := getEditableCollection(tx, userID, collectionID)
		if err != nil {
			return err
		}

		return tx.Model(collection).Update("is_published", isPublished).Error
	})
	if err != nil {
		return nil, err
	}

	return GetCollection(userID, collectionID)
}

func RemoveTranslationFromCollection(userID uint, collectionID, translationID uuid.UUID) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := getEditableCollection(tx, userID, collectionID); err != nil {
			return err
		}

		return tx.
			Where("collection_id = ? AND translation_id = ?", collectionID, translationID).
			Delete(&models.CollectionTranslation{}).Error
	})
}

type ReorderCollectionTranslationsRequest struct {
	TranslationIDs []uuid.UUID `json:"translation_ids" binding:"required"`
}

func ReorderCollectionTranslations(userID uint, collectionID uuid.UUID, translationIDs []uuid.UUID) (*CollectionDetail, error) {
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := getEditableCollection(tx, userID, collectionID); err != nil {
			return err
		}

		for position, translationID := range translationIDs {
			if err := tx.Model(&models.CollectionTranslation{}).
				Where("collection_id = ? AND translation_id = ?", collectionID, translationID).
				Update("position", position).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return GetCollection(userID, collectionID)
}

func AddCollectionToVocabulary(userID uint, collectionID uuid.UUID, translationIDs []uuid.UUID) (*AddCollectionToVocabularyResult, error) {
	if _, err := getAccessibleCollection(db.DB, userID, collectionID); err != nil {
		return nil, err
	}

	query := db.DB.
		Table("collection_translations").
		Where("collection_id = ?", collectionID)
	if len(translationIDs) > 0 {
		query = query.Where("translation_id IN ?", translationIDs)
	}

	var selectedIDs []uuid.UUID
	if err := query.
		Order("position ASC, translation_id ASC").
		Pluck("translation_id", &selectedIDs).Error; err != nil {
		return nil, err
	}

	result := &AddCollectionToVocabularyResult{Total: len(selectedIDs)}
	for _, translationID := range selectedIDs {
		if _, err := CreateVocabularyByTranslation(userID, translationID); err != nil {
			if VocabularyAlreadyExistsError(err) {
				result.Skipped++
				continue
			}
			return nil, err
		}
		result.Added++
	}

	userAdd := models.CollectionUserAdd{CollectionID: collectionID, UserID: userID}
	if err := db.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&userAdd).Error; err != nil {
		return nil, err
	}

	var userAddCount int64
	db.DB.Model(&models.CollectionUserAdd{}).Where("collection_id = ?", collectionID).Count(&userAddCount)
	result.UserAddCount = int(userAddCount)

	return result, nil
}

type UpdateCollectionTitleRequest struct {
	Title string `json:"title" binding:"required"`
}

func UpdateCollectionTitle(userID uint, collectionID uuid.UUID, req UpdateCollectionTitleRequest) (*CollectionDetail, error) {
	title := truncateTitle(strings.TrimSpace(req.Title))
	if title == "" {
		return nil, ErrCollectionTitleRequired
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		collection, err := getEditableCollection(tx, userID, collectionID)
		if err != nil {
			return err
		}

		return tx.Model(collection).Update("title", title).Error
	})
	if err != nil {
		return nil, err
	}

	return GetCollection(userID, collectionID)
}

func JoinCollectionByToken(userID uint, token string) (*CollectionDetail, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInvalidInviteToken
	}

	var collection models.Collection
	err := db.DB.Where("invite_token = ? AND deleted_at IS NULL", token).First(&collection).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidInviteToken
		}
		return nil, err
	}

	if !collection.IsOwnedBy(userID) && !collection.IsAdmin {
		member := models.CollectionMember{CollectionID: collection.ID, UserID: userID}
		if err := db.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
			return nil, err
		}
	}

	return GetCollection(userID, collection.ID)
}
