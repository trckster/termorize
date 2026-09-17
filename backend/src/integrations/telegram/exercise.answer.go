package telegram

import (
	"errors"
	"fmt"
	"sort"
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
	if exercise.Deleted {
		t := getBotTextsForTelegramID(telegramID)
		return true, SendReplyMessage(message.Chat.ID, cancelledExerciseText(exercise.ExerciseType, t), message.MessageID)
	}

	t := getBotTextsForTelegramID(telegramID)
	switch exercise.Status {
	case enums.ExerciseStatusIgnored:
		return true, sendIgnoredExerciseMessage(message.Chat.ID, message.ReplyToMessage.MessageID, message.Chat.ID, exercise, t)
	case enums.ExerciseStatusCompleted:
		if editsExerciseAnswer(exercise.ExerciseType) {
			return true, nil
		}
		return true, SendMessage(message.Chat.ID, t.ExerciseCompleted)
	case enums.ExerciseStatusFailed:
		if editsExerciseAnswer(exercise.ExerciseType) {
			return true, nil
		}
		return true, SendMessage(message.Chat.ID, t.ExerciseFailed)
	case enums.ExerciseStatusPending, enums.ExerciseStatusInProgress:
	default:
		return true, nil
	}

	if len(exercise.Vocabulary) == 0 || exercise.Vocabulary[0].Translation == nil {
		_ = services.MarkExerciseVocabularyResultWithoutProgress(exercise.ExerciseID, services.ExerciseVocabularyResultIgnored, services.ExerciseVocabularyResultReasonDeletedVocabulary)
		_ = services.IgnoreExercise(exercise.ExerciseID)
		return true, sendDeletedVocabularyMessage(message.Chat.ID, message.ReplyToMessage.MessageID, message.Chat.ID, t.ExerciseVocabularyDeleted)
	}

	if exercise.ExerciseType == enums.ExerciseTypeChoiceDirect ||
		exercise.ExerciseType == enums.ExerciseTypeChoiceReversed ||
		exercise.ExerciseType == enums.ExerciseTypeMatchPairs {
		return true, SendReplyMessage(message.Chat.ID, t.ExerciseUseButtons, message.MessageID)
	}

	if !editsExerciseAnswer(exercise.ExerciseType) {
		if err := removeMessageInlineKeyboard(message.Chat.ID, message.ReplyToMessage.MessageID); err != nil {
			logger.L().Warnw("failed to remove inline keyboard", "error", err, "chat_id", message.Chat.ID, "message_id", message.ReplyToMessage.MessageID)
		}
	}

	result, err := services.VerifyExerciseAnswer(exercise.ExerciseID, exercise.UserID, message.Text)
	if err != nil {
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return true, nil
		}

		if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
			return true, sendDeletedVocabularyMessage(message.Chat.ID, message.ReplyToMessage.MessageID, message.Chat.ID, t.ExerciseVocabularyDeleted)
		}

		return false, err
	}

	return true, sendExerciseAnswerResult(message.Chat.ID, message.ReplyToMessage.MessageID, exercise, result, t)
}

func editsExerciseAnswer(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed,
		enums.ExerciseTypeCharactersDirect, enums.ExerciseTypeCharactersReversed,
		enums.ExerciseTypeDescriptionDirect, enums.ExerciseTypeDescriptionReversed:
		return true
	default:
		return false
	}
}

