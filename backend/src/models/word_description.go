package models

import (
	"time"

	"github.com/google/uuid"
)

type WordDescription struct {
	ID                uuid.UUID `gorm:"default:gen_random_uuid()"`
	WordID            uuid.UUID
	TranslationWordID *uuid.UUID `gorm:"not null"`
	Model             string
	Description       string
	CreatedAt         time.Time
	Word              *Word
	TranslationWord   *Word
}
