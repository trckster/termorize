package models

import (
	"termorize/src/enums"
	"time"

	"github.com/google/uuid"
)

type Exercise struct {
	ID                uuid.UUID            `json:"id" gorm:"default:gen_random_uuid()"`
	Type              enums.ExerciseType   `json:"type"`
	Status            enums.ExerciseStatus `json:"status"`
	UserID            uint                 `json:"-"`
	TelegramMessageID *int64               `json:"-"`
	MatchState        *string              `json:"-" gorm:"column:match_state"`
	CharacterState    *string              `json:"-" gorm:"column:character_state"`

	ScheduledFor   *time.Time `json:"-"`
	StartedAt      *time.Time `json:"starts_at"`
	ReminderSentAt *time.Time `json:"-"`
	FinishedAt     *time.Time `json:"finishes_at"`
	CreatedAt      time.Time  `json:"-"`
	UpdatedAt      time.Time  `json:"-"`

	User       *User        `json:"-"`
	Vocabulary []Vocabulary `json:"vocabularies,omitempty" gorm:"many2many:vocabulary_exercises;"`
}

type ExerciseVocabulary struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ExerciseID     uuid.UUID  `json:"exercise_id"`
	VocabularyID   uuid.UUID  `json:"vocabulary_id"`
	IsCorrect      bool       `json:"is_correct"`
	Position       int        `json:"position"`
	Result         *string    `json:"result"`
	ResultReason   *string    `json:"result_reason"`
	ProgressDelta  *int       `json:"progress_delta"`
	KnowledgeAfter *int       `json:"knowledge_after"`
	AnsweredAt     *time.Time `json:"answered_at"`

	Exercise   *Exercise   `json:"-" gorm:"foreignKey:ExerciseID"`
	Vocabulary *Vocabulary `json:"-" gorm:"foreignKey:VocabularyID"`
}

func (ExerciseVocabulary) TableName() string {
	return "vocabulary_exercises"
}
