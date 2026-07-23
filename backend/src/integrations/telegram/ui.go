package telegram

import (
	"encoding/base64"
	"math"
	"strconv"

	"github.com/google/uuid"

	"termorize/src/enums"
	"termorize/src/services"
)

const (
	callbackTypeMenu       = "menu"
	callbackTypeExercise   = "exercise"
	callbackTypeVocabulary = "vocabulary"

	menuActionBack                 = "back"
	menuActionCancel               = "cancel"
	menuActionDeleteTranslation    = "delete_translation"
	menuActionAddTranslation       = "add_translation"
	menuActionVocabulary           = "your_vocabulary"
	menuActionStatistics           = "statistics"
	menuActionSettings             = "settings"
	menuActionWhatsGoingOn         = "whats_going_on"
	menuActionChangeSourceLang     = "change_source_lang"
	menuActionChangeTargetLang     = "change_target_lang"
	menuActionChangeSystemLang     = "change_system_lang"
	menuActionToggleDailyExercises = "toggle_daily_exercises"
	menuActionSetSourceLang        = "set_source_lang"
	menuActionSetTargetLang        = "set_target_lang"
	menuActionSetSystemLang        = "set_system_lang"

	exerciseActionAnswer         = "answer"
	exerciseActionIDK            = "idk"
	exerciseActionMatchTap       = "mt"
	exerciseActionMatchNoop      = "mn"
	exerciseActionCharacterTap   = "ct"
	exerciseActionCharacterNoop  = "cn"
	exerciseActionCharacterClear = "cc"

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

func buildVocabularyOverviewKeyboard(t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{
		{{Text: t.ButtonBack, CallbackData: callbackTypeMenu + ":" + menuActionBack}},
	}
}

func buildSettingsKeyboard(systemLang enums.Language, dailyExercisesEnabled bool, t BotTexts) [][]inlineKeyboardButton {
	dailyExercisesText := t.ButtonEnableDailyExercises
	if dailyExercisesEnabled {
		dailyExercisesText = t.ButtonDisableDailyExercises
	}

	return [][]inlineKeyboardButton{
		{{Text: t.ButtonChangeSystemLanguage, CallbackData: callbackTypeMenu + ":" + menuActionChangeSystemLang}},
		{{Text: dailyExercisesText, CallbackData: callbackTypeMenu + ":" + menuActionToggleDailyExercises}},
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

func buildExerciseKeyboard(exerciseID uuid.UUID, options []services.ExerciseOption) [][]inlineKeyboardButton {
	rows := make([][]inlineKeyboardButton, 0, 2)
	compactExerciseID := compactCallbackUUID(exerciseID)

	for index, option := range options {
		rowIndex := index / 2
		if len(rows) <= rowIndex {
			rows = append(rows, []inlineKeyboardButton{})
		}

		rows[rowIndex] = append(rows[rowIndex], inlineKeyboardButton{
			Text:         option.Label,
			CallbackData: callbackTypeExercise + ":" + exerciseActionAnswer + ":" + compactExerciseID + ":" + compactCallbackUUID(option.VocabularyID),
		})
	}

	return rows
}

func buildMatchKeyboard(exerciseID uuid.UUID, board *services.MatchBoardState) [][]inlineKeyboardButton {
	compactExerciseID := compactCallbackUUID(exerciseID)
	noopCallback := callbackTypeExercise + ":" + exerciseActionMatchNoop

	rows := make([][]inlineKeyboardButton, 0, len(board.Order)/2)
	for slot, canonical := range board.Order {
		if canonical < 0 || canonical >= len(board.Cards) {
			continue
		}
		card := board.Cards[canonical]

		var button inlineKeyboardButton
		if result, resolved := board.Resolved[card.VocabularyID]; resolved {
			prefix := "❌ "
			switch result {
			case services.ExerciseVocabularyResultCorrect:
				prefix = "✅ "
			case services.ExerciseVocabularyResultAlmost:
				prefix = "👌 "
			}
			button = inlineKeyboardButton{Text: prefix + card.Word, CallbackData: noopCallback}
		} else {
			text := card.Word
			if board.CardWrong[card.ID] > 0 {
				text = "⚠️ " + text
			}
			if canonical == board.Pending {
				text = "▸ " + text
			}
			button = inlineKeyboardButton{
				Text:         text,
				CallbackData: callbackTypeExercise + ":" + exerciseActionMatchTap + ":" + compactExerciseID + ":" + strconv.Itoa(canonical),
			}
		}

		rowIndex := slot / 2
		if len(rows) <= rowIndex {
			rows = append(rows, []inlineKeyboardButton{})
		}
		rows[rowIndex] = append(rows[rowIndex], button)
	}

	return rows
}

func buildCharacterKeyboard(exerciseID uuid.UUID, board *services.CharacterBoardState, texts BotTexts) [][]inlineKeyboardButton {
	if board == nil || len(board.Characters) == 0 {
		return [][]inlineKeyboardButton{}
	}

	side := int(math.Ceil(math.Sqrt(float64(len(board.Characters) + 1))))
	compactExerciseID := compactCallbackUUID(exerciseID)
	noopCallback := callbackTypeExercise + ":" + exerciseActionCharacterNoop
	chosen := make(map[int]bool, len(board.Chosen))
	for _, index := range board.Chosen {
		chosen[index] = true
	}

	rows := make([][]inlineKeyboardButton, side)
	for rowIndex := 0; rowIndex < side; rowIndex++ {
		rows[rowIndex] = make([]inlineKeyboardButton, 0, side)
		for columnIndex := 0; columnIndex < side; columnIndex++ {
			slot := rowIndex*side + columnIndex
			if slot == side*side-1 {
				rows[rowIndex] = append(rows[rowIndex], inlineKeyboardButton{
					Text:         texts.ButtonExerciseClear,
					CallbackData: callbackTypeExercise + ":" + exerciseActionCharacterClear + ":" + compactExerciseID,
				})
				continue
			}

			button := inlineKeyboardButton{Text: " ", CallbackData: noopCallback}
			if slot < len(board.Order) {
				canonical := board.Order[slot]
				if canonical >= 0 && canonical < len(board.Characters) && !chosen[canonical] {
					button = inlineKeyboardButton{
						Text:         displayCharacter(board.Characters[canonical]),
						CallbackData: callbackTypeExercise + ":" + exerciseActionCharacterTap + ":" + compactExerciseID + ":" + strconv.Itoa(canonical),
					}
				}
			}
			rows[rowIndex] = append(rows[rowIndex], button)
		}
	}

	return rows
}

func displayCharacter(character string) string {
	switch character {
	case " ":
		return "␠"
	case "\t":
		return "⇥"
	default:
		return character
	}
}

func compactCallbackUUID(id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString(id[:])
}

func getSupportedSystemLanguages() []enums.Language {
	return []enums.Language{enums.LanguageEn, enums.LanguageRu}
}

func menuActionToText(action string, t BotTexts) (string, bool) {
	switch action {
	case menuActionStatistics:
		return t.MenuStatistics, true
	case menuActionWhatsGoingOn:
		return t.MenuWhatsGoingOn, true
	default:
		return "", false
	}
}
