package telegram

import (
	"encoding/base64"
	"errors"
	"strings"
	"termorize/src/logger"
	"termorize/src/services"

	"github.com/google/uuid"
)

func handleCallbackQuery(callback *callbackQuery) error {
	if callback == nil || callback.From == nil {
		return nil
	}

	deleted, err := services.IsUserDeletedByTelegramID(callback.From.ID)
	if err != nil {
		return err
	}
	if deleted {
		return nil
	}

	if callback.ID != "" && !shouldDeferCallbackAnswer(callback) {
		if err := answerTelegramCallbackQuery(callback.ID); err != nil {
			logger.L().Warnw("failed to answer callback query", "error", err, "callback_id", callback.ID)
		}
	}

	return routeCallbackData(callback)
}

func shouldDeferCallbackAnswer(callback *callbackQuery) bool {
	if callback.From == nil || callback.Message == nil {
		return false
	}

	handlerType, payload, ok := parseCallbackData(callback.Data)
	return ok && handlerType == callbackTypeExercise && len(payload) > 0 && payload[0] == exerciseActionMatchTap
}

func parseCallbackData(data string) (string, []string, bool) {
	parts := strings.Split(data, ":")
	if len(parts) < 2 || parts[0] == "" {
		return "", nil, false
	}

	return parts[0], parts[1:], true
}

func routeCallbackData(callback *callbackQuery) error {
	handlerType, payload, ok := parseCallbackData(callback.Data)
	if !ok {
		return nil
	}

	switch handlerType {
	case callbackTypeExercise:
		return handleExerciseCallback(callback, payload)
	case callbackTypeMenu:
		return handleMenuCallback(callback, payload)
	case callbackTypeVocabulary:
		return handleVocabularyCallback(callback, payload)
	case callbackTypePronunciation:
		handlePronunciationCallback(callback, payload)
		return nil
	default:
		return nil
	}
}

func parseCallbackUUID(value string) (uuid.UUID, error) {
	if id, err := uuid.Parse(value); err == nil {
		return id, nil
	}

	bytes, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return uuid.Nil, err
	}

	return uuid.FromBytes(bytes)
}

func logCallbackAnswerError(err error, callbackID string) {
	if err == nil {
		return
	}
	if isExpiredCallbackQueryError(err) {
		logger.L().Debugw("telegram callback query expired", "callback_id", callbackID)
		return
	}

	logger.L().Warnw("failed to answer telegram callback query", "error", err, "callback_id", callbackID)
}

func isExpiredCallbackQueryError(err error) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) &&
		apiErr.ErrorCode == 400 &&
		strings.Contains(apiErr.Description, "query is too old")
}
