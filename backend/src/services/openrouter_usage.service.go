package services

import (
	"errors"
	"fmt"
	"termorize/src/data/db"
	"termorize/src/integrations/openrouter"
	"termorize/src/logger"
	"termorize/src/models"
	"time"
)

const (
	openRouterSpendingWindow = 24 * time.Hour
	openRouterUserLimit      = 1.0
	openRouterAdminLimit     = 10.0
)

var ErrOpenRouterSpendingLimit = errors.New("OpenRouter spending limit reached")

type OpenRouterSpendingLimitError struct {
	Limit   float64
	RetryAt time.Time
}

func (e *OpenRouterSpendingLimitError) Error() string {
	return fmt.Sprintf("%s; try again after %s", ErrOpenRouterSpendingLimit, e.RetryAt.UTC().Format(time.RFC3339))
}

func (e *OpenRouterSpendingLimitError) Unwrap() error {
	return ErrOpenRouterSpendingLimit
}

func AsOpenRouterSpendingLimitError(err error) (*OpenRouterSpendingLimitError, bool) {
	var limitErr *OpenRouterSpendingLimitError
	ok := errors.As(err, &limitErr)
	return limitErr, ok
}

func CheckOpenRouterSpendingLimit(userID uint) error {
	var user models.User
	if err := db.DB.Select("is_admin").First(&user, userID).Error; err != nil {
		return err
	}

	limit := openRouterUserLimit
	if user.IsAdmin {
		limit = openRouterAdminLimit
	}

	now := time.Now().UTC()
	var usages []models.OpenRouterUsage
	if err := db.DB.
		Select("cost", "created_at").
		Where("user_id = ? AND created_at > ?", userID, now.Add(-openRouterSpendingWindow)).
		Order("created_at ASC, id ASC").
		Find(&usages).Error; err != nil {
		return err
	}

	total := 0.0
	for _, usage := range usages {
		total += usage.Cost
	}
	if total < limit {
		return nil
	}

	retryAt := now.Add(openRouterSpendingWindow)
	remaining := total
	for _, usage := range usages {
		remaining -= usage.Cost
		if remaining < limit {
			retryAt = usage.CreatedAt.Add(openRouterSpendingWindow)
			break
		}
	}

	return &OpenRouterSpendingLimitError{Limit: limit, RetryAt: retryAt}
}

func RecordOpenRouterUsage(userID uint, usage openrouter.Usage) {
	// Responses without accounting metadata did not reach billable inference.
	if usage.GenerationID == "" && usage.Cost == 0 && usage.TotalTokens == 0 {
		return
	}

	var generationID *string
	if usage.GenerationID != "" {
		generationID = &usage.GenerationID
	}
	record := models.OpenRouterUsage{
		UserID:           userID,
		GenerationID:     generationID,
		Model:            usage.Model,
		Cost:             usage.Cost,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
	if err := db.DB.Create(&record).Error; err != nil {
		logger.L().Errorw("failed to persist openrouter usage", "error", err, "user_id", userID, "generation_id", usage.GenerationID)
	}
}
