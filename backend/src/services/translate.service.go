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

type TranslateRequest struct {
	FromWord     string         `json:"from_word" binding:"required"`
	FromLanguage enums.Language `json:"from_language" binding:"required,enum=Language"`
	ToLanguage   enums.Language `json:"to_language" binding:"required,enum=Language"` // TODO move to controller + nefield + fix check of existing
}

type TranslateResponse struct {
	Translation string `json:"translation"`
}

func Translate(req TranslateRequest) (*TranslateResponse, error) {
	if req.FromLanguage == req.ToLanguage {
		return nil, errors.New("languages must differ")
	}

	sourceWord, err := GetOrCreateWord(req.FromWord, req.FromLanguage)
	if err != nil {
		return nil, err
	}

	existingTranslation, err := findExistingTranslation(sourceWord.ID, req.ToLanguage)
	if err != nil {
		return nil, err
	}

	if existingTranslation != nil {
		return &TranslateResponse{Translation: existingTranslation.Word}, nil
	}

	googleClient := google.NewTranslateClient()
	translatedText, err := googleClient.Translate(req.FromWord, string(req.FromLanguage), string(req.ToLanguage))
	if err != nil {
		return nil, err
	}

	targetWord, err := GetOrCreateWord(translatedText, req.ToLanguage)
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

	return &TranslateResponse{Translation: targetWord.Word}, nil
}

func findExistingTranslation(sourceWordID uuid.UUID, targetLanguage enums.Language) (*models.Word, error) {
	var translation models.Translation

	result := db.DB.
		Joins("JOIN words AS original_word ON original_word.id = translations.original_id").
		Joins("JOIN words AS translation_word ON translation_word.id = translations.translation_id").
		Preload("Original").
		Preload("Translation").
		Where("translations.source != ?", enums.TranslationSourceUser).
		Where("(translations.original_id = ? AND translation_word.language = ?) OR (translations.translation_id = ? AND original_word.language = ?)",
			sourceWordID, string(targetLanguage), sourceWordID, string(targetLanguage)).
		First(&translation)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	if translation.OriginalID == sourceWordID {
		return translation.Translation, nil
	}
	return translation.Original, nil
}
