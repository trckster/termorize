package models

import (
	"termorize/src/enums"
	"time"

	"github.com/google/uuid"
)

type DailyIdiom struct {
	ID       uuid.UUID `gorm:"default:gen_random_uuid()"`
	Date     time.Time `gorm:"type:date"`
	Language enums.Language
	WordID   uuid.UUID
}
