package telegram

import (
	"termorize/src/logger"
	"termorize/src/services"
)

func ensurePrivateMessageUser(message *message) (bool, error) {
	telegramID, username, firstName, lastName := extractMessageUser(message)

	active, err := services.EnsureUserByTelegramID(telegramID, username, firstName, lastName)
	if err != nil {
		logger.L().Warnw("failed to ensure telegram user", "error", err, "telegram_id", telegramID)
		return false, err
	}
	if !active {
		return false, nil
	}

	if err := services.UpdateUserTelegramBotEnabled(telegramID, true); err != nil {
		logger.L().Warnw("failed to enable telegram bot for user", "error", err, "telegram_id", telegramID)
		return false, err
	}

	return true, nil
}

func extractMessageUser(message *message) (int64, string, string, string) {
	telegramID := message.Chat.ID
	username := message.Chat.Username
	firstName := message.Chat.FirstName
	lastName := ""

	if message.From != nil {
		telegramID = message.From.ID
		username = message.From.Username
		firstName = message.From.FirstName
		lastName = message.From.LastName
	}

	return telegramID, username, firstName, lastName
}
