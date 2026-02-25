package models

import (
	"github.com/google/uuid"
	"termorize/src/enums"
	"time"
)

type Exercise struct {
	ID     uuid.UUID            `json:"id" gorm:"default:gen_random_uuid()"`
	Type   enums.ExerciseType   `json:"type"`
	Status enums.ExerciseStatus `json:"status"`
	UserID uint                 `json:"-"`

	ScheduledFor *time.Time `json:"-"`
	StartedAt    *time.Time `json:"starts_at"`
	FinishedAt   *time.Time `json:"finishes_at"`
	CreatedAt    time.Time  `json:"-"`
	UpdatedAt    time.Time  `json:"-"`

	User *User `json:"-"`
}
