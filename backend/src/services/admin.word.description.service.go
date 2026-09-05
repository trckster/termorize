package services

import (
	"errors"
	"strings"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidDescriptionModel = errors.New("invalid description model")
var ErrDescriptionPreviewConflict = errors.New("description changed or preview expired; generate a new preview")

// Keep automatic generation on config.GetOpenRouterModel(). These choices are admin-only.
var AdminDescriptionModels = []struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Tier string `json:"tier"`
}{
	{"google/gemini-2.5-flash", "Gemini 2.5 Flash", "basic"},
	{"moonshotai/kimi-k2.6", "Kimi K2.6", "medium"},
	{"openai/gpt-5.6-sol", "GPT-5.6 Sol", "smart"},
}

type AdminWordDescription struct {
	ID          uuid.UUID      `json:"id"`
	WordID      uuid.UUID      `json:"word_id"`
	Word        string         `json:"word"`
	Language    enums.Language `json:"language"`
	Model       string         `json:"model"`
	Description string         `json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	ApprovedAt  *time.Time     `json:"approved_at"`
}

type AdminWordDescriptionsResponse struct {
	Data       []AdminWordDescription `json:"data"`
	Pagination Pagination             `json:"pagination"`
}

type WordDescriptionPreview struct {
	ID                  uuid.UUID `gorm:"default:gen_random_uuid()" json:"id"`
	DescriptionID       uuid.UUID `json:"-"`
	UserID              uint      `json:"-"`
	Model               string    `json:"model"`
	Description         string    `json:"description"`
	OriginalDescription string    `json:"-"`
	OriginalModel       string    `json:"-"`
	OriginalCreatedAt   time.Time `json:"-"`
	CreatedAt           time.Time `json:"created_at"`
}

func GetWordDescriptionsForAdmin(page, pageSize int, search string) (*AdminWordDescriptionsResponse, error) {
	if page < 1 {
		return nil, ErrInvalidPage
	}
	if pageSize < 1 || pageSize > 100 {
		return nil, ErrInvalidPageSize
	}
	query := db.DB.Table("word_descriptions AS descriptions").Joins("JOIN words ON words.id = descriptions.word_id")
	if search = strings.TrimSpace(search); search != "" {
		query = query.Where("words.word ILIKE ? OR descriptions.description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	data := make([]AdminWordDescription, 0, pageSize)
	if err := query.Select("descriptions.*, words.word, words.language").
		Order("descriptions.created_at DESC, descriptions.id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Scan(&data).Error; err != nil {
		return nil, err
	}
	return &AdminWordDescriptionsResponse{Data: data, Pagination: Pagination{Page: page, PageSize: pageSize, Total: total, TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize))}}, nil
}

func PreviewWordDescriptionForAdmin(id uuid.UUID, userID uint, model string) (*WordDescriptionPreview, error) {
	valid := false
	for _, option := range AdminDescriptionModels {
		if model == option.ID {
			valid = true
		}
	}
	if !valid {
		return nil, ErrInvalidDescriptionModel
	}
	var existing models.WordDescription
	if err := db.DB.Preload("Word").First(&existing, "id = ?", id).Error; err != nil {
		return nil, err
	}
	description, err := generateValidatedDescription(*existing.Word, openrouter.NewClientWithModel(model))
	if err != nil {
		return nil, err
	}
	// Discarded previews expire and are removed on subsequent regeneration.
	if err := db.DB.Where("created_at < ?", time.Now().Add(-24*time.Hour)).Delete(&WordDescriptionPreview{}).Error; err != nil {
		return nil, err
	}
	preview := WordDescriptionPreview{DescriptionID: id, UserID: userID, Model: model, Description: description,
		OriginalDescription: existing.Description, OriginalModel: existing.Model, OriginalCreatedAt: existing.CreatedAt}
	if err := db.DB.Create(&preview).Error; err != nil {
		return nil, err
	}
	return &preview, nil
}

func ApproveWordDescriptionForAdmin(id, previewID uuid.UUID, userID uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var preview WordDescriptionPreview
		if err := tx.Where("id = ? AND description_id = ? AND user_id = ?", previewID, id, userID).First(&preview).Error; err != nil {
			return err
		}
		if time.Since(preview.CreatedAt) > 24*time.Hour {
			return ErrDescriptionPreviewConflict
		}
		var existing models.WordDescription
		if err := tx.First(&existing, "id = ?", id).Error; err != nil {
			return err
		}
		lockKey := "word-description:" + existing.WordID.String() + ":" + config.GetOpenRouterModel()
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&existing, "id = ?", id).Error; err != nil {
			return err
		}
		if existing.Description != preview.OriginalDescription || existing.Model != preview.OriginalModel || !existing.CreatedAt.Equal(preview.OriginalCreatedAt) {
			return ErrDescriptionPreviewConflict
		}
		now := time.Now()
		replacement := models.WordDescription{WordID: existing.WordID, Model: preview.Model, Description: preview.Description, CreatedAt: now, ApprovedAt: &now}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "word_id"}, {Name: "model"}}, DoUpdates: clause.Assignments(map[string]any{"description": preview.Description, "created_at": now, "approved_at": now})}).Create(&replacement).Error; err != nil {
			return err
		}
		if existing.Model != preview.Model {
			if err := tx.Delete(&existing).Error; err != nil {
				return err
			}
		}
		return tx.Where("description_id = ?", id).Delete(&WordDescriptionPreview{}).Error
	})
}
