package models

import (
	"termorize/src/enums"
	"time"

	"github.com/google/uuid"
)

type Word struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Word      string         `json:"word" gorm:"not null;uniqueIndex:idx_word_language"`
	Language  enums.Language `json:"language" gorm:"type:varchar(10);not null;uniqueIndex:idx_word_language"`
	CreatedAt time.Time      `json:"created_at"`
}
