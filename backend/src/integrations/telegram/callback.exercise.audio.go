package telegram

import (
	"termorize/src/enums"
	"termorize/src/services"
)

func handleAudioLanguageCallback(callback *callbackQuery, payload []string, texts BotTexts) (bool, error) {
	if callback.Message == nil || len(payload) != 3 {
		return false, nil
	}
	action := payload[0]
	if action != exerciseActionIgnoreAudioLanguage && action != exerciseActionUndoAudioLanguage {
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
		(exercise.ExerciseType != enums.ExerciseTypeAudioDirect && exercise.ExerciseType != enums.ExerciseTypeAudioReversed) {
		return true, nil
	}
	spokenLanguage := exercise.OriginalLanguage
	if exercise.ExerciseType == enums.ExerciseTypeAudioReversed {
		spokenLanguage = exercise.TranslationLanguage
	}
	if language != spokenLanguage {
		return true, nil
	}

	if action == exerciseActionIgnoreAudioLanguage {
		if _, err := services.IgnoreAudioLanguageForExercise(exerciseID, exercise.UserID); err != nil {
			return true, err
		}
		answerLanguage := exercise.TranslationLanguage
		if exercise.ExerciseType == enums.ExerciseTypeAudioReversed {
			answerLanguage = exercise.OriginalLanguage
		}
		return true, editCancelledAudioExerciseMessage(
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			answerLanguage,
			exerciseID,
			language,
			texts,
		)
	}

	if _, err := services.RemoveIgnoredAudioLanguage(exercise.UserID, language); err != nil {
		return true, err
	}
	return true, editMessageInlineKeyboardTolerant(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		[][]inlineKeyboardButton{},
	)
}
