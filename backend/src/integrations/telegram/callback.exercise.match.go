package telegram

import (
	"errors"
	"strconv"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/services"

	"github.com/google/uuid"
)

func parseExerciseMatchPayload(payload []string) (uuid.UUID, int, bool) {
	if len(payload) != 3 || payload[0] != exerciseActionMatchTap {
		return uuid.Nil, 0, false
	}

	exerciseID, err := parseCallbackUUID(payload[1])
	if err != nil {
		return uuid.Nil, 0, false
	}

	tappedIndex, err := strconv.Atoi(payload[2])
	if err != nil || tappedIndex < 0 || tappedIndex > 9 {
		return uuid.Nil, 0, false
	}

	return exerciseID, tappedIndex, true
}

func handleMatchTap(callback *callbackQuery, payload []string, t BotTexts) error {
	if callback.Message == nil {
		return nil
	}

	callbackAnswerAttempted := false
	defer func() {
		if callback.ID == "" || callbackAnswerAttempted {
			return
		}
		callbackAnswerAttempted = true
		logCallbackAnswerError(answerTelegramCallbackQuery(callback.ID), callback.ID)
	}()

	exerciseID, tappedIdx, ok := parseExerciseMatchPayload(payload)
	if !ok {
		return nil
	}

	exercise, err := services.GetExerciseByTelegramMessage(callback.Message.MessageID, callback.From.ID)
	if err != nil {
		return err
	}

	if exercise == nil {
		exercise, err = recoverPendingMatchExerciseFromCallback(callback, exerciseID)
		if err != nil {
			return err
		}
	}

	if exercise == nil || exercise.ExerciseID != exerciseID {
		return nil
	}

	switch exercise.Status {
	case enums.ExerciseStatusIgnored:
		return SendMessage(callback.From.ID, t.ExerciseOutdated)
	case enums.ExerciseStatusCompleted, enums.ExerciseStatusFailed:
		result, resultErr := services.GetCompletedMatchPairsResult(exercise.ExerciseID, exercise.UserID)
		if resultErr != nil {
			return resultErr
		}
		return EditMatchBoardMessage(callback.Message.Chat.ID, callback.Message.MessageID, buildMatchResultSummaryText(result, t), [][]inlineKeyboardButton{})
	}

	board, wasWrong, finished, finalizeAttempts, err := services.ApplyMatchTap(exercise.ExerciseID, exercise.UserID, tappedIdx)
	if err != nil {
		if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
			if removeErr := removeMessageInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID); removeErr != nil {
				logger.L().Warnw("failed to remove inline keyboard", "error", removeErr, "chat_id", callback.Message.Chat.ID, "message_id", callback.Message.MessageID)
			}
			return SendMessage(callback.From.ID, t.ExerciseVocabularyDeleted)
		}
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil
		}

		return err
	}

	if !finished {
		if wasWrong {
			callbackAnswerAttempted = true
			logCallbackAnswerError(answerTelegramCallbackQueryWithText(callback.ID, t.MatchNotAMatchToast), callback.ID)
		}

		return EditMatchBoardMessage(callback.Message.Chat.ID, callback.Message.MessageID, buildMatchBoardText(board, t), buildMatchKeyboard(exercise.ExerciseID, board))
	}

	if len(finalizeAttempts) == 0 {
		return nil
	}

	result, err := services.CompleteMatchPairsExercise(exercise.ExerciseID, exercise.UserID, finalizeAttempts)
	if err != nil {
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil
		}
		if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
			if removeErr := removeMessageInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID); removeErr != nil {
				logger.L().Warnw("failed to remove inline keyboard", "error", removeErr, "chat_id", callback.Message.Chat.ID, "message_id", callback.Message.MessageID)
			}
			return SendMessage(callback.From.ID, t.ExerciseVocabularyDeleted)
		}

		return err
	}

	return EditMatchBoardMessage(callback.Message.Chat.ID, callback.Message.MessageID, buildMatchResultSummaryText(result, t), [][]inlineKeyboardButton{})
}

func recoverPendingMatchExerciseFromCallback(callback *callbackQuery, exerciseID uuid.UUID) (*services.TelegramMessageExercise, error) {
	exercise, err := services.GetExerciseByTelegramExerciseID(exerciseID, callback.From.ID)
	if err != nil {
		return nil, err
	}
	if exercise == nil || exercise.Status != enums.ExerciseStatusPending || exercise.ExerciseType != enums.ExerciseTypeMatchPairs {
		return nil, nil
	}

	order, ok := extractMatchOrderFromReplyMarkup(callback.Message.ReplyMarkup, exerciseID)
	if !ok {
		return nil, nil
	}

	if err := services.StartMatchExercise(exerciseID, callback.Message.MessageID, order); err != nil {
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil, nil
		}
		return nil, err
	}

	return services.GetExerciseByTelegramMessage(callback.Message.MessageID, callback.From.ID)
}

func extractMatchOrderFromReplyMarkup(markup *inlineKeyboardMarkup, exerciseID uuid.UUID) ([]int, bool) {
	if markup == nil {
		return nil, false
	}

	expectedCards := services.MatchPairsVocabularyCount * 2
	order := make([]int, 0, expectedCards)
	seen := make(map[int]bool, expectedCards)

	for _, row := range markup.InlineKeyboard {
		for _, button := range row {
			handlerType, payload, ok := parseCallbackData(button.CallbackData)
			if !ok || handlerType != callbackTypeExercise {
				return nil, false
			}

			buttonExerciseID, canonical, ok := parseExerciseMatchPayload(payload)
			if !ok || buttonExerciseID != exerciseID || seen[canonical] {
				return nil, false
			}

			seen[canonical] = true
			order = append(order, canonical)
		}
	}

	if len(order) != expectedCards {
		return nil, false
	}

	for i := 0; i < expectedCards; i++ {
		if !seen[i] {
			return nil, false
		}
	}

	return order, true
}
