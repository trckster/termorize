package telegram

import (
	"strings"
	"termorize/src/enums"
	"termorize/src/services"
)

func handleDescriptionLanguageCallback(callback *callbackQuery, payload []string, texts BotTexts) (bool, error) {
	if callback.Message == nil || len(payload) != 3 {
		return false, nil
	}
	action := payload[0]
	if action != exerciseActionIgnoreDescriptionLanguage && action != exerciseActionUndoDescriptionLanguage {
		return false, nil
	}
	exerciseID, err := parseCallbackUUID(payload[1])
	if err != nil {
		return true, nil
	}
	language := enums.Language(payload[2])
	if !enums.IsSupportedLanguage(language) {
		return true, nil
	}

	exercise, err := services.GetExerciseByTelegramMessage(callback.Message.MessageID, callback.From.ID)
	if err != nil {
		return true, err
	}
	if exercise == nil || exercise.ExerciseID != exerciseID ||
		(exercise.ExerciseType != enums.ExerciseTypeDescriptionDirect && exercise.ExerciseType != enums.ExerciseTypeDescriptionReversed) {
		return true, nil
	}
	descriptionLanguage := exercise.OriginalLanguage
	if exercise.ExerciseType == enums.ExerciseTypeDescriptionReversed {
		descriptionLanguage = exercise.TranslationLanguage
	}
	if language != descriptionLanguage {
		return true, nil
	}

	if action == exerciseActionIgnoreDescriptionLanguage {
		if _, err := services.IgnoreDescriptionLanguageForExercise(exerciseID, exercise.UserID); err != nil {
			return true, err
		}
		return true, editCancelledDescriptionExerciseMessage(
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			callback.Message.Text,
			exerciseID,
			language,
			texts,
		)
	}

	if _, err := services.RemoveIgnoredDescriptionLanguage(exercise.UserID, language); err != nil {
		return true, err
	}
	return true, editMessageTextTolerant(editMessageTextRequest{
		ChatID:      callback.Message.Chat.ID,
		MessageID:   callback.Message.MessageID,
		Text:        strings.TrimSuffix(callback.Message.Text, "\n\n"+texts.ExerciseDescriptionCancelled),
		ReplyMarkup: &inlineKeyboardMarkup{InlineKeyboard: [][]inlineKeyboardButton{}},
	})
}
