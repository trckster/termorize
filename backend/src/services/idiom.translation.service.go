package services

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"strings"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/models"
	"unicode"
	"unicode/utf8"
)

var ErrInvalidIdiomTarget = errors.New("invalid idiom translation language")

// IdiomTargetLanguage uses the existing saved translator pair, just like the web action.
func IdiomTargetLanguage(settings models.UserSettings, language enums.Language) enums.Language {
	settings = settings.WithDefaults()
	if settings.TranslationSourceLanguage != language {
		return settings.TranslationSourceLanguage
	}
	return settings.TranslationTargetLanguage
}

// TranslateDailyIdiom accepts a persisted selection, never arbitrary text. Its cache
// includes only this idiom-specific LLM source, excluding Google and generic LLM output.
func TranslateDailyIdiom(ctx context.Context, id uuid.UUID, target enums.Language) (*TranslationResult, error) {
	var selection models.DailyIdiom
	if err := db.DB.WithContext(ctx).First(&selection, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if !enums.IsSupportedLanguage(target) || target == selection.Language {
		return nil, ErrInvalidIdiomTarget
	}
	var word models.Word
	if err := db.DB.WithContext(ctx).First(&word, "id = ?", selection.WordID).Error; err != nil {
		return nil, err
	}
	find := func(conn *gorm.DB) (*models.Translation, error) {
		var result models.Translation
		err := conn.Joins("JOIN words AS target_word ON target_word.id = translations.translation_id").
			Preload("Original").Preload("Translation").
			Where("original_id = ? AND target_word.language = ? AND source = ?", word.ID, target, enums.TranslationSourceIdiomLLM).First(&result).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return &result, err
	}
	cached, err := find(db.DB.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return translationResult(cached), nil
	}
	text, err := openrouter.NewClient().TranslateIdiom(ctx, word.Word, word.Language.DisplayName(), target.DisplayName())
	if err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" || !utf8.ValidString(text) || utf8.RuneCountInString(text) > 1000 || strings.ContainsFunc(text, unicode.IsControl) {
		return nil, errors.New("invalid idiom translation text")
	}
	var result *TranslationResult
	err = db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockWordWrites(tx); err != nil {
			return err
		}
		cached, err := find(tx)
		if err != nil {
			return err
		}
		if cached != nil {
			result = translationResult(cached)
			return nil
		}
		translated, err := GetOrCreateWord(tx, text, target)
		if err != nil {
			return err
		}
		translation := models.Translation{OriginalID: word.ID, TranslationID: translated.ID, Source: enums.TranslationSourceIdiomLLM}
		if err := tx.Create(&translation).Error; err != nil {
			return err
		}
		translation.Original = &word
		translation.Translation = translated
		result = translationResult(&translation)
		return nil
	})
	return result, err
}

func GetDailyIdiomSelection(id uuid.UUID) (*models.DailyIdiom, error) {
	var selection models.DailyIdiom
	err := db.DB.First(&selection, "id = ?", id).Error
	return &selection, err
}
