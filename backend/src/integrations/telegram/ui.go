package telegram

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"unicode/utf8"

	"github.com/google/uuid"

	"termorize/src/enums"
	"termorize/src/services"
)

const (
	callbackTypeMenu          = "menu"
	callbackTypeExercise      = "exercise"
	callbackTypeVocabulary    = "vocabulary"
	callbackTypePronunciation = "pronunciation"

	menuActionBack                 = "back"
	menuActionCancel               = "cancel"
	menuActionDeleteTranslation    = "delete_translation"
	menuActionAddTranslation       = "add_translation"
	menuActionVocabulary           = "your_vocabulary"
	menuActionStatistics           = "statistics"
	menuActionSettings             = "settings"
	menuActionWhatsGoingOn         = "whats_going_on"
	menuActionTranslationPair      = "translation_pair"
	menuActionChangeSourceLang     = "change_source_lang"
	menuActionChangeTargetLang     = "change_target_lang"
	menuActionChangePairSourceLang = "change_pair_source_lang"
	menuActionChangePairTargetLang = "change_pair_target_lang"
	menuActionChangeSystemLang     = "change_system_lang"
	menuActionToggleDailyExercises = "toggle_daily_exercises"
	menuActionSetSourceLang        = "set_source_lang"
	menuActionSetTargetLang        = "set_target_lang"
	menuActionSetPairSourceLang    = "set_pair_source_lang"
	menuActionSetPairTargetLang    = "set_pair_target_lang"
	menuActionSwapTranslationPair  = "swap_translation_pair"
	menuActionSetSystemLang        = "set_system_lang"

	exerciseActionAnswer              = "answer"
	exerciseActionIDK                 = "idk"
	exerciseActionMatchTap            = "mt"
	exerciseActionMatchNoop           = "mn"
	exerciseActionCharacterTap        = "ct"
	exerciseActionCharacterNoop       = "cn"
	exerciseActionCharacterBackspace  = "cc"
	exerciseActionIgnoreAudioLanguage = "ai"
	exerciseActionUndoAudioLanguage   = "au"

	vocabularyActionAdd    = "add"
	vocabularyActionDelete = "delete"

	// Telegram clients elide inline-button labels by rendered width. Twelve
	// runes is the heuristic cutoff for keeping two exercise buttons per row.
	exerciseCompactButtonMaxRunes = 12
)

func getMenuKeyboard(sourceLanguage, targetLanguage enums.Language, t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{
		{{Text: t.ButtonOpenApp, URL: telegramMiniAppURL}},
		{{Text: formatTranslationPair(sourceLanguage, targetLanguage, t), CallbackData: callbackTypeMenu + ":" + menuActionTranslationPair}},
		{{Text: t.ButtonAddTranslation, CallbackData: callbackTypeMenu + ":" + menuActionAddTranslation}, {Text: t.ButtonDeleteWord, CallbackData: callbackTypeMenu + ":" + menuActionDeleteTranslation}},
		{{Text: t.ButtonVocabulary, CallbackData: callbackTypeMenu + ":" + menuActionVocabulary}, {Text: t.ButtonStatistics, CallbackData: callbackTypeMenu + ":" + menuActionStatistics}},
		{{Text: t.ButtonSettings, CallbackData: callbackTypeMenu + ":" + menuActionSettings}, {Text: t.ButtonWhatsGoingOn, CallbackData: callbackTypeMenu + ":" + menuActionWhatsGoingOn}},
	}
}

func buildTranslationPairKeyboard(sourceLanguage, targetLanguage enums.Language, t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{
		{
			{Text: sourceLanguage.Flag() + " " + t.ButtonChangeSourceLanguage, CallbackData: callbackTypeMenu + ":" + menuActionChangePairSourceLang},
			{Text: targetLanguage.Flag() + " " + t.ButtonChangeTargetLanguage, CallbackData: callbackTypeMenu + ":" + menuActionChangePairTargetLang},
		},
		{{Text: t.ButtonSwapDirection, CallbackData: callbackTypeMenu + ":" + menuActionSwapTranslationPair}},
		{{Text: t.ButtonBack, CallbackData: callbackTypeMenu + ":" + menuActionBack}},
	}
}

func buildAudioExerciseKeyboard(exerciseID uuid.UUID, spokenLanguage enums.Language, t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{
		{{
			Text:         t.ButtonExerciseIDK,
			CallbackData: callbackTypeExercise + ":" + exerciseActionIDK + ":" + exerciseID.String(),
		}},
		{{
			Text:         fmt.Sprintf(t.ButtonIgnoreAudioLanguageFormat, localizedLanguageName(spokenLanguage, t)),
			CallbackData: callbackTypeExercise + ":" + exerciseActionIgnoreAudioLanguage + ":" + compactCallbackUUID(exerciseID) + ":" + string(spokenLanguage),
		}},
	}
}

