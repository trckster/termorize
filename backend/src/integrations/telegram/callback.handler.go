package telegram

import (
	"strings"
	"termorize/src/logger"
	"termorize/src/services"

	"github.com/google/uuid"
)

type callbackDataHandler func(callback *callbackQuery, payload []string) error

var callbackDataHandlers = map[string]callbackDataHandler{
	"exercise": handleExerciseCallback,
	"menu":     handleMenuCallback,
}

var menuBackKeyboard = [][]inlineKeyboardButton{{{Text: "Back", CallbackData: "menu:back"}}}

func handleCallbackQuery(callback *callbackQuery) error {
	if callback == nil {
		return nil
	}

	if callback.ID != "" {
		if err := answerTelegramCallbackQuery(callback.ID); err != nil {
			logger.L().Warnw("failed to answer callback query", "error", err, "callback_id", callback.ID)
		}
	}

	if callback.From == nil {
		return nil
	}

	return routeCallbackData(callback)
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

	handler, exists := callbackDataHandlers[handlerType]
	if !exists {
		return nil
	}

	return handler(callback, payload)
}

func parseExerciseCallbackPayload(payload []string) (string, uuid.UUID, string, bool) {
	if len(payload) != 3 {
		return "", uuid.Nil, "", false
	}

	exerciseID, err := uuid.Parse(payload[1])
	if err != nil {
		return "", uuid.Nil, "", false
	}

	if payload[2] != "o2t" && payload[2] != "t2o" {
		return "", uuid.Nil, "", false
	}

	return payload[0], exerciseID, payload[2], true
}

func handleExerciseCallback(callback *callbackQuery, payload []string) error {
	action, exerciseID, questionType, ok := parseExerciseCallbackPayload(payload)
	if !ok || action != "idk" {
		return nil
	}

	if callback.Message != nil {
		if err := removeMessageInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID); err != nil {
			logger.L().Warnw("failed to remove inline keyboard", "error", err, "chat_id", callback.Message.Chat.ID, "message_id", callback.Message.MessageID)
		}
	}

	updated, err := services.FailExercise(exerciseID)
	if err != nil {
		return err
	}

	if !updated {
		return nil
	}

	words, err := services.GetExerciseWordsByTelegram(exerciseID, callback.From.ID)
	if err != nil {
		return err
	}

	if words == nil {
		return nil
	}

	answerText := buildIDKAnswer(words.OriginalWord, words.TranslationWord, questionType)
	return SendMessage(callback.From.ID, answerText)
}

func handleMenuCallback(callback *callbackQuery, payload []string) error {
	if callback.Message == nil {
		return nil
	}

	if len(payload) != 1 {
		return nil
	}

	action := payload[0]
	if action == "back" {
		return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, menuMessageText, menuKeyboard)
	}

	selectionText, ok := menuActionToText(action)
	if !ok {
		return nil
	}

	return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, selectionText, menuBackKeyboard)
}

func menuActionToText(action string) (string, bool) {
	switch action {
	case "add_translation":
		return "Work in progress here!", true
	case "delete_translation":
		return "Work in progress here!", true
	case "your_vocabulary":
		return "Work in progress here!", true
	case "statistics":
		return "Work in progress here!", true
	case "settings":
		return "Work in progress here!", true
	default:
		return "", false
	}
}

func buildIDKAnswer(originalWord string, translationWord string, questionType string) string {
	if questionType == "t2o" {
		return "Correct original word: " + originalWord
	}

	return "Correct translation: " + translationWord
}
