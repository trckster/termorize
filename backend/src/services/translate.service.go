package services

import (
	"errors"
	"strings"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/google"
	"termorize/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TranslationResult struct {
	TranslationID  uuid.UUID
	SourceWord     string
	TranslatedWord string
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

func Translate(fromWord string, fromLanguage enums.Language, toLanguage enums.Language) (string, error) {
	result, err := TranslateWithTranslation(fromWord, fromLanguage, toLanguage)
	if err != nil {
		return "", err
	}

	return result.TranslatedWord, nil
}

func TranslateWithTranslation(fromWord string, fromLanguage enums.Language, toLanguage enums.Language) (*TranslationResult, error) {
	sourceWord, err := GetOrCreateWord(fromWord, fromLanguage)
	if err != nil {
		return nil, err
	}

	existingTranslation, err := findExistingTranslation(sourceWord.ID, toLanguage)
	if err != nil {
		return nil, err
	}

	if existingTranslation != nil {
		return &TranslationResult{TranslationID: existingTranslation.ID, SourceWord: existingTranslation.Original.Word, TranslatedWord: existingTranslation.Translation.Word}, nil
	}

	googleClient := google.NewTranslateClient()
	translatedText, err := googleClient.Translate(fromWord, string(fromLanguage), string(toLanguage))
	if err != nil {
		return nil, err
	}

	targetWord, err := GetOrCreateWord(translatedText, toLanguage)
	if err != nil {
		return nil, err
	}

	translation := models.Translation{
		OriginalID:    sourceWord.ID,
		TranslationID: targetWord.ID,
		Source:        enums.TranslationSourceGoogle,
	}

	if err := db.DB.Create(&translation).Error; err != nil {
		return nil, err
	}

	translation.Original = sourceWord
	translation.Translation = targetWord

	return &TranslationResult{TranslationID: translation.ID, SourceWord: sourceWord.Word, TranslatedWord: targetWord.Word}, nil
}

func findExistingTranslation(sourceWordID uuid.UUID, targetLanguage enums.Language) (*models.Translation, error) {
	var translation models.Translation

	result := db.DB.
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
