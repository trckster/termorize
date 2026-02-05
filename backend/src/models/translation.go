package models

import (
	"errors"
	"termorize/src/enums"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Translation struct {
	ID        uuid.UUID               `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Word1ID   uuid.UUID               `json:"word_1_id" gorm:"column:word_1_id;type:uuid;not null;index"`
	Word2ID   uuid.UUID               `json:"word_2_id" gorm:"column:word_2_id;type:uuid;not null;index"`
	Source    enums.TranslationSource `json:"source" gorm:"type:varchar(20);not null"`
	UserID    *uint                   `json:"user_id" gorm:"index"`
	CreatedAt time.Time               `json:"created_at"`
	Word1     *Word                   `json:"-" gorm:"foreignKey:Word1ID"`
	Word2     *Word                   `json:"-" gorm:"foreignKey:Word2ID"`
	User      *User                   `json:"-" gorm:"foreignKey:UserID"`
}

func (t *Translation) BeforeCreate(tx *gorm.DB) error {
	var count int64
	tx.Model(&Translation{}).
		Where("user_id = ? AND ((word_1_id = ? AND word_2_id = ?) OR (word_1_id = ? AND word_2_id = ?))",
			t.UserID, t.Word1ID, t.Word2ID, t.Word2ID, t.Word1ID).
		Count(&count)

	if count > 0 {
		return errors.New("translation already exists")
	}
	return nil
}
