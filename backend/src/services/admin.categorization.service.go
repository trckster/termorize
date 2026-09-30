package services

import (
	"context"
	"errors"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidPartOfSpeech = errors.New("invalid part of speech")
var ErrPartOfSpeechPermanent = errors.New("this word already has a permanent category; refresh the list")

type UnknownWordsResponse struct {
	Data       []models.Word `json:"data"`
	Pagination Pagination    `json:"pagination"`
}

func categorizationPagination(page, pageSize int) error {
	if page <= 0 {
		return ErrInvalidPage
	}
	if pageSize < 1 || pageSize > 100 {
		return ErrInvalidPageSize
	}
	return nil
}

func GetUnknownWords(ctx context.Context, page, pageSize int) (*UnknownWordsResponse, error) {
	if err := categorizationPagination(page, pageSize); err != nil {
		return nil, err
	}
	query := db.DB.WithContext(ctx).Model(&models.Word{}).Where("part_of_speech = ?", enums.PartOfSpeechUnknown)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	words := make([]models.Word, 0, pageSize)
	if err := query.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&words).Error; err != nil {
		return nil, err
	}
	return &UnknownWordsResponse{Data: words, Pagination: Pagination{Page: page, PageSize: pageSize, Total: total, TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize))}}, nil
}

func GetMismatchedVocabulary(ctx context.Context, page, pageSize int) (*VocabularyListResponse, error) {
	if err := categorizationPagination(page, pageSize); err != nil {
		return nil, err
	}
	query := db.DB.WithContext(ctx).Model(&models.Vocabulary{}).
		Joins("JOIN translations AS t ON t.id = vocabulary.translation_id").
		Joins("JOIN words AS original ON original.id = t.original_id").
		Joins("JOIN words AS translated ON translated.id = t.translation_id").
		Where("vocabulary.deleted_at IS NULL").
		Where("original.part_of_speech IS NOT NULL AND translated.part_of_speech IS NOT NULL AND original.part_of_speech <> translated.part_of_speech")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	rows := make([]models.Vocabulary, 0, pageSize)
	if err := query.Select("vocabulary.*").Preload("Translation.Original").Preload("Translation.Translation").
		Order("vocabulary.created_at DESC, vocabulary.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	return &VocabularyListResponse{Data: rows, Pagination: Pagination{Page: page, PageSize: pageSize, Total: total, TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize))}}, nil
}

func SetWordPartOfSpeech(ctx context.Context, id uuid.UUID, category enums.PartOfSpeech) (*models.Word, error) {
	if !category.Valid() {
		return nil, ErrInvalidPartOfSpeech
	}
	var word models.Word
	result := db.DB.WithContext(ctx).Model(&word).Clauses(clause.Returning{}).
		Where("id = ? AND (part_of_speech IS NULL OR part_of_speech = ?)", id, enums.PartOfSpeechUnknown).
		Update("part_of_speech", category)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		if err := db.DB.WithContext(ctx).First(&word, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, gorm.ErrRecordNotFound
			}
			return nil, err
		}
		return nil, ErrPartOfSpeechPermanent
	}
	return &word, nil
}
