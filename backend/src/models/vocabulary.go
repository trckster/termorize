package models

import (
	"termorize/src/enums"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ProgressEntry struct {
	Knowledge int                 `json:"knowledge"`
	Type      enums.KnowledgeType `json:"type"`
}

type Vocabulary struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID        uint           `json:"user_id" gorm:"not null;index"`
	TranslationID uuid.UUID      `json:"translation_id" gorm:"type:uuid;not null;uniqueIndex:idx_user_translation"`
	Progress      datatypes.JSON `json:"progress" gorm:"type:jsonb;default:'[]'"`
	CreatedAt     time.Time      `json:"created_at"`
	MasteredAt    *time.Time     `json:"mastered_at"`
	Translation   *Translation   `json:"-" gorm:"foreignKey:TranslationID"`
	User          *User          `json:"-" gorm:"foreignKey:UserID"`
}

func (v *Vocabulary) TableName() string {
	return "vocabulary"
}
