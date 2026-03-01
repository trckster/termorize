package telegram

import (
	"fmt"
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
		updated, translationKnowledge, err := services.CompleteExercise(exercise.ExerciseID)
		if err != nil {
			return false, err
		}

		if !updated {
			return true, nil
		}

		answerText := buildExerciseSuccessResultText(translationKnowledge)
		return true, SendMessageMarkdown(message.Chat.ID, answerText)
	}

	updated, translationKnowledge, err := services.FailExercise(exercise.ExerciseID)
	if err != nil {
		return false, err
	}

	if !updated {
		return true, nil
	}

	answerText := buildExerciseInvalidResultText(
		exercise.OriginalWord,
		exercise.TranslationWord,
		exercise.OriginalLanguage,
		exercise.TranslationLanguage,
		translationKnowledge,
	)
	return true, SendMessageMarkdown(message.Chat.ID, answerText)
}

func buildExerciseSuccessResultText(translationKnowledge int) string {
	return telegramTextExerciseSuccess + "\n\n" + fmt.Sprintf(telegramTextExerciseTranslationKnowledgeUpFormat, translationKnowledge)
}

func buildExerciseInvalidResultText(originalWord string, translationWord string, originalLanguage enums.Language, translationLanguage enums.Language, translationKnowledge int) string {
	answerPair := buildExerciseAnswerPairText(originalWord, translationWord, originalLanguage, translationLanguage)
	return telegramTextExerciseInvalid + "\n\n" + answerPair + "\n\n" + fmt.Sprintf(telegramTextExerciseTranslationKnowledgeDownFormat, translationKnowledge)
}

func buildExerciseIDKResultText(originalWord string, translationWord string, originalLanguage enums.Language, translationLanguage enums.Language, translationKnowledge int) string {
	answerPair := buildExerciseAnswerPairText(originalWord, translationWord, originalLanguage, translationLanguage)
	return telegramTextExerciseIDK + "\n\n" + answerPair + "\n\n" + fmt.Sprintf(telegramTextExerciseTranslationKnowledgeDownFormat, translationKnowledge)
}

func buildExerciseAnswerPairText(originalWord string, translationWord string, originalLanguage enums.Language, translationLanguage enums.Language) string {
	return fmt.Sprintf(
		telegramTextExerciseAnswerPairFormat,
		originalLanguage.Flag(),
		originalWord,
		translationWord,
		translationLanguage.Flag(),
	)
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
