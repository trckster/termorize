package services

import (
	"errors"
	"termorize/src/data/db"
	"termorize/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VocabularyShare struct {
	Translation *models.Translation `json:"translation"`
	SharePath   string              `json:"share_path"`
}

func ShareVocabulary(userID uint, vocabularyID uuid.UUID) (*VocabularyShare, error) {
	var item models.Vocabulary
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ? AND deleted_at IS NULL", vocabularyID, userID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrVocabularyNotFound
			}
			return err
		}
		if item.ShareToken == nil {
			token, err := GenerateInviteToken()
			if err != nil {
				return err
			}
			if err := tx.Model(&item).Update("share_token", token).Error; err != nil {
				return err
			}
			item.ShareToken = &token
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return GetPublicVocabularyShare(*item.ShareToken)
}

func StopSharingVocabulary(userID uint, vocabularyID uuid.UUID) error {
	result := db.DB.Model(&models.Vocabulary{}).Where("id = ? AND user_id = ? AND deleted_at IS NULL", vocabularyID, userID).Update("share_token", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrVocabularyNotFound
	}
	return nil
}

func GetPublicVocabularyShare(token string) (*VocabularyShare, error) {
	if len(token) != collectionInviteTokenLength {
		return nil, ErrVocabularyNotFound
	}
	var item models.Vocabulary
	err := db.DB.Where("share_token = ? AND deleted_at IS NULL", token).
		Where("user_id IN (?)", db.DB.Model(&models.User{}).Select("id").Where("deleted_at IS NULL")).
		Preload("Translation.Original").Preload("Translation.Translation").First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVocabularyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &VocabularyShare{Translation: item.Translation, SharePath: "/words/" + token}, nil
}
