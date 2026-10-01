package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"termorize/src/classification"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidPartOfSpeech = errors.New("invalid part of speech")

var categorizationRunner struct {
	sync.RWMutex
	worker *classification.Worker
}

func SetCategorizationWorker(worker *classification.Worker) func() {
	categorizationRunner.Lock()
	previous := categorizationRunner.worker
	categorizationRunner.worker = worker
	categorizationRunner.Unlock()
	return func() {
		categorizationRunner.Lock()
		categorizationRunner.worker = previous
		categorizationRunner.Unlock()
	}
}

func categorizationWorker() *classification.Worker {
	categorizationRunner.RLock()
	defer categorizationRunner.RUnlock()
	return categorizationRunner.worker
}

type CategorizationStats struct {
	Total       int64                 `json:"total"`
	Categorized int64                 `json:"categorized"`
	Unknown     int64                 `json:"unknown"`
	Pending     int64                 `json:"pending"`
	Worker      *classification.Stats `json:"worker" gorm:"-"`
}

func GetCategorizationStats(ctx context.Context) (*CategorizationStats, error) {
	var stats CategorizationStats
	if err := db.DB.WithContext(ctx).Model(&models.Word{}).Select(`count(*) AS total,
		count(*) FILTER (WHERE part_of_speech IS NULL) AS pending,
		count(*) FILTER (WHERE part_of_speech = 'unknown') AS unknown,
		count(*) FILTER (WHERE part_of_speech IS NOT NULL AND part_of_speech <> 'unknown') AS categorized`).Scan(&stats).Error; err != nil {
		return nil, err
	}
	if worker := categorizationWorker(); worker != nil {
		status := worker.Stats()
		stats.Worker = &status
	}
	return &stats, nil
}

func RestartCategorization(ctx context.Context) error {
	worker := categorizationWorker()
	if worker == nil {
		return classification.ErrUnavailable
	}
	return worker.Restart(ctx)
}

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
	return getCategoryWords(ctx, page, pageSize, "", true)
}

func GetCategoryWords(ctx context.Context, page, pageSize int, search string) (*UnknownWordsResponse, error) {
	return getCategoryWords(ctx, page, pageSize, search, false)
}

func getCategoryWords(ctx context.Context, page, pageSize int, search string, unresolved bool) (*UnknownWordsResponse, error) {
	if err := categorizationPagination(page, pageSize); err != nil {
		return nil, err
	}
	query := db.DB.WithContext(ctx).Model(&models.Word{})
	if unresolved {
		query = query.Where("part_of_speech IS NULL OR part_of_speech = ?", enums.PartOfSpeechUnknown)
	}
	if search = strings.TrimSpace(search); search != "" {
		search = strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(search)
		query = query.Where("word ILIKE ?", "%"+search+"%")
	}
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
		Where("id = ?", id).
		Updates(map[string]any{"part_of_speech": category, "category_revision": gorm.Expr("category_revision + 1")})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &word, nil
}