func buildAudioUndoKeyboard(exerciseID uuid.UUID, spokenLanguage enums.Language, t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{{{
		Text:         fmt.Sprintf(t.ButtonRemoveAudioLanguageFormat, localizedLanguageName(spokenLanguage, t)),
		CallbackData: callbackTypeExercise + ":" + exerciseActionUndoAudioLanguage + ":" + compactCallbackUUID(exerciseID) + ":" + string(spokenLanguage),
	}}}
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

func buildVocabularyAddKeyboard(translationID uuid.UUID, t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{{{
		Text:         t.ButtonVocabularyAdd,
		CallbackData: callbackTypeVocabulary + ":" + vocabularyActionAdd + ":" + translationID.String(),
	}}, pronunciationButtonRow(translationID, t)}
}

func buildVocabularyDeleteKeyboard(vocabularyID uuid.UUID, translationID uuid.UUID, t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{{{
		Text:         t.ButtonVocabularyDelete,
		CallbackData: callbackTypeVocabulary + ":" + vocabularyActionDelete + ":" + vocabularyID.String(),
	}}, pronunciationButtonRow(translationID, t)}
}

func buildPronunciationKeyboard(translationID uuid.UUID, t BotTexts) [][]inlineKeyboardButton {
	return [][]inlineKeyboardButton{pronunciationButtonRow(translationID, t)}
}

func pronunciationButtonRow(translationID uuid.UUID, t BotTexts) []inlineKeyboardButton {
	return []inlineKeyboardButton{{
		Text:         t.ButtonPronunciation,
		CallbackData: callbackTypePronunciation + ":" + compactCallbackUUID(translationID),
	}}
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

	return buildLanguageSelectionKeyboardForAction(excludeLang1, excludeLang2, action, menuActionAddTranslation, t)
}

func buildTranslationPairLanguageSelectionKeyboard(excludeLang1, excludeLang2 enums.Language, isSource bool, t BotTexts) [][]inlineKeyboardButton {
	action := menuActionSetPairTargetLang
	if isSource {
		action = menuActionSetPairSourceLang
	}

	return buildLanguageSelectionKeyboardForAction(excludeLang1, excludeLang2, action, menuActionTranslationPair, t)
}

func buildLanguageSelectionKeyboardForAction(excludeLang1, excludeLang2 enums.Language, action, cancelAction string, t BotTexts) [][]inlineKeyboardButton {
	var rows [][]inlineKeyboardButton
	for _, langStr := range enums.AllLanguages() {
		lang := enums.Language(langStr)
		if lang == excludeLang1 || lang == excludeLang2 {
			continue
		}
		rows = append(rows, []inlineKeyboardButton{{
			Text:         localizedLanguageWithFlag(lang, t),
			CallbackData: callbackTypeMenu + ":" + action + ":" + langStr,
		}})
	}

	rows = append(rows, []inlineKeyboardButton{{
		Text:         t.ButtonCancel,
		CallbackData: callbackTypeMenu + ":" + cancelAction,
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
	compactExerciseID := compactCallbackUUID(exerciseID)
	buttons := make([]inlineKeyboardButton, 0, len(options))
	labels := make([]string, 0, len(options))

	for _, option := range options {
		buttons = append(buttons, inlineKeyboardButton{
			Text:         option.Label,
			CallbackData: callbackTypeExercise + ":" + exerciseActionAnswer + ":" + compactExerciseID + ":" + compactCallbackUUID(option.VocabularyID),
		})
		labels = append(labels, option.Label)
	}

	return groupExerciseButtons(buttons, exerciseButtonsPerRow(labels))
}

func buildMatchKeyboard(exerciseID uuid.UUID, board *services.MatchBoardState) [][]inlineKeyboardButton {
	compactExerciseID := compactCallbackUUID(exerciseID)
	noopCallback := callbackTypeExercise + ":" + exerciseActionMatchNoop

	buttons := make([]inlineKeyboardButton, 0, len(board.Order))
	labels := make([]string, 0, len(board.Order))
	for _, canonical := range board.Order {
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

		buttons = append(buttons, button)
		labels = append(labels, card.Word)
	}

	return groupExerciseButtons(buttons, exerciseButtonsPerRow(labels))
}

func exerciseButtonsPerRow(labels []string) int {
	for _, label := range labels {
		if utf8.RuneCountInString(label) > exerciseCompactButtonMaxRunes {
			return 1
		}
	}

	return 2
}

func groupExerciseButtons(buttons []inlineKeyboardButton, buttonsPerRow int) [][]inlineKeyboardButton {
	rows := make([][]inlineKeyboardButton, 0, (len(buttons)+buttonsPerRow-1)/buttonsPerRow)
	for index, button := range buttons {
		rowIndex := index / buttonsPerRow
		if len(rows) <= rowIndex {
			rows = append(rows, make([]inlineKeyboardButton, 0, buttonsPerRow))
		}
		rows[rowIndex] = append(rows[rowIndex], button)
	}

	return rows
}

func buildCharacterKeyboard(exerciseID uuid.UUID, board *services.CharacterBoardState, t BotTexts) [][]inlineKeyboardButton {
	if board == nil || len(board.Characters) == 0 {
		return [][]inlineKeyboardButton{}
	}

	side := int(math.Ceil(math.Sqrt(float64(len(board.Order)))))
	compactExerciseID := compactCallbackUUID(exerciseID)
	noopCallback := callbackTypeExercise + ":" + exerciseActionCharacterNoop
	chosen := make(map[int]bool, len(board.Chosen))
	for _, index := range board.Chosen {
		chosen[index] = true
	}

	rows := make([][]inlineKeyboardButton, 0, side+1)
	for rowIndex := 0; rowIndex < side; rowIndex++ {
		row := make([]inlineKeyboardButton, 0, side)
		for columnIndex := 0; columnIndex < side; columnIndex++ {
			slot := rowIndex*side + columnIndex
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
			row = append(row, button)
		}
		rows = append(rows, row)
	}
	rows = append(rows, []inlineKeyboardButton{
		{
			Text:         "⌫",
			CallbackData: callbackTypeExercise + ":" + exerciseActionCharacterBackspace + ":" + compactExerciseID,
		},
		{
			Text:         t.ButtonExerciseIDK,
			CallbackData: callbackTypeExercise + ":" + exerciseActionIDK + ":" + exerciseID.String(),
		},
	})

	return rows
}

func displayCharacter(character string) string {
	switch character {
	case " ":
		return "⎵"
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
	return enums.AllSystemLanguageValues()
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
