package models

import (
	"database/sql/driver"
	"encoding/json"
	"termorize/src/enums"
	"time"

	"github.com/google/uuid"
)

type ProgressEntry struct {
	Knowledge int                 `json:"knowledge"`
	Type      enums.KnowledgeType `json:"type"`
}

type ProgressEntries []ProgressEntry

func (p ProgressEntries) Value() (driver.Value, error) {
	if p == nil {
		return []byte("[]"), nil
	}

	return json.Marshal(p)
}

func (p *ProgressEntries) Scan(value any) error {
	return json.Unmarshal(value.([]byte), p)
}

type Vocabulary struct {
	ID            uuid.UUID       `json:"id" gorm:"default:gen_random_uuid()"`
	UserID        uint            `json:"-"`
	TranslationID uuid.UUID       `json:"-"`
	Progress      ProgressEntries `json:"progress" gorm:"default:'[]'"`
	CreatedAt     time.Time       `json:"created_at"`
	MasteredAt    *time.Time      `json:"mastered_at"`
	Translation   *Translation    `json:"translation"`
	User          *User           `json:"-"`
}

func (v *Vocabulary) TableName() string {
	return "vocabulary"
}
