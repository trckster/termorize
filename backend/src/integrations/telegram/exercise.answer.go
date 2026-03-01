package telegram

import (
	"strings"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/services"
)

func handleExerciseAnswer(message *message) (bool, error) {
	if message.ReplyToMessage == nil {
		return false, nil
	}

	telegramID, _, _, _ := extractMessageUser(message)
	exercise, err := services.GetExerciseByTelegramMessage(message.ReplyToMessage.MessageID, telegramID)
	if err != nil {
		return false, err
	}

	if exercise == nil {
		return false, nil
	}

	switch exercise.Status {
	case enums.ExerciseStatusIgnored:
		return true, SendMessage(message.Chat.ID, telegramTextExerciseOutdated)
	case enums.ExerciseStatusCompleted:
		return true, SendMessage(message.Chat.ID, telegramTextExerciseCompleted)
	case enums.ExerciseStatusFailed:
		return true, SendMessage(message.Chat.ID, telegramTextExerciseFailed)
	case enums.ExerciseStatusPending, enums.ExerciseStatusInProgress:
	default:
		return true, nil
	}

	if err := removeMessageInlineKeyboard(message.Chat.ID, message.ReplyToMessage.MessageID); err != nil {
		logger.L().Warnw("failed to remove inline keyboard", "error", err, "chat_id", message.Chat.ID, "message_id", message.ReplyToMessage.MessageID)
	}

	if isCorrectExerciseAnswer(message.Text, exercise.ExerciseType, exercise.OriginalWord, exercise.TranslationWord) {
		if err := services.CompleteExercise(exercise.ExerciseID); err != nil {
			return false, err
		}
		return true, SendMessage(message.Chat.ID, telegramTextExerciseSuccess)
	}

	updated, err := services.FailExercise(exercise.ExerciseID)
	if err != nil {
		return false, err
	}

	if !updated {
		return true, nil
	}

	answerText := buildIDKAnswer(exercise.OriginalWord, exercise.TranslationWord, exercise.ExerciseType)
	return true, SendMessageMarkdown(message.Chat.ID, answerText)
}

func isCorrectExerciseAnswer(answer string, exerciseType enums.ExerciseType, originalWord string, translationWord string) bool {
	normalizedAnswer := strings.TrimSpace(answer)

	switch exerciseType {
	case enums.ExerciseTypeBasicDirect:
		return strings.EqualFold(normalizedAnswer, strings.TrimSpace(translationWord))
	case enums.ExerciseTypeBasicReversed:
		return strings.EqualFold(normalizedAnswer, strings.TrimSpace(originalWord))
	default:
		return false
	}
}

func buildIDKAnswer(originalWord string, translationWord string, exerciseType enums.ExerciseType) string {
	if exerciseType == enums.ExerciseTypeBasicReversed {
		return telegramTextIDKOriginalPrefix + "*" + originalWord + "*"
	}

	return telegramTextIDKTranslationPrefix + "*" + translationWord + "*"
}
