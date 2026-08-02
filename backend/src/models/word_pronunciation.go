package models

import (
	"time"

	"github.com/google/uuid"
)

const PronunciationMIMETypeMP3 = "audio/mpeg"

type WordPronunciation struct {
	ID             uuid.UUID `gorm:"default:gen_random_uuid()"`
	WordID         uuid.UUID
	Model          string
	Voice          string
	Audio          []byte
	MIMEType       string
	TelegramFileID *string
	CreatedAt      time.Time
	Word           *Word
}
