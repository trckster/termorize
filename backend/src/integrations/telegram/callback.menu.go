package telegram

import (
	"termorize/src/enums"
	"termorize/src/services"
)

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

		return editMainMenu(callback, t)
	}

	if action == menuActionTranslationPair {
		if _, err := services.UpdateUserTelegramState(callback.From.ID, enums.TelegramStateNone); err != nil {
			return err
		}

		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}
		if user == nil {
			return nil
		}

		return editTranslationPair(
			callback,
			user.Settings.TranslationSourceLanguage,
			user.Settings.TranslationTargetLanguage,
			t,
		)
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

	if action == menuActionVocabulary {
		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		messageText, err := buildVocabularyMenuText(user.ID, t)
		if err != nil {
			return err
		}

		return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, messageText, buildVocabularyOverviewKeyboard(t))
	}

	if action == menuActionStatistics {
		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		messageText, err := buildStatisticsMenuText(user.ID, t)
		if err != nil {
			return err
		}

		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, getMenuBackKeyboard(t))
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

	if action == menuActionChangePairSourceLang || action == menuActionChangePairTargetLang {
		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		isSource := action == menuActionChangePairSourceLang
		keyboard := buildTranslationPairLanguageSelectionKeyboard(
			user.Settings.TranslationSourceLanguage,
			user.Settings.TranslationTargetLanguage,
			isSource,
			t,
		)
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
		if !enums.IsSupportedLanguage(langCode) {
			return nil
		}
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

	if action == menuActionSetPairSourceLang || action == menuActionSetPairTargetLang {
		if len(payload) != 2 {
			return nil
		}

		langCode := enums.Language(payload[1])
		if !enums.IsSupportedLanguage(langCode) {
			return nil
		}

		isSource := action == menuActionSetPairSourceLang
		user, err := services.UpdateUserTranslationLanguage(callback.From.ID, isSource, langCode)
		if err != nil {
			return err
		}
		if user == nil {
			return nil
		}

		return editTranslationPair(
			callback,
			user.Settings.TranslationSourceLanguage,
			user.Settings.TranslationTargetLanguage,
			t,
		)
	}

	if action == menuActionSwapTranslationPair {
		user, err := services.SwapUserTranslationLanguages(callback.From.ID)
		if err != nil {
			return err
		}
		if user == nil {
			return nil
		}

		return editTranslationPair(
			callback,
			user.Settings.TranslationSourceLanguage,
			user.Settings.TranslationTargetLanguage,
			t,
		)
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

	return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, selectionText, getMenuBackKeyboard(t))
}

func editMainMenu(callback *callbackQuery, t BotTexts) error {
	user, err := services.GetUserByTelegramID(callback.From.ID)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	sourceLanguage := user.Settings.TranslationSourceLanguage
	targetLanguage := user.Settings.TranslationTargetLanguage
	return EditMessageTextWithInlineKeyboardMarkdown(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		t.Menu,
		getMenuKeyboard(sourceLanguage, targetLanguage, t),
	)
}

func editTranslationPair(callback *callbackQuery, sourceLanguage, targetLanguage enums.Language, t BotTexts) error {
	return EditMessageTextWithInlineKeyboardMarkdown(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		buildTranslationPairText(sourceLanguage, targetLanguage, t),
		buildTranslationPairKeyboard(sourceLanguage, targetLanguage, t),
	)
}
