package telegram

import (
	"strings"
	"termorize/src/logger"
	"termorize/src/services"
)

type messageCommandHandler func(message *message, args string) error

var messageCommandHandlers = map[string]messageCommandHandler{
	"ping": func(message *message, args string) error {
		return SendMessage(message.Chat.ID, "pong")
	},
}

func handleMessage(message *message) error {
	if message.Chat.Type == Private {
		if err := ensurePrivateMessageUser(message); err != nil {
			return err
		}
	}

	if message.Text == "" {
		return nil
	}

	if message.Chat.Type != Private {
		return SendMessage(message.Chat.ID, "Nah... Don't feel like answering here rn")
	}

	if command, args, ok := parseMessageCommand(message.Text); ok {
		if err := routeMessageCommand(message, command, args); err != nil {
			return err
		}
		return nil
	}

	return SendMessage(message.Chat.ID, message.Text)
}

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

func parseMessageCommand(text string) (string, string, bool) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") {
		return "", "", false
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", "", false
	}

	command := strings.TrimPrefix(parts[0], "/")
	if command == "" {
		return "", "", false
	}

	if index := strings.Index(command, "@"); index >= 0 {
		command = command[:index]
	}

	if command == "" {
		return "", "", false
	}

	arguments := ""
	if len(parts) > 1 {
		arguments = strings.Join(parts[1:], " ")
	}

	return strings.ToLower(command), arguments, true
}

func routeMessageCommand(message *message, command string, args string) error {
	handler, exists := messageCommandHandlers[command]
	if !exists {
		return nil
	}

	return handler(message, args)
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
