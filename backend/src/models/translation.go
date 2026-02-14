package models

import (
	"errors"
	"termorize/src/enums"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Translation struct {
	ID            uuid.UUID               `json:"id" gorm:"default:gen_random_uuid()"`
	OriginalID    uuid.UUID               `json:"-"`
	TranslationID uuid.UUID               `json:"-"`
	Source        enums.TranslationSource `json:"source"`
	UserID        *uint                   `json:"-"`
	CreatedAt     time.Time               `json:"-"`
	Original      *Word                   `json:"original"`
	Translation   *Word                   `json:"translation"`
	User          *User                   `json:"-"`
}

func (t *Translation) BeforeSave(_ *gorm.DB) error {
	if t.OriginalID == t.TranslationID {
		return errors.New("original and translation IDs can't be the same")
	}
	return nil
}

func (t *Translation) BeforeCreate(tx *gorm.DB) error {
	var count int64
	tx.Model(&Translation{}).
		Where("user_id = ? AND ((original_id = ? AND translation_id = ?) OR (original_id = ? AND translation_id = ?))",
			t.UserID, t.OriginalID, t.TranslationID, t.TranslationID, t.OriginalID).
		Count(&count)

	if count > 0 {
		return errors.New("translation already exists")
	}
	return nil
}
