package telegram

import (
	"fmt"
	"strings"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/utils"
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

	answerDistance, almostCorrectDistance := getExerciseAnswerMetrics(message.Text, exercise.ExerciseType, exercise.Vocabulary)

	if answerDistance == 0 {
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

	if answerDistance <= almostCorrectDistance {
		updated, translationKnowledge, err := services.CompleteExerciseWithProgress(exercise.ExerciseID, services.ExerciseAlmostCorrectProgressDelta)
		if err != nil {
			return false, err
		}

		if !updated {
			return true, nil
		}

		answerText := buildExerciseAlmostResultText(
			exercise.OriginalWord,
			exercise.TranslationWord,
			exercise.OriginalLanguage,
			exercise.TranslationLanguage,
			translationKnowledge,
			t,
		)
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

func buildExerciseAlmostResultText(originalWord string, translationWord string, originalLanguage enums.Language, translationLanguage enums.Language, translationKnowledge int, t BotTexts) string {
	answerPair := buildExerciseAnswerPairText(originalWord, translationWord, originalLanguage, translationLanguage, t)
	return t.ExerciseAlmost + "\n\n" + answerPair + "\n\n" + fmt.Sprintf(t.ExerciseTranslationKnowledgeUpFormat, translationKnowledge)
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

func getExerciseAnswerMetrics(answer string, exerciseType enums.ExerciseType, vocabulary []models.Vocabulary) (int, int) {
	expectedAnswer := getExpectedExerciseAnswer(exerciseType, vocabulary)
	if expectedAnswer == "" {
		return 2, 1
	}

	distance := utils.LevenshteinDistance(normalizeExerciseAnswer(answer), expectedAnswer)
	return distance, getAlmostCorrectDistanceThreshold(expectedAnswer)
}

func normalizeExerciseAnswer(value string) string {
	return strings.ToLower(russianYoReplacer.Replace(strings.TrimSpace(value)))
}

func getExpectedExerciseAnswer(exerciseType enums.ExerciseType, vocabulary []models.Vocabulary) string {
	if len(vocabulary) == 0 || vocabulary[0].Translation == nil {
		return ""
	}

	vocabularyItem := vocabulary[0]

	switch exerciseType {
	case enums.ExerciseTypeBasicDirect:
		if vocabularyItem.Translation.Translation == nil {
			return ""
		}
		return normalizeExerciseAnswer(vocabularyItem.Translation.Translation.Word)
	case enums.ExerciseTypeBasicReversed:
		if vocabularyItem.Translation.Original == nil {
			return ""
		}
		return normalizeExerciseAnswer(vocabularyItem.Translation.Original.Word)
	default:
		return ""
	}
}

func getAlmostCorrectDistanceThreshold(expectedAnswer string) int {
	if len([]rune(expectedAnswer)) > 10 {
		return 2
	}

	return 1
}
