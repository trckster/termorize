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
	menuActionSetSourceLang     = "set_source_lang"
	menuActionSetTargetLang     = "set_target_lang"

	exerciseActionIDK = "idk"

	vocabularyActionAdd    = "add"
	vocabularyActionDelete = "delete"
)

var menuKeyboard = [][]inlineKeyboardButton{
	{{Text: telegramButtonMenuOpenApp, URL: telegramMiniAppURL}},
	{{Text: telegramButtonMenuAddTranslation, CallbackData: callbackTypeMenu + ":" + menuActionAddTranslation}, {Text: telegramButtonMenuDeleteWord, CallbackData: callbackTypeMenu + ":" + menuActionDeleteTranslation}},
	{{Text: telegramButtonMenuVocabulary, CallbackData: callbackTypeMenu + ":" + menuActionVocabulary}, {Text: telegramButtonMenuStatistics, CallbackData: callbackTypeMenu + ":" + menuActionStatistics}},
	{{Text: telegramButtonMenuSettings, CallbackData: callbackTypeMenu + ":" + menuActionSettings}, {Text: telegramButtonMenuWhatsGoingOn, CallbackData: callbackTypeMenu + ":" + menuActionWhatsGoingOn}},
}

var menuBackKeyboard = [][]inlineKeyboardButton{{{Text: telegramButtonMenuBack, CallbackData: callbackTypeMenu + ":" + menuActionBack}}}
var menuCancelKeyboard = [][]inlineKeyboardButton{{{Text: telegramButtonMenuCancel, CallbackData: callbackTypeMenu + ":" + menuActionCancel}}}

func buildVocabularyAddKeyboard(translationID string) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{{{
		Text:         telegramButtonVocabularyAdd,
		CallbackData: callbackTypeVocabulary + ":" + vocabularyActionAdd + ":" + translationID,
	}}}
}

func buildVocabularyDeleteKeyboard(vocabularyID string) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{{{
		Text:         telegramButtonVocabularyDelete,
		CallbackData: callbackTypeVocabulary + ":" + vocabularyActionDelete + ":" + vocabularyID,
	}}}
}

func buildAddTranslationKeyboard(sourceLang, targetLang enums.Language) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{
		{
			{Text: "Change " + sourceLang.Flag(), CallbackData: callbackTypeMenu + ":" + menuActionChangeSourceLang},
			{Text: "Change " + targetLang.Flag(), CallbackData: callbackTypeMenu + ":" + menuActionChangeTargetLang},
		},
		{{Text: telegramButtonMenuCancel, CallbackData: callbackTypeMenu + ":" + menuActionCancel}},
	}
}

func buildLanguageSelectionKeyboard(excludeLang1, excludeLang2 enums.Language, isSource bool) [][]inlineKeyboardButton {
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
		Text:         telegramButtonMenuCancel,
		CallbackData: callbackTypeMenu + ":" + menuActionAddTranslation,
	}})

	return rows
}

func menuActionToText(action string) (string, bool) {
	switch action {
	case menuActionVocabulary:
		return telegramTextMenuVocabulary, true
	case menuActionStatistics:
		return telegramTextMenuStatistics, true
	case menuActionSettings:
		return telegramTextMenuSettings, true
	case menuActionWhatsGoingOn:
		return telegramTextMenuWhatsGoingOn, true
	default:
		return "", false
	}
}