func sendExerciseAnswerResult(chatID int64, messageID int64, exercise *services.TelegramMessageExercise, result *services.VerifyAnswerResult, t BotTexts) error {
	var answerText string
	switch result.Result {
	case "correct":
		answerText = buildExerciseSuccessResultText(result.Knowledge, t)
		if editsExerciseAnswer(exercise.ExerciseType) {
			answerText = t.ExerciseSuccess + "\n\n" + buildExerciseAnswerPairText(
				exercise.OriginalWord, exercise.TranslationWord,
				exercise.OriginalLanguage, exercise.TranslationLanguage, t,
			) + "\n\n" + fmt.Sprintf(t.ExerciseTranslationKnowledgeUpFormat, result.Knowledge)
		}
	case "almost":
		answerText = buildExerciseAlmostResultText(
			exercise.OriginalWord, exercise.TranslationWord,
			exercise.OriginalLanguage, exercise.TranslationLanguage, result.Knowledge, t,
		)
	default:
		answerText = buildExerciseInvalidResultText(
			exercise.OriginalWord, exercise.TranslationWord,
			exercise.OriginalLanguage, exercise.TranslationLanguage, result.Knowledge, t,
		)
	}
	if editsExerciseAnswer(exercise.ExerciseType) {
		return editMessageTextTolerant(editMessageTextRequest{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        answerText,
			ParseMode:   telegramParseModeMarkdown,
			ReplyMarkup: &inlineKeyboardMarkup{InlineKeyboard: [][]inlineKeyboardButton{}},
		})
	}
	return SendMessageMarkdown(chatID, answerText)
}

func cancelledExerciseText(exerciseType enums.ExerciseType, texts BotTexts) string {
	if exerciseType == enums.ExerciseTypeDescriptionDirect || exerciseType == enums.ExerciseTypeDescriptionReversed {
		return texts.ExerciseDescriptionCancelled
	}
	return texts.ExerciseAudioCancelled
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

func buildMatchBoardText(board *services.MatchBoardState, t BotTexts) string {
	total := services.MatchPairsVocabularyCount
	matched := board.MatchedCount
	if matched > total {
		matched = total
	}

	dots := strings.Repeat("●", matched) + strings.Repeat("○", total-matched)

	return t.MatchExerciseTitle + "\n\n" + fmt.Sprintf(t.MatchProgressFormat, matched, total) + " " + dots
}

func buildMatchResultSummaryText(result *services.MatchPairsCompleteResult, t BotTexts) string {
	header := t.MatchSummaryCompleted
	if result.Status != enums.ExerciseStatusCompleted {
		header = t.MatchSummaryFailed
	}

	rows := make([]services.ExerciseListVocabulary, len(result.Results))
	copy(rows, result.Results)
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].Position < rows[j].Position
	})

	total := services.MatchPairsVocabularyCount
	matchedCount := 0

	lines := make([]string, 0, len(rows)+2)
	lines = append(lines, header, "")

	for _, row := range rows {
		emoji := "⚪"
		if row.ExerciseResult != nil {
			switch *row.ExerciseResult {
			case services.ExerciseVocabularyResultCorrect:
				emoji = "✅"
				matchedCount++
			case services.ExerciseVocabularyResultAlmost:
				emoji = "👌"
				matchedCount++
			case services.ExerciseVocabularyResultWrong:
				emoji = "❌"
			}
		}

		pairText := t.MatchPairDeletedVocabulary
		if row.Translation != nil && row.Translation.Original != nil && row.Translation.Translation != nil {
			pairText = buildExerciseAnswerPairText(
				row.Translation.Original.Word,
				row.Translation.Translation.Word,
				row.Translation.Original.Language,
				row.Translation.Translation.Language,
				t,
			)
		}

		lines = append(lines, emoji+" "+pairText)
	}

	lines = append(lines, "", fmt.Sprintf(t.MatchSummaryKnowledgeFormat, matchedCount, total))

	return strings.Join(lines, "\n")
}

func buildExerciseAnswerPairText(originalWord string, translationWord string, originalLanguage enums.Language, translationLanguage enums.Language, t BotTexts) string {
	return fmt.Sprintf(
		t.ExerciseAnswerPairFormat,
		originalLanguage.Flag(),
		escapeTelegramMarkdown(originalWord),
		escapeTelegramMarkdown(translationWord),
		translationLanguage.Flag(),
	)
}
