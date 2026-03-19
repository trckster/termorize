package telegram

import "termorize/src/enums"

const (
	callbackTypeMenu       = "menu"
	callbackTypeExercise   = "exercise"
	callbackTypeVocabulary = "vocabulary"

	menuActionBack              = "back"
	menuActionCancel            = "cancel"
	menuActionDeleteTranslation = "delete_translation"
	menuActionAddTranslation    = "add_translation"
	menuActionVocabulary        = "your_vocabulary"
	menuActionStatistics        = "statistics"
	menuActionSettings          = "settings"
	menuActionWhatsGoingOn      = "whats_going_on"
	menuActionChangeSourceLang  = "change_source_lang"
	menuActionChangeTargetLang  = "change_target_lang"
	menuActionChangeSystemLang  = "change_system_lang"
	menuActionSetSourceLang     = "set_source_lang"
	menuActionSetTargetLang     = "set_target_lang"
	menuActionSetSystemLang     = "set_system_lang"

	exerciseActionIDK = "idk"

	vocabularyActionAdd    = "add"
	vocabularyActionDelete = "delete"
)

func getMenuKeyboard(t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{
		{{Text: t.ButtonOpenApp, URL: telegramMiniAppURL}},
		{{Text: t.ButtonAddTranslation, CallbackData: callbackTypeMenu + ":" + menuActionAddTranslation}, {Text: t.ButtonDeleteWord, CallbackData: callbackTypeMenu + ":" + menuActionDeleteTranslation}},
		{{Text: t.ButtonVocabulary, CallbackData: callbackTypeMenu + ":" + menuActionVocabulary}, {Text: t.ButtonStatistics, CallbackData: callbackTypeMenu + ":" + menuActionStatistics}},
		{{Text: t.ButtonSettings, CallbackData: callbackTypeMenu + ":" + menuActionSettings}, {Text: t.ButtonWhatsGoingOn, CallbackData: callbackTypeMenu + ":" + menuActionWhatsGoingOn}},
	}
}

func getMenuBackKeyboard(t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{{{Text: t.ButtonBack, CallbackData: callbackTypeMenu + ":" + menuActionBack}}}
}

func buildSettingsKeyboard(systemLang enums.Language, t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{
		{{Text: t.ButtonSystemLanguage + " " + systemLang.Flag(), CallbackData: callbackTypeMenu + ":" + menuActionChangeSystemLang}},
		{{Text: t.ButtonBack, CallbackData: callbackTypeMenu + ":" + menuActionBack}},
	}
}

func getMenuCancelKeyboard(t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{{{Text: t.ButtonCancel, CallbackData: callbackTypeMenu + ":" + menuActionCancel}}}
}

func buildVocabularyAddKeyboard(translationID string, t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{{{
		Text:         t.ButtonVocabularyAdd,
		CallbackData: callbackTypeVocabulary + ":" + vocabularyActionAdd + ":" + translationID,
	}}}
}

func buildVocabularyDeleteKeyboard(vocabularyID string, t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{{{
		Text:         t.ButtonVocabularyDelete,
		CallbackData: callbackTypeVocabulary + ":" + vocabularyActionDelete + ":" + vocabularyID,
	}}}
}

func buildAddTranslationKeyboard(sourceLang, targetLang enums.Language, t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{
		{
			{Text: t.ButtonChangeLanguagePrefix + sourceLang.Flag(), CallbackData: callbackTypeMenu + ":" + menuActionChangeSourceLang},
			{Text: t.ButtonChangeLanguagePrefix + targetLang.Flag(), CallbackData: callbackTypeMenu + ":" + menuActionChangeTargetLang},
		},
		{{Text: t.ButtonCancel, CallbackData: callbackTypeMenu + ":" + menuActionCancel}},
	}
}

func buildLanguageSelectionKeyboard(excludeLang1, excludeLang2 enums.Language, isSource bool, t BotTexts) [][]inlineKeyboardButton {
	action := menuActionSetTargetLang
	if isSource {
		action = menuActionSetSourceLang
	}

	var rows [][]inlineKeyboardButton
	for _, langStr := range enums.AllLanguages() {
		lang := enums.Language(langStr)
		if lang == excludeLang1 || lang == excludeLang2 {
			continue
		}
		rows = append(rows, []inlineKeyboardButton{{
			Text:         lang.DisplayNameWithFlag(),
			CallbackData: callbackTypeMenu + ":" + action + ":" + langStr,
		}})
	}

	rows = append(rows, []inlineKeyboardButton{{
		Text:         t.ButtonCancel,
		CallbackData: callbackTypeMenu + ":" + menuActionAddTranslation,
	}})

	return rows
}

func buildSystemLanguageSelectionKeyboard(t BotTexts) [][]inlineKeyboardButton {
	var rows [][]inlineKeyboardButton
	for _, lang := range getSupportedSystemLanguages() {
		langStr := string(lang)
		rows = append(rows, []inlineKeyboardButton{{
			Text:         lang.DisplayNameWithFlag(),
			CallbackData: callbackTypeMenu + ":" + menuActionSetSystemLang + ":" + langStr,
		}})
	}

	rows = append(rows, []inlineKeyboardButton{{
		Text:         t.ButtonCancel,
		CallbackData: callbackTypeMenu + ":" + menuActionSettings,
	}})

	return rows
}

func getSupportedSystemLanguages() []enums.Language {
	return []enums.Language{enums.LanguageEn, enums.LanguageRu}
}

func menuActionToText(action string, t BotTexts) (string, bool) {
	switch action {
	case menuActionVocabulary:
		return t.MenuVocabulary, true
	case menuActionStatistics:
		return t.MenuStatistics, true
	case menuActionSettings:
		return t.MenuSettings, true
	case menuActionWhatsGoingOn:
		return t.MenuWhatsGoingOn, true
	default:
		return "", false
	}
}
