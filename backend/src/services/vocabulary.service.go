package services

import (
	"encoding/json"
	"errors"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CreateVocabularyRequest struct {
	Word1     string         `json:"word_1" binding:"required"`
	Word2     string         `json:"word_2" binding:"required"`
	Language1 enums.Language `json:"language_1" binding:"required,enum=Language"`
	Language2 enums.Language `json:"language_2" binding:"required,enum=Language"`
}

type VocabularyResponse struct {
	ID            uuid.UUID                `json:"id"`
	UserID        uint                     `json:"user_id"`
	TranslationID uuid.UUID                `json:"translation_id"`
	Progress      []models.ProgressEntry   `json:"progress"`
	CreatedAt     time.Time                `json:"created_at"`
	MasteredAt    *time.Time               `json:"mastered_at"`
	Translation   *TranslationWithWordsDTO `json:"translation"`
}

type TranslationWithWordsDTO struct {
	ID     uuid.UUID               `json:"id"`
	Word1  *models.Word            `json:"word_1"`
	Word2  *models.Word            `json:"word_2"`
	Source enums.TranslationSource `json:"source"`
	UserID *uint                   `json:"user_id"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type VocabularyListResponse struct {
	Data       []VocabularyResponse `json:"data"`
	Pagination Pagination           `json:"pagination"`
}

func GetOrCreateWord(word string, language enums.Language) (*models.Word, error) {
	var existingWord models.Word
	result := db.DB.Where("word = ? AND language = ?", word, language).First(&existingWord)

	if result.Error == nil {
		return &existingWord, nil
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	newWord := models.Word{
		Word:     word,
		Language: language,
	}

	if err := db.DB.Create(&newWord).Error; err != nil {
		return nil, err
	}

	return &newWord, nil
}

func CreateVocabulary(userID uint, req CreateVocabularyRequest) (*models.Vocabulary, error) {
	if req.Language1 == req.Language2 {
		return nil, errors.New("languages must differ")
	}

	word1, err := GetOrCreateWord(req.Word1, req.Language1)
	if err != nil {
		return nil, err
	}

	word2, err := GetOrCreateWord(req.Word2, req.Language2)
	if err != nil {
		return nil, err
	}

	translation := models.Translation{
		Word1ID: word1.ID,
		Word2ID: word2.ID,
		Source:  enums.TranslationSourceUser,
		UserID:  &userID,
	}

	if err := db.DB.Create(&translation).Error; err != nil {
		return nil, err
	}

	progressJSON, _ := json.Marshal([]models.ProgressEntry{{
		Knowledge: 0,
		Type:      enums.KnowledgeTypeTranslation,
	}})

	vocabulary := models.Vocabulary{
		UserID:        userID,
		TranslationID: translation.ID,
		Progress:      datatypes.JSON(progressJSON),
	}

	if err := db.DB.Create(&vocabulary).Error; err != nil {
		return nil, err
	}

	vocabulary.Translation = &translation
	vocabulary.Translation.Word1 = word1
	vocabulary.Translation.Word2 = word2

	return &vocabulary, nil
}

func GetVocabulary(userID uint, page, pageSize int) (*VocabularyListResponse, error) {
	if page < 1 {
		return nil, errors.New("page must be greater than 0")
	}
	if pageSize < 1 || pageSize > 250 {
		return nil, errors.New("page size must be between 1 and 250")
	}

	var vocabularies []models.Vocabulary
	var total int64

	offset := (page - 1) * pageSize

	if err := db.DB.Where("user_id = ?", userID).
		Preload("Translation").
		Preload("Translation.Word1").
		Preload("Translation.Word2").
		Offset(offset).
		Limit(pageSize).
		Order("created_at desc").
		Find(&vocabularies).
		Error; err != nil {
		return nil, err
	}

	if err := db.DB.Model(&models.Vocabulary{}).
		Where("user_id = ?", userID).
		Count(&total).
		Error; err != nil {
		return nil, err
	}

	responses := make([]VocabularyResponse, len(vocabularies))
	for i, vocab := range vocabularies {
		var progress []models.ProgressEntry
		json.Unmarshal(vocab.Progress, &progress)

		translationDTO := &TranslationWithWordsDTO{
			ID:     vocab.Translation.ID,
			Word1:  vocab.Translation.Word1,
			Word2:  vocab.Translation.Word2,
			Source: vocab.Translation.Source,
			UserID: vocab.Translation.UserID,
		}

		responses[i] = VocabularyResponse{
			ID:            vocab.ID,
			UserID:        vocab.UserID,
			TranslationID: vocab.TranslationID,
			Progress:      progress,
			CreatedAt:     vocab.CreatedAt,
			MasteredAt:    vocab.MasteredAt,
			Translation:   translationDTO,
		}
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	return &VocabularyListResponse{
		Data: responses,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func DeleteVocabulary(userID uint, vocabID uuid.UUID) error {
	result := db.DB.Where("id = ? AND user_id = ?", vocabID, userID).Delete(&models.Vocabulary{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("vocabulary item not found")
	}
	return nil
}
