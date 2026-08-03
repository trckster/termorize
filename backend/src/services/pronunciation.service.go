package services

import (
	"errors"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/integrations/openrouter"
	"termorize/src/models"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrWordNotFound = errors.New("word not found")
var pronunciationGenerationGroup singleflight.Group

func GetOrCreateWordPronunciation(wordID uuid.UUID) (*models.WordPronunciation, error) {
	var word models.Word
	if err := db.DB.Select("id", "word").Where("id = ?", wordID).First(&word).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWordNotFound
		}
		return nil, err
	}

	model := config.GetOpenRouterTTSModel()
	voice := config.GetOpenRouterTTSVoice()
	pronunciation, err := FindWordPronunciationMetadata(wordID, model, voice)
	if err != nil {
		return nil, err
	}

	if pronunciation != nil {
		pronunciation.Audio, pronunciation.MIMEType, err = GetWordPronunciationAudio(pronunciation.ID)
		if err != nil {
			return nil, err
		}

		return pronunciation, nil
	}

	key := wordID.String() + "\x00" + model + "\x00" + voice
	generated, err, _ := pronunciationGenerationGroup.Do(key, func() (any, error) {
		pronunciation, err := FindWordPronunciationMetadata(wordID, model, voice)
		if err != nil {
			return nil, err
		}

		if pronunciation != nil {
			pronunciation.Audio, pronunciation.MIMEType, err = GetWordPronunciationAudio(pronunciation.ID)
			return pronunciation, err
		}

		audio, err := openrouter.NewSpeechClient().GenerateSpeech(word.Word)
		if err != nil {
			return nil, err
		}

		return StoreWordPronunciation(wordID, model, voice, audio)
	})
	if err != nil {
		return nil, err
	}

	return generated.(*models.WordPronunciation), nil
}

func GetTranslationTargetWord(translationID uuid.UUID) (*models.Word, error) {
	var translation models.Translation
	if err := db.DB.
		Preload("Translation").
		Where("id = ?", translationID).
		First(&translation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTranslationNotFound
		}
		return nil, err
	}

	return translation.Translation, nil
}

// FindWordPronunciationMetadata deliberately excludes audio so Telegram file_id
// cache hits do not pull the canonical MP3 out of PostgreSQL.
func FindWordPronunciationMetadata(wordID uuid.UUID, model, voice string) (*models.WordPronunciation, error) {
	var pronunciation models.WordPronunciation
	result := db.DB.
		Select("id", "word_id", "model", "voice", "mime_type", "telegram_file_id", "created_at").
		Where("word_id = ? AND model = ? AND voice = ?", wordID, model, voice).
		First(&pronunciation)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &pronunciation, nil
}

func GetWordPronunciationAudio(id uuid.UUID) ([]byte, string, error) {
	var pronunciation models.WordPronunciation
	if err := db.DB.
		Select("audio", "mime_type").
		Where("id = ?", id).
		First(&pronunciation).Error; err != nil {
		return nil, "", err
	}

	return pronunciation.Audio, pronunciation.MIMEType, nil
}

func StoreWordPronunciation(wordID uuid.UUID, model, voice string, audio []byte) (*models.WordPronunciation, error) {
	pronunciation := models.WordPronunciation{
		WordID:   wordID,
		Model:    model,
		Voice:    voice,
		Audio:    audio,
		MIMEType: models.PronunciationMIMETypeMP3,
	}

	result := db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "word_id"}, {Name: "model"}, {Name: "voice"}},
		DoNothing: true,
	}).Create(&pronunciation)
	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		if err := db.DB.
			Where("word_id = ? AND model = ? AND voice = ?", wordID, model, voice).
			First(&pronunciation).Error; err != nil {
			return nil, err
		}
	}

	return &pronunciation, nil
}

func SetWordPronunciationTelegramFileID(id uuid.UUID, fileID string) error {
	return db.DB.Model(&models.WordPronunciation{}).
		Where("id = ?", id).
		Update("telegram_file_id", fileID).Error
}

func GetVocabularyTranslationID(userID uint, vocabularyID uuid.UUID) (uuid.UUID, error) {
	var vocabulary models.Vocabulary
	if err := db.DB.
		Select("translation_id").
		Where("id = ? AND user_id = ?", vocabularyID, userID).
		First(&vocabulary).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, ErrVocabularyNotFound
		}
		return uuid.Nil, err
	}

	return vocabulary.TranslationID, nil
}
