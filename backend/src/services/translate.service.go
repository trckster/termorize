package services

import (
	"errors"
	"strings"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/google"
	"termorize/src/models"
	"termorize/src/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TranslationResult struct {
	TranslationID    uuid.UUID
	SourceWordID     uuid.UUID
	TranslatedWordID uuid.UUID
	SourceWord       string
	TranslatedWord   string
	Source           enums.TranslationSource
}

func DetectLanguage(text string) (enums.Language, bool, error) {
	googleClient := google.NewTranslateClient()
	detected, err := googleClient.DetectLanguage(text)
	if err != nil {
		return "", false, err
	}

	language := enums.Language(strings.ToLower(strings.TrimSpace(detected)))
	for _, supported := range enums.AllLanguages() {
		if supported == string(language) {
			return language, true, nil
		}
	}

	return language, false, nil
}

func Translate(fromWord string, fromLanguage enums.Language, toLanguage enums.Language) (*TranslationResult, error) {
	normalizedWord := utils.NormalizeWordCasingForLanguage(fromWord, string(fromLanguage))
	var existingWord models.Word
	lookup := db.DB.Where("LOWER(word) = LOWER(?) AND language = ?", normalizedWord, fromLanguage).First(&existingWord)
	if lookup.Error != nil && !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
		return nil, lookup.Error
	}
	if lookup.Error == nil {
		cached, err := findExistingTranslation(db.DB, existingWord.ID, toLanguage)
		if err != nil {
			return nil, err
		}
		if cached != nil {
			return translationResult(cached), nil
		}
	}

	// Google may be slow; hold the word-write lock only while persisting the result.
	googleClient := google.NewTranslateClient()
	translatedText, err := googleClient.Translate(fromWord, string(fromLanguage), string(toLanguage))
	if err != nil {
		return nil, err
	}
	_, translatedText = utils.NormalizeTranslationPairCasing(fromWord, string(fromLanguage), translatedText, string(toLanguage))

	var result TranslationResult
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		sourceWord, err := GetOrCreateWord(tx, fromWord, fromLanguage)
		if err != nil {
			return err
		}
		existingTranslation, err := findExistingTranslation(tx, sourceWord.ID, toLanguage)
		if err != nil {
			return err
		}
		if existingTranslation != nil {
			result = *translationResult(existingTranslation)
			return nil
		}

		targetWord, err := GetOrCreateWord(tx, translatedText, toLanguage)
		if err != nil {
			return err
		}

		translation := models.Translation{
			OriginalID:    sourceWord.ID,
			TranslationID: targetWord.ID,
			Source:        enums.TranslationSourceGoogle,
		}

		if err := tx.Create(&translation).Error; err != nil {
			return err
		}

		result = TranslationResult{
			TranslationID:    translation.ID,
			SourceWordID:     sourceWord.ID,
			TranslatedWordID: targetWord.ID,
			SourceWord:       sourceWord.Word,
			TranslatedWord:   targetWord.Word,
			Source:           translation.Source,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func translationResult(translation *models.Translation) *TranslationResult {
	return &TranslationResult{
		TranslationID:    translation.ID,
		SourceWordID:     translation.Original.ID,
		TranslatedWordID: translation.Translation.ID,
		SourceWord:       translation.Original.Word,
		TranslatedWord:   translation.Translation.Word,
		Source:           translation.Source,
	}
}

func findExistingTranslation(conn *gorm.DB, sourceWordID uuid.UUID, targetLanguage enums.Language) (*models.Translation, error) {
	var translation models.Translation

	result := conn.
		Joins("JOIN words AS translation_word ON translation_word.id = translations.translation_id").
		Preload("Original").
		Preload("Translation").
		Where("translations.source != ?", enums.TranslationSourceUser).
		Where("original_id = ?", sourceWordID).
		Where("translation_word.language = ?", string(targetLanguage)).
		First(&translation)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &translation, nil
}
