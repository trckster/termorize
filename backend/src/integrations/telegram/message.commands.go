package telegram

import (
	"strings"
	"termorize/src/enums"
	"termorize/src/services"
)

func parseMessageCommand(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") {
		return "", false
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", false
	}

	command := strings.TrimPrefix(parts[0], "/")
	if command == "" {
		return "", false
	}

	if index := strings.Index(command, "@"); index >= 0 {
		command = command[:index]
	}

	if command == "" {
		return "", false
	}

	return strings.ToLower(command), true
}

func routeMessageCommand(message *message, command string) error {
	switch command {
	case "ping":
		return SendMessage(message.Chat.ID, telegramTextPong)
	case "help", "start":
		return SendMessage(message.Chat.ID, telegramTextHelp)
	case "menu":
		return SendMessageWithInlineKeyboardMarkdown(message.Chat.ID, telegramTextMenu, menuKeyboard)
	case "cancel":
		telegramID, _, _, _ := extractMessageUser(message)
		updated, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone)
		if err != nil {
			return err
		}

		if !updated {
			return SendMessage(message.Chat.ID, telegramTextCancelNothing)
		}

		return SendMessage(message.Chat.ID, telegramTextCancelDone)
	default:
		return SendMessage(message.Chat.ID, telegramTextUnknownCommand)
	}
}
