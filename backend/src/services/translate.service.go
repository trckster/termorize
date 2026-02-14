package services

import (
	"errors"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/google"
	"termorize/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Translate(fromWord string, fromLanguage enums.Language, toLanguage enums.Language) (string, error) {
	sourceWord, err := GetOrCreateWord(fromWord, fromLanguage)
	if err != nil {
		return "", err
	}

	existingTranslation, err := findExistingTranslation(sourceWord.ID, toLanguage)
	if err != nil {
		return "", err
	}

	if existingTranslation != nil {
		return existingTranslation.Word, nil
	}

	googleClient := google.NewTranslateClient()
	translatedText, err := googleClient.Translate(fromWord, string(fromLanguage), string(toLanguage))
	if err != nil {
		return "", err
	}

	targetWord, err := GetOrCreateWord(translatedText, toLanguage)
	if err != nil {
		return "", err
	}

	translation := models.Translation{
		OriginalID:    sourceWord.ID,
		TranslationID: targetWord.ID,
		Source:        enums.TranslationSourceGoogle,
	}

	if err := db.DB.Create(&translation).Error; err != nil {
		return "", err
	}

	return targetWord.Word, nil
}

func findExistingTranslation(sourceWordID uuid.UUID, targetLanguage enums.Language) (*models.Word, error) {
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

	return translation.Translation, nil
}
