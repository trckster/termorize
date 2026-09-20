package models

import (
	"termorize/src/enums"
	"time"

	"github.com/google/uuid"
)

type Word struct {
	ID        uuid.UUID      `json:"id" gorm:"default:gen_random_uuid()"`
	Word      string         `json:"word"`
	Language  enums.Language `json:"language"`
	Type      enums.Type     `json:"type" gorm:"default:unknown"`
	CreatedAt time.Time      `json:"-"`
}
