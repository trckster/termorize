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
	t := getBotTextsForTelegramID(message.Chat.ID)

	switch command {
	case "ping":
		return SendMessage(message.Chat.ID, t.Pong)
	case "help", "start":
		return SendMessage(message.Chat.ID, t.Help)
	case "menu":
		return SendMessageWithInlineKeyboardMarkdown(message.Chat.ID, t.Menu, getMenuKeyboard(t))
	case "cancel":
		telegramID, _, _, _ := extractMessageUser(message)
		updated, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone)
		if err != nil {
			return err
		}

		if !updated {
			return SendMessage(message.Chat.ID, t.CancelNothing)
		}

		return SendMessage(message.Chat.ID, t.CancelDone)
	default:
		return SendMessage(message.Chat.ID, t.UnknownCommand)
	}
}
