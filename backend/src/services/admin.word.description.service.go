package services

import (
	"errors"
	"strings"
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
var ErrInvalidDescriptionTranslation = errors.New("invalid description translation")

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
	Translation         string         `json:"translation"`
	TranslationLanguage enums.Language `json:"translation_language"`
	ID                  uuid.UUID      `json:"id"`
	WordID              uuid.UUID      `json:"word_id"`
	Word                string         `json:"word"`
	Language            enums.Language `json:"language"`
	Model               string         `json:"model"`
	Description         string         `json:"description"`
	CreatedAt           time.Time      `json:"created_at"`
	ApprovedAt          *time.Time     `json:"approved_at"`
}

type AdminWordDescriptionsResponse struct {
	Data       []AdminWordDescription `json:"data"`
	Pagination Pagination             `json:"pagination"`
}

type WordDescriptionPreview struct {
	TranslationWordID   uuid.UUID      `json:"translation_word_id"`
	Translation         string         `json:"translation"`
	TranslationLanguage enums.Language `json:"translation_language"`
	Model               string         `json:"model"`
	Description         string         `json:"description"`
	OriginalDescription string         `json:"original_description"`
}

func GetWordDescriptionsForAdmin(page, pageSize int, search string) (*AdminWordDescriptionsResponse, error) {
	if page < 1 {
		return nil, ErrInvalidPage
	}
	if pageSize < 1 || pageSize > 100 {
		return nil, ErrInvalidPageSize
	}
	query := db.DB.Table("word_descriptions AS descriptions").Joins("JOIN words ON words.id = descriptions.word_id").
		Joins("LEFT JOIN words AS translations ON translations.id = descriptions.translation_word_id")
	if search = strings.TrimSpace(search); search != "" {
		query = query.Where("words.word ILIKE ? OR descriptions.description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	data := make([]AdminWordDescription, 0, pageSize)
	if err := query.Select("descriptions.*, words.word, words.language, translations.word AS translation, translations.language AS translation_language").
		Order("descriptions.created_at DESC, descriptions.id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Scan(&data).Error; err != nil {
		return nil, err
	}
	return &AdminWordDescriptionsResponse{Data: data, Pagination: Pagination{Page: page, PageSize: pageSize, Total: total, TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize))}}, nil
}

func PreviewWordDescriptionForAdmin(id uuid.UUID, model string) (*WordDescriptionPreview, error) {
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
	if err := db.DB.Preload("Word").Preload("TranslationWord").First(&existing, "id = ?", id).Error; err != nil {
		return nil, err
	}
	translation := existing.TranslationWord
	if translation == nil {
		// Legacy clues have no recorded meaning. Use the oldest saved translation
		// for the preview and return it so approval preserves that exact context.
		var pair models.Translation
		if err := db.DB.Preload("Original").Preload("Translation").
			Where("original_id = ? OR translation_id = ?", existing.WordID, existing.WordID).
			Order("created_at ASC, id ASC").Take(&pair).Error; err != nil {
			return nil, err
		}
		translation = pair.Translation
		if pair.TranslationID == existing.WordID {
			translation = pair.Original
		}
	}
	description, err := generateValidatedDescription(*existing.Word, *translation, openrouter.NewClientWithModel(model))
	if err != nil {
		return nil, err
	}
	return &WordDescriptionPreview{
		Model: model, Description: description, OriginalDescription: existing.Description,
		TranslationWordID: translation.ID, Translation: translation.Word, TranslationLanguage: translation.Language,
	}, nil
}

func ApproveWordDescriptionForAdmin(id, translationWordID uuid.UUID, model, description string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.WordDescription
		if err := tx.First(&existing, "id = ?", id).Error; err != nil {
			return err
		}
		lockKey := descriptionLockKey(existing.WordID, translationWordID)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&existing, "id = ?", id).Error; err != nil {
			return err
		}
		if existing.TranslationWordID != nil {
			if *existing.TranslationWordID != translationWordID {
				return ErrInvalidDescriptionTranslation
			}
		} else {
			var count int64
			if err := tx.Model(&models.Translation{}).
				Where("(original_id = ? AND translation_id = ?) OR (original_id = ? AND translation_id = ?)",
					existing.WordID, translationWordID, translationWordID, existing.WordID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrInvalidDescriptionTranslation
			}
		}
		now := time.Now()
		replacement := models.WordDescription{WordID: existing.WordID, TranslationWordID: &translationWordID, Model: model, Description: description, CreatedAt: now, ApprovedAt: &now}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "word_id"}, {Name: "translation_word_id"}, {Name: "model"}}, DoUpdates: clause.Assignments(map[string]any{"description": description, "created_at": now, "approved_at": now})}).Create(&replacement).Error; err != nil {
			return err
		}
		if existing.Model != model || existing.TranslationWordID == nil {
			if err := tx.Delete(&existing).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
