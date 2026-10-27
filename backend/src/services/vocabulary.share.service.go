package services

import (
	"errors"
	"strings"
	"termorize/src/data/db"
	"termorize/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrInvalidVocabularySelection = errors.New("select between 1 and 1000 available vocabulary items")

type ShareVocabularyRequest struct {
	Title         string      `json:"title" binding:"required,max=200"`
	VocabularyIDs []uuid.UUID `json:"vocabulary_ids" binding:"required,min=1,max=1000"`
}

// ShareVocabulary snapshots selected translations into an ordinary personal collection.
// All selection checks and writes are atomic, so invalid selections never publish a partial share.
func ShareVocabulary(userID uint, req ShareVocabularyRequest) (*CollectionDetail, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrCollectionTitleRequired
	}
	if len(req.VocabularyIDs) == 0 || len(req.VocabularyIDs) > 1000 {
		return nil, ErrInvalidVocabularySelection
	}
	ids := make([]uuid.UUID, 0, len(req.VocabularyIDs))
	seen := make(map[uuid.UUID]bool)
	for _, id := range req.VocabularyIDs {
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	token, err := GenerateInviteToken()
	if err != nil {
		return nil, err
	}
	owner := userID
	collection := models.Collection{Title: title, OwnerID: &owner, IsPublished: true, InviteToken: token}
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var vocabulary []models.Vocabulary
		if err := tx.Where("user_id = ? AND deleted_at IS NULL AND id IN ?", userID, ids).Find(&vocabulary).Error; err != nil {
			return err
		}
		if len(vocabulary) != len(ids) {
			return ErrInvalidVocabularySelection
		}
		translations := make(map[uuid.UUID]uuid.UUID, len(vocabulary))
		for _, item := range vocabulary {
			translations[item.ID] = item.TranslationID
		}
		if err := tx.Create(&collection).Error; err != nil {
			return err
		}
		links := make([]models.CollectionTranslation, 0, len(ids))
		linked := make(map[uuid.UUID]bool)
		for _, id := range ids {
			translationID := translations[id]
			if linked[translationID] {
				continue
			}
			links = append(links, models.CollectionTranslation{CollectionID: collection.ID, TranslationID: translationID, Position: len(links)})
			linked[translationID] = true
		}
		return tx.Create(&links).Error
	})
	if err != nil {
		return nil, err
	}
	return GetCollection(userID, collection.ID)
}
