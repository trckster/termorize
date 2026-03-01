package telegram

import (
	"termorize/src/logger"
	"termorize/src/services"
)

func ensurePrivateMessageUser(message *message) error {
	telegramID, username, firstName, lastName := extractMessageUser(message)

	if err := services.EnsureUserByTelegramID(telegramID, username, firstName, lastName); err != nil {
		logger.L().Warnw("failed to ensure telegram user", "error", err, "telegram_id", telegramID)
		return err
	}

	if err := services.UpdateUserTelegramBotEnabled(telegramID, true); err != nil {
		logger.L().Warnw("failed to enable telegram bot for user", "error", err, "telegram_id", telegramID)
		return err
	}

	return nil
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
