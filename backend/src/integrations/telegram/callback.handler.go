package telegram

import (
	"strings"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/services"

	"github.com/google/uuid"
)

type callbackDataHandler func(callback *callbackQuery, payload []string) error

var callbackDataHandlers = map[string]callbackDataHandler{
	"exercise": handleExerciseCallback,
	"menu":     handleMenuCallback,
}

var menuBackKeyboard = [][]inlineKeyboardButton{{{Text: telegramButtonMenuBack, CallbackData: "menu:back"}}}
var menuCancelKeyboard = [][]inlineKeyboardButton{{{Text: telegramButtonMenuCancel, CallbackData: "menu:cancel"}}}

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

func parseExerciseCallbackPayload(payload []string) (string, uuid.UUID, bool) {
	if len(payload) != 2 {
		return "", uuid.Nil, false
	}

	exerciseID, err := uuid.Parse(payload[1])
	if err != nil {
		return "", uuid.Nil, false
	}

	return payload[0], exerciseID, true
}

func handleExerciseCallback(callback *callbackQuery, payload []string) error {
	action, exerciseID, ok := parseExerciseCallbackPayload(payload)
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

	answerText := buildIDKAnswer(words.OriginalWord, words.TranslationWord, words.ExerciseType)
	return SendMessageMarkdown(callback.From.ID, answerText)
}

func handleMenuCallback(callback *callbackQuery, payload []string) error {
	if callback.Message == nil {
		return nil
	}

	if len(payload) != 1 {
		return nil
	}

	action := payload[0]
	if action == "back" || action == "cancel" {
		if _, err := services.UpdateUserTelegramState(callback.From.ID, enums.TelegramStateNone); err != nil {
			return err
		}

		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, telegramTextMenu, menuKeyboard)
	}

	if action == "delete_translation" {
		if _, err := services.UpdateUserTelegramState(callback.From.ID, enums.TelegramStateDeletingVocabulary); err != nil {
			return err
		}

		return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, telegramTextMenuDeleteWord, menuCancelKeyboard)
	}

	if action == "add_translation" {
		if _, err := services.UpdateUserTelegramState(callback.From.ID, enums.TelegramStateAddingVocabulary); err != nil {
			return err
		}

		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		messageText := buildAddVocabularyFirstText(user.Settings.NativeLanguage.DisplayNameWithFlag(), user.Settings.MainLearningLanguage.DisplayNameWithFlag())
		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, menuCancelKeyboard)
	}

	selectionText, ok := menuActionToText(action)
	if !ok {
		return nil
	}

	return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, selectionText, menuBackKeyboard)
}

func menuActionToText(action string) (string, bool) {
	switch action {
	case "your_vocabulary":
		return telegramTextMenuVocabulary, true
	case "statistics":
		return telegramTextMenuStatistics, true
	case "settings":
		return telegramTextMenuSettings, true
	default:
		return "", false
	}
}

func buildIDKAnswer(originalWord string, translationWord string, exerciseType enums.ExerciseType) string {
	if exerciseType == enums.ExerciseTypeBasicReversed {
		return telegramTextIDKOriginalPrefix + "*" + originalWord + "*"
	}

	return telegramTextIDKTranslationPrefix + "*" + translationWord + "*"
}
