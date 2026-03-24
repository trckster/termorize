package telegram

import (
	"strings"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/services"

	"github.com/google/uuid"
)

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

	switch handlerType {
	case callbackTypeExercise:
		return handleExerciseCallback(callback, payload)
	case callbackTypeMenu:
		return handleMenuCallback(callback, payload)
	case callbackTypeVocabulary:
		return handleVocabularyCallback(callback, payload)
	default:
		return nil
	}
}

func parseExerciseIDKPayload(payload []string) (uuid.UUID, bool) {
	if len(payload) != 2 || payload[0] != exerciseActionIDK {
		return uuid.Nil, false
	}

	exerciseID, err := uuid.Parse(payload[1])
	if err != nil {
		return uuid.Nil, false
	}

	return exerciseID, true
}

func handleExerciseCallback(callback *callbackQuery, payload []string) error {
	exerciseID, ok := parseExerciseIDKPayload(payload)
	if !ok {
		return nil
	}

	if callback.Message == nil {
		return nil
	}

	t := getBotTextsForTelegramID(callback.From.ID)

	exercise, err := services.GetExerciseByTelegramMessage(callback.Message.MessageID, callback.From.ID)
	if err != nil {
		return err
	}

	if exercise == nil || exercise.ExerciseID != exerciseID {
		return nil
	}

	switch exercise.Status {
	case enums.ExerciseStatusIgnored:
		return SendMessage(callback.From.ID, t.ExerciseOutdated)
	case enums.ExerciseStatusCompleted:
		return SendMessage(callback.From.ID, t.ExerciseCompleted)
	case enums.ExerciseStatusFailed:
		return SendMessage(callback.From.ID, t.ExerciseFailed)
	}

	if err := removeMessageInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID); err != nil {
		logger.L().Warnw("failed to remove inline keyboard", "error", err, "chat_id", callback.Message.Chat.ID, "message_id", callback.Message.MessageID)
	}

	updated, translationKnowledge, err := services.FailExercise(exerciseID)
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

	answerText := buildExerciseIDKResultText(
		words.OriginalWord,
		words.TranslationWord,
		words.OriginalLanguage,
		words.TranslationLanguage,
		translationKnowledge,
		t,
	)
	return SendMessageMarkdown(callback.From.ID, answerText)
}

func handleMenuCallback(callback *callbackQuery, payload []string) error {
	if callback.Message == nil {
		return nil
	}

	if len(payload) == 0 {
		return nil
	}

	t := getBotTextsForTelegramID(callback.From.ID)
	action := payload[0]

	if action == menuActionBack || action == menuActionCancel {
		if _, err := services.UpdateUserTelegramState(callback.From.ID, enums.TelegramStateNone); err != nil {
			return err
		}

		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, t.Menu, getMenuKeyboard(t))
	}

	if action == menuActionDeleteTranslation {
		if _, err := services.UpdateUserTelegramState(callback.From.ID, enums.TelegramStateDeletingVocabulary); err != nil {
			return err
		}

		return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, t.MenuDeleteWord, getMenuCancelKeyboard(t))
	}

	if action == menuActionAddTranslation {
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

		messageText := buildAddVocabularyFirstText(
			user.Settings.TranslationSourceLanguage.DisplayNameWithFlag(),
			user.Settings.TranslationTargetLanguage.DisplayNameWithFlag(),
			t,
		)
		keyboard := buildAddTranslationKeyboard(user.Settings.TranslationSourceLanguage, user.Settings.TranslationTargetLanguage, t)
		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, keyboard)
	}

	if action == menuActionChangeSourceLang || action == menuActionChangeTargetLang {
		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		isSource := action == menuActionChangeSourceLang
		keyboard := buildLanguageSelectionKeyboard(user.Settings.TranslationSourceLanguage, user.Settings.TranslationTargetLanguage, isSource, t)
		return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, t.ChooseLanguage, keyboard)
	}

	if action == menuActionChangeSystemLang {
		keyboard := buildSystemLanguageSelectionKeyboard(t)
		return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, t.ChooseLanguage, keyboard)
	}

	if action == menuActionSetSourceLang || action == menuActionSetTargetLang {
		if len(payload) != 2 {
			return nil
		}

		langCode := enums.Language(payload[1])
		isSource := action == menuActionSetSourceLang

		user, err := services.UpdateUserTranslationLanguage(callback.From.ID, isSource, langCode)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		messageText := buildAddVocabularyFirstText(
			user.Settings.TranslationSourceLanguage.DisplayNameWithFlag(),
			user.Settings.TranslationTargetLanguage.DisplayNameWithFlag(),
			t,
		)
		keyboard := buildAddTranslationKeyboard(user.Settings.TranslationSourceLanguage, user.Settings.TranslationTargetLanguage, t)
		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, keyboard)
	}

	if action == menuActionSetSystemLang {
		if len(payload) != 2 {
			return nil
		}

		langCode := enums.Language(payload[1])
		isSupported := false
		for _, lang := range getSupportedSystemLanguages() {
			if lang == langCode {
				isSupported = true
				break
			}
		}
		if !isSupported {
			return nil
		}

		user, err := services.UpdateUserSystemLanguage(callback.From.ID, langCode)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		updatedTexts := GetBotTexts(user.Settings.SystemLanguage)
		keyboard := buildSettingsKeyboard(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, updatedTexts)
		messageText := BuildSettingsText(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, updatedTexts)
		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, keyboard)
	}

	if action == menuActionToggleDailyExercises {
		user, err := services.UpdateUserTelegramDailyQuestionsEnabled(callback.From.ID, true)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		updatedTexts := GetBotTexts(user.Settings.SystemLanguage)
		keyboard := buildSettingsKeyboard(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, updatedTexts)
		messageText := BuildSettingsText(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, updatedTexts)
		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, keyboard)
	}

	if action == menuActionSettings {
		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		keyboard := buildSettingsKeyboard(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, t)
		messageText := BuildSettingsText(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, t)
		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, keyboard)
	}

	selectionText, ok := menuActionToText(action, t)
	if !ok {
		return nil
	}

	return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, selectionText, getMenuBackKeyboard(t))
}
