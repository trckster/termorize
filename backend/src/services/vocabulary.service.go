package services

import (
	"errors"
	"strings"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreateVocabularyRequest struct {
	Original            string         `json:"original" binding:"required"`
	Translation         string         `json:"translation" binding:"required"`
	OriginalLanguage    enums.Language `json:"original_language" binding:"required,enum=Language"`
	TranslationLanguage enums.Language `json:"translation_language" binding:"required,enum=Language,nefield=OriginalLanguage"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type VocabularyListResponse struct {
	Data       []models.Vocabulary `json:"data"`
	Pagination Pagination          `json:"pagination"`
}

const translationAlreadyExistsError = "translation already exists"

func IsTranslationAlreadyExistsError(err error) bool {
	return err != nil && err.Error() == translationAlreadyExistsError
}

func GetOrCreateWord(word string, language enums.Language) (*models.Word, error) {
	normalizedWord := strings.TrimSpace(word)

	var existingWord models.Word
	result := db.DB.Where("LOWER(word) = LOWER(?) AND language = ?", normalizedWord, language).First(&existingWord)

	if result.Error == nil {
		return &existingWord, nil
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	newWord := models.Word{
		Word:     normalizedWord,
		Language: language,
	}

	if err := db.DB.Create(&newWord).Error; err != nil {
		return nil, err
	}

	return &newWord, nil
}

func CreateVocabulary(userID uint, req CreateVocabularyRequest) (*models.Vocabulary, error) {
	originalWord, err := GetOrCreateWord(req.Original, req.OriginalLanguage)
	if err != nil {
		return nil, err
	}

	translatedWord, err := GetOrCreateWord(req.Translation, req.TranslationLanguage)
	if err != nil {
		return nil, err
	}

	translation := models.Translation{
		OriginalID:    originalWord.ID,
		TranslationID: translatedWord.ID,
		Source:        enums.TranslationSourceUser,
		UserID:        &userID,
	}

	if err := db.DB.Create(&translation).Error; err != nil {
		return nil, err
	}

	vocabulary := models.Vocabulary{
		UserID:        userID,
		TranslationID: translation.ID,
		Progress: models.ProgressEntries{{
			Knowledge: 0,
			Type:      enums.KnowledgeTypeTranslation,
		}},
	}

	if err := db.DB.Create(&vocabulary).Error; err != nil {
		return nil, err
	}

	vocabulary.Translation = &translation
	vocabulary.Translation.Original = originalWord
	vocabulary.Translation.Translation = translatedWord

	return &vocabulary, nil
}

func CreateVocabularyByTranslation(userID uint, translationID uuid.UUID) (*models.Vocabulary, error) {
	var translation models.Translation
	if err := db.DB.
		Preload("Original").
		Preload("Translation").
		Where("id = ?", translationID).
		First(&translation).Error; err != nil {
		return nil, err
	}

	var count int64
	if err := db.DB.
		Model(&models.Vocabulary{}).
		Joins("JOIN translations ON translations.id = vocabulary.translation_id").
		Where("vocabulary.user_id = ?", userID).
		Where("(translations.original_id = ? AND translations.translation_id = ?) OR (translations.original_id = ? AND translations.translation_id = ?)", translation.OriginalID, translation.TranslationID, translation.TranslationID, translation.OriginalID).
		Count(&count).Error; err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, errors.New(translationAlreadyExistsError)
	}

	vocabulary := models.Vocabulary{
		UserID:        userID,
		TranslationID: translationID,
		Progress: models.ProgressEntries{{
			Knowledge: 0,
			Type:      enums.KnowledgeTypeTranslation,
		}},
	}

	if err := db.DB.Create(&vocabulary).Error; err != nil {
		return nil, err
	}

	vocabulary.Translation = &translation
	return &vocabulary, nil
}

func GetVocabulary(userID uint, page, pageSize int) (*VocabularyListResponse, error) {
	if page <= 0 {
		return nil, errors.New("page must be greater than 0")
	}

	if pageSize < 1 || pageSize > 1000 {
		return nil, errors.New("page size must be between 1 and 1000")
	}

	var total int64
	if err := db.DB.Model(&models.Vocabulary{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, err
	}

	var vocabularyItems []models.Vocabulary
	offset := (page - 1) * pageSize

	if err := db.DB.
		Where("user_id = ?", userID).
		Preload("Translation").
		Preload("Translation.Original").
		Preload("Translation.Translation").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&vocabularyItems).Error; err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return &VocabularyListResponse{
		Data: vocabularyItems,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func DeleteVocabulary(userID uint, vocabID uuid.UUID) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var vocabulary models.Vocabulary
		if err := tx.Where("id = ? AND user_id = ?", vocabID, userID).First(&vocabulary).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("vocabulary item not found")
			}
			return err
		}

		if err := tx.Delete(&vocabulary).Error; err != nil {
			return err
		}

		var translation models.Translation
		if err := tx.Where("id = ? AND source = ? AND user_id = ?", vocabulary.TranslationID, enums.TranslationSourceUser, userID).
			First(&translation).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		if err := tx.Delete(&translation).Error; err != nil {
			return err
		}

		wordIDs := []uuid.UUID{translation.OriginalID, translation.TranslationID}

		for _, wordID := range wordIDs {
			var wordUsageCount int64
			if err := tx.Model(&models.Translation{}).
				Where("original_id = ? OR translation_id = ?", wordID, wordID).
				Count(&wordUsageCount).Error; err != nil {
				return err
			}

			if wordUsageCount == 0 {
				if err := tx.Where("id = ?", wordID).Delete(&models.Word{}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func DeleteVocabularyByWord(userID uint, word string) (bool, error) {
	normalizedWord := strings.TrimSpace(word)
	if normalizedWord == "" {
		return false, nil
	}

	var vocabulary models.Vocabulary
	err := db.DB.
		Model(&models.Vocabulary{}).
		Select("vocabulary.*").
		Joins("JOIN translations ON translations.id = vocabulary.translation_id").
		Joins("JOIN words AS original_words ON original_words.id = translations.original_id").
		Joins("JOIN words AS translation_words ON translation_words.id = translations.translation_id").
		Where("vocabulary.user_id = ?", userID).
		Where("LOWER(original_words.word) = LOWER(?) OR LOWER(translation_words.word) = LOWER(?)", normalizedWord, normalizedWord).
		Order("vocabulary.created_at DESC").
		First(&vocabulary).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}

		return false, err
	}

	if err := DeleteVocabulary(userID, vocabulary.ID); err != nil {
		return false, err
	}

	return true, nil
}
