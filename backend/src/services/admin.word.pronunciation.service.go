package services

import (
	"errors"
	"strings"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const adminWordPronunciationsMaxPageSize = 100

type AdminWordPronunciation struct {
	ID              uuid.UUID      `json:"id"`
	WordID          uuid.UUID      `json:"word_id"`
	Word            string         `json:"word"`
	Language        enums.Language `json:"language"`
	Model           string         `json:"model"`
	Voice           string         `json:"voice"`
	MIMEType        string         `json:"mime_type"`
	SizeBytes       int64          `json:"size_bytes"`
	HasTelegramFile bool           `json:"has_telegram_file"`
	CreatedAt       time.Time      `json:"created_at"`
}

type AdminWordPronunciationsResponse struct {
	Data       []AdminWordPronunciation `json:"data"`
	Pagination Pagination               `json:"pagination"`
}

func GetWordPronunciationsForAdmin(page, pageSize int, search string) (*AdminWordPronunciationsResponse, error) {
	if page <= 0 {
		return nil, ErrInvalidPage
	}
	if pageSize < 1 || pageSize > adminWordPronunciationsMaxPageSize {
		return nil, ErrInvalidPageSize
	}

	query := db.DB.Table("word_pronunciations AS pronunciations").
		Joins("JOIN words ON words.id = pronunciations.word_id")
	if search = strings.TrimSpace(search); search != "" {
		query = query.Where("words.word ILIKE ?", "%"+search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	data := make([]AdminWordPronunciation, 0, pageSize)
	if err := query.
		Select(`
			pronunciations.id,
			pronunciations.word_id,
			words.word,
			words.language,
			pronunciations.model,
			pronunciations.voice,
			pronunciations.mime_type,
			octet_length(pronunciations.audio) AS size_bytes,
			(pronunciations.telegram_file_id IS NOT NULL) AS has_telegram_file,
			pronunciations.created_at
		`).
		Order("pronunciations.created_at DESC, pronunciations.id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Scan(&data).Error; err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return &AdminWordPronunciationsResponse{
		Data: data,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func GetWordPronunciationAudioForAdmin(id uuid.UUID) ([]byte, string, error) {
	audio, mimeType, err := GetWordPronunciationAudio(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", ErrWordPronunciationNotFound
	}
	return audio, mimeType, err
}

func RegenerateWordPronunciationForAdmin(id uuid.UUID) (*AdminWordPronunciation, error) {
	var existing models.WordPronunciation
	if err := db.DB.Preload("Word").First(&existing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWordPronunciationNotFound
		}
		return nil, err
	}

	generated, err := generateWordPronunciation(*existing.Word)
	if err != nil {
		return nil, err
	}

	var replacement models.WordPronunciation
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.WordPronunciation{}, "id = ?", id).Error; err != nil {
			return err
		}

		replacement = models.WordPronunciation{
			WordID:   existing.WordID,
			Model:    generated.Model,
			Voice:    generated.Voice,
			Audio:    generated.Audio,
			MIMEType: models.PronunciationMIMETypeMP3,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "word_id"}, {Name: "model"}, {Name: "voice"}},
			DoUpdates: clause.Assignments(map[string]any{
				"audio":            generated.Audio,
				"mime_type":        models.PronunciationMIMETypeMP3,
				"telegram_file_id": nil,
				"created_at":       time.Now(),
			}),
		}).Create(&replacement).Error; err != nil {
			return err
		}

		return tx.Select("id").
			Where("word_id = ? AND model = ? AND voice = ?", existing.WordID, generated.Model, generated.Voice).
			First(&replacement).Error
	})
	if err != nil {
		return nil, err
	}

	return getAdminWordPronunciation(replacement.ID)
}

func getAdminWordPronunciation(id uuid.UUID) (*AdminWordPronunciation, error) {
	var pronunciation AdminWordPronunciation
	err := db.DB.Table("word_pronunciations AS pronunciations").
		Select(`
			pronunciations.id,
			pronunciations.word_id,
			words.word,
			words.language,
			pronunciations.model,
			pronunciations.voice,
			pronunciations.mime_type,
			octet_length(pronunciations.audio) AS size_bytes,
			(pronunciations.telegram_file_id IS NOT NULL) AS has_telegram_file,
			pronunciations.created_at
		`).
		Joins("JOIN words ON words.id = pronunciations.word_id").
		Where("pronunciations.id = ?", id).
		Take(&pronunciation).Error
	if err != nil {
		return nil, err
	}
	return &pronunciation, nil
}
