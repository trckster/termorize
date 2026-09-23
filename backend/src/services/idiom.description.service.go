package services

import (
	"context"
	"errors"
	"strings"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/models"
	"unicode"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetDailyIdiomDescription(ctx context.Context, selectionID uuid.UUID) (*models.WordDescription, error) {
	var word models.Word
	if err := db.DB.WithContext(ctx).Table("words").Select("words.*").
		Joins("JOIN daily_idioms ON daily_idioms.word_id = words.id").
		Where("daily_idioms.id = ? AND words.type = ?", selectionID, enums.TypeIdiom).
		Take(&word).Error; err != nil {
		return nil, err
	}
	var cached models.WordDescription
	if err := idiomDescriptionCacheQuery(db.DB.WithContext(ctx), word.ID).Take(&cached).Error; err == nil {
		return &cached, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// This word lock coordinates generators across processes after daily selection has committed.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", idiomDescriptionLockKey(word.ID)).Error; err != nil {
			return err
		}
		if err := idiomDescriptionCacheQuery(tx, word.ID).Take(&cached).Error; err == nil {
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		text, err := generateValidatedIdiomDescription(ctx, word, openrouter.NewClient())
		if err != nil {
			return err
		}
		cached = models.WordDescription{WordID: word.ID, Model: config.GetOpenRouterModel(), Description: text}
		return tx.Create(&cached).Error
	})
	if err != nil {
		return nil, err
	}
	return &cached, nil
}

func idiomDescriptionCacheQuery(tx *gorm.DB, wordID uuid.UUID) *gorm.DB {
	return tx.Where("word_id = ?", wordID).
		Order("(translation_word_id IS NULL) DESC, created_at DESC, id DESC")
}

func idiomDescriptionLockKey(wordID uuid.UUID) string {
	return "idiom-description:" + wordID.String()
}

func generateValidatedIdiomDescription(ctx context.Context, word models.Word, client openrouter.Client) (string, error) {
	if word.Type != enums.TypeIdiom {
		return "", ErrInvalidDescriptionTranslation
	}
	generated, err := client.GenerateIdiomDescription(ctx, word.Word, word.Language.DisplayName())
	if err != nil {
		return "", errors.Join(ErrDescriptionGenerationFailed, err)
	}
	if generated == nil {
		return "", ErrDescriptionGenerationFailed
	}
	text := strings.TrimSpace(generated.Description)
	if text == "" || len([]rune(text)) > maxDescriptionRunes || text != strings.ToLower(text) ||
		strings.HasSuffix(text, ".") || strings.ContainsAny(text, "\r\n") ||
		strings.IndexFunc(text, unicode.IsControl) >= 0 || descriptionMentionsAnswer(text, word.Word) ||
		strings.IndexFunc(text, unicode.IsLetter) < 0 {
		return "", ErrDescriptionGenerationFailed
	}
	for _, meaning := range strings.Split(text, ";") {
		for _, item := range strings.Split(meaning, ",") {
			if strings.TrimSpace(item) == "" {
				return "", ErrDescriptionGenerationFailed
			}
		}
	}
	detected, supported, err := DetectLanguage(text)
	if err != nil {
		return "", errors.Join(ErrDescriptionGenerationFailed, err)
	}
	if !supported || detected != word.Language {
		return "", ErrDescriptionGenerationFailed
	}
	return text, nil
}
