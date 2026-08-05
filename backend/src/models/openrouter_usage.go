package models

import (
	"time"

	"github.com/google/uuid"
)

// OpenRouterUsage is the immutable billing record for one completed OpenRouter request.
type OpenRouterUsage struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid()"`
	UserID           uint      `json:"user_id"`
	GenerationID     *string   `json:"generation_id,omitempty"`
	Model            string    `json:"model"`
	Cost             float64   `json:"cost" gorm:"type:numeric(20,10)"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	CreatedAt        time.Time `json:"created_at"`
}

func (OpenRouterUsage) TableName() string {
	return "openrouter_usages"
}
