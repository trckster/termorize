package models

import (
	"time"

	"github.com/google/uuid"
)

type TranslationDescription struct {
	ID            uuid.UUID `gorm:"default:gen_random_uuid()"`
	TranslationID uuid.UUID
	Model         string
	Description   string
	CreatedAt     time.Time
	Translation   *Translation
}
