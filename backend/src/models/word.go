package models

import (
	"termorize/src/enums"
	"time"

	"github.com/google/uuid"
)

type Word struct {
	ID        uuid.UUID      `json:"-" gorm:"default:gen_random_uuid()"`
	Word      string         `json:"word"`
	Language  enums.Language `json:"language"`
	CreatedAt time.Time      `json:"-"`
}
