package telegram

const (
	callbackTypeMenu     = "menu"
	callbackTypeExercise = "exercise"

	menuActionBack              = "back"
	menuActionCancel            = "cancel"
	menuActionDeleteTranslation = "delete_translation"
	menuActionAddTranslation    = "add_translation"
	menuActionVocabulary        = "your_vocabulary"
	menuActionStatistics        = "statistics"
	menuActionSettings          = "settings"

	exerciseActionIDK = "idk"
)

var menuKeyboard = [][]inlineKeyboardButton{
	{{Text: telegramButtonMenuAddTranslation, CallbackData: callbackTypeMenu + ":" + menuActionAddTranslation}, {Text: telegramButtonMenuDeleteWord, CallbackData: callbackTypeMenu + ":" + menuActionDeleteTranslation}},
	{{Text: telegramButtonMenuVocabulary, CallbackData: callbackTypeMenu + ":" + menuActionVocabulary}, {Text: telegramButtonMenuStatistics, CallbackData: callbackTypeMenu + ":" + menuActionStatistics}},
	{{Text: telegramButtonMenuSettings, CallbackData: callbackTypeMenu + ":" + menuActionSettings}},
}

var menuBackKeyboard = [][]inlineKeyboardButton{{{Text: telegramButtonMenuBack, CallbackData: callbackTypeMenu + ":" + menuActionBack}}}
var menuCancelKeyboard = [][]inlineKeyboardButton{{{Text: telegramButtonMenuCancel, CallbackData: callbackTypeMenu + ":" + menuActionCancel}}}

func menuActionToText(action string) (string, bool) {
	switch action {
	case menuActionVocabulary:
		return telegramTextMenuVocabulary, true
	case menuActionStatistics:
		return telegramTextMenuStatistics, true
	case menuActionSettings:
		return telegramTextMenuSettings, true
	default:
		return "", false
	}
}
