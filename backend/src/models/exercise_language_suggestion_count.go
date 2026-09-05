package models

import (
	"termorize/src/enums"
	"time"
)

type ExerciseLanguageSuggestionCount struct {
	UserID     uint           `gorm:"primaryKey"`
	Family     string         `gorm:"primaryKey"`
	Language   enums.Language `gorm:"primaryKey"`
	ShownCount int16
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
