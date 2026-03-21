package telegram

import (
	"fmt"
	"strings"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/services"
)

var russianYoReplacer = strings.NewReplacer("ё", "е", "Ё", "Е")

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

	t := getBotTextsForTelegramID(telegramID)

	switch exercise.Status {
	case enums.ExerciseStatusIgnored:
		return true, SendMessage(message.Chat.ID, t.ExerciseOutdated)
	case enums.ExerciseStatusCompleted:
		return true, SendMessage(message.Chat.ID, t.ExerciseCompleted)
	case enums.ExerciseStatusFailed:
		return true, SendMessage(message.Chat.ID, t.ExerciseFailed)
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

		answerText := buildExerciseSuccessResultText(translationKnowledge, t)
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
		t,
	)
	return true, SendMessageMarkdown(message.Chat.ID, answerText)
}

func buildExerciseSuccessResultText(translationKnowledge int, t BotTexts) string {
	return t.ExerciseSuccess + "\n\n" + fmt.Sprintf(t.ExerciseTranslationKnowledgeUpFormat, translationKnowledge)
}

func buildExerciseInvalidResultText(originalWord string, translationWord string, originalLanguage enums.Language, translationLanguage enums.Language, translationKnowledge int, t BotTexts) string {
	answerPair := buildExerciseAnswerPairText(originalWord, translationWord, originalLanguage, translationLanguage, t)
	return t.ExerciseInvalid + "\n\n" + answerPair + "\n\n" + fmt.Sprintf(t.ExerciseTranslationKnowledgeDownFormat, translationKnowledge)
}

func buildExerciseIDKResultText(originalWord string, translationWord string, originalLanguage enums.Language, translationLanguage enums.Language, translationKnowledge int, t BotTexts) string {
	answerPair := buildExerciseAnswerPairText(originalWord, translationWord, originalLanguage, translationLanguage, t)
	return t.ExerciseIDK + "\n\n" + answerPair + "\n\n" + fmt.Sprintf(t.ExerciseTranslationKnowledgeDownFormat, translationKnowledge)
}

func buildExerciseAnswerPairText(originalWord string, translationWord string, originalLanguage enums.Language, translationLanguage enums.Language, t BotTexts) string {
	return fmt.Sprintf(
		t.ExerciseAnswerPairFormat,
		originalLanguage.Flag(),
		originalWord,
		translationWord,
		translationLanguage.Flag(),
	)
}

func isCorrectExerciseAnswer(answer string, exerciseType enums.ExerciseType, originalWord string, translationWord string) bool {
	normalizedAnswer := normalizeExerciseAnswer(answer)

	switch exerciseType {
	case enums.ExerciseTypeBasicDirect:
		return strings.EqualFold(normalizedAnswer, normalizeExerciseAnswer(translationWord))
	case enums.ExerciseTypeBasicReversed:
		return strings.EqualFold(normalizedAnswer, normalizeExerciseAnswer(originalWord))
	default:
		return false
	}
}

func normalizeExerciseAnswer(value string) string {
	return russianYoReplacer.Replace(strings.TrimSpace(value))
}
