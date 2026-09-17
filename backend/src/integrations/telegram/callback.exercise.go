package telegram

import (
	"errors"
	"termorize/src/enums"
	"termorize/src/services"

	"github.com/google/uuid"
)

func parseExerciseIDKPayload(payload []string) (uuid.UUID, bool) {
	if len(payload) != 2 || payload[0] != exerciseActionIDK {
		return uuid.Nil, false
	}

	exerciseID, err := uuid.Parse(payload[1])
	if err != nil {
		return uuid.Nil, false
	}

	return exerciseID, true
}

func parseExerciseAnswerPayload(payload []string) (uuid.UUID, uuid.UUID, bool) {
	if len(payload) != 3 || payload[0] != exerciseActionAnswer {
		return uuid.Nil, uuid.Nil, false
	}

	exerciseID, err := parseCallbackUUID(payload[1])
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}

	selectedVocabularyID, err := parseCallbackUUID(payload[2])
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}

	return exerciseID, selectedVocabularyID, true
}

func handleExerciseCallback(callback *callbackQuery, payload []string) error {
	if callback.Message == nil {
		return nil
	}

	t := getBotTextsForTelegramID(callback.From.ID)
	if handled, err := handleAudioLanguageCallback(callback, payload, t); handled {
		return err
	}
	if handled, err := handleDescriptionLanguageCallback(callback, payload, t); handled {
		return err
	}

	if len(payload) > 0 && payload[0] == exerciseActionMatchNoop {
		return nil
	}
	if len(payload) > 0 && payload[0] == exerciseActionCharacterNoop {
		return nil
	}
	if len(payload) >= 2 && payload[0] == exerciseActionMatchTap {
		return handleMatchTap(callback, payload, t)
	}
	if len(payload) >= 2 && payload[0] == exerciseActionCharacterTap {
		return handleCharacterTap(callback, payload, t)
	}
	if len(payload) >= 2 && payload[0] == exerciseActionCharacterBackspace {
		return handleCharacterBackspace(callback, payload, t)
	}

	exerciseID, selectedVocabularyID, hasAnswer := parseExerciseAnswerPayload(payload)
	if !hasAnswer {
		var ok bool
		exerciseID, ok = parseExerciseIDKPayload(payload)
		if !ok {
			return nil
		}
	}

	exercise, err := services.GetExerciseByTelegramMessage(callback.Message.MessageID, callback.From.ID)
	if err != nil {
		return err
	}
	if exercise == nil && !hasAnswer {
		exercise, err = recoverPendingCharacterExerciseFromCallback(callback, exerciseID)
		if err != nil {
			return err
		}
	}

	if exercise == nil || exercise.ExerciseID != exerciseID {
		return nil
	}
	if exercise.Deleted {
		return SendMessage(callback.From.ID, cancelledExerciseText(exercise.ExerciseType, t))
	}

	switch exercise.Status {
	case enums.ExerciseStatusIgnored:
		return sendIgnoredExerciseMessage(callback.Message.Chat.ID, callback.Message.MessageID, callback.From.ID, exercise, t)
	case enums.ExerciseStatusCompleted:
		if editsExerciseAnswer(exercise.ExerciseType) {
			return nil
		}
		return SendMessage(callback.From.ID, t.ExerciseCompleted)
	case enums.ExerciseStatusFailed:
		if editsExerciseAnswer(exercise.ExerciseType) {
			return nil
		}
		return SendMessage(callback.From.ID, t.ExerciseFailed)
	}

	if len(exercise.Vocabulary) == 0 || exercise.Vocabulary[0].Translation == nil {
		_ = services.MarkExerciseVocabularyResultWithoutProgress(exercise.ExerciseID, services.ExerciseVocabularyResultIgnored, services.ExerciseVocabularyResultReasonDeletedVocabulary)
		_ = services.IgnoreExercise(exercise.ExerciseID)
		return sendDeletedVocabularyMessage(callback.Message.Chat.ID, callback.Message.MessageID, callback.From.ID, t.ExerciseVocabularyDeleted)
	}

	if !hasAnswer && exercise.ExerciseType != enums.ExerciseTypeBasicDirect && exercise.ExerciseType != enums.ExerciseTypeBasicReversed &&
		exercise.ExerciseType != enums.ExerciseTypeAudioDirect && exercise.ExerciseType != enums.ExerciseTypeAudioReversed &&
		exercise.ExerciseType != enums.ExerciseTypeDescriptionDirect && exercise.ExerciseType != enums.ExerciseTypeDescriptionReversed &&
		exercise.ExerciseType != enums.ExerciseTypeCharactersDirect && exercise.ExerciseType != enums.ExerciseTypeCharactersReversed {
		return nil
	}

	if hasAnswer {
		result, err := services.VerifyExerciseChoice(exerciseID, exercise.UserID, selectedVocabularyID)
		if err != nil {
			if errors.Is(err, services.ErrExerciseNotInProgress) {
				return nil
			}

			if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
				return sendDeletedVocabularyMessage(callback.Message.Chat.ID, callback.Message.MessageID, callback.From.ID, t.ExerciseVocabularyDeleted)
			}

			return err
		}

		return sendExerciseAnswerResult(callback.Message.Chat.ID, callback.Message.MessageID, exercise, result, t)
	}

	updated, translationKnowledge, err := services.FinishExercise(
		exerciseID,
		enums.ExerciseStatusFailed,
		services.ExerciseVocabularyResultIgnored,
		services.ExerciseVocabularyResultReasonSkipped,
		services.ExerciseWrongProgressDeltaForType(exercise.ExerciseType),
	)
	if err != nil {
		return err
	}

	if !updated {
		return nil
	}

	words, err := services.GetExerciseWordsByTelegram(exerciseID, callback.From.ID)
	if err != nil {
		return err
	}

	if words == nil {
		return nil
	}

	answerText := buildExerciseIDKResultText(
		words.OriginalWord,
		words.TranslationWord,
		words.OriginalLanguage,
		words.TranslationLanguage,
		translationKnowledge,
		t,
	)
	keyboard := &inlineKeyboardMarkup{InlineKeyboard: [][]inlineKeyboardButton{}}
	if exercise.ExerciseType == enums.ExerciseTypeAudioDirect || exercise.ExerciseType == enums.ExerciseTypeAudioReversed {
		return editMessageCaptionTolerant(editMessageCaptionRequest{
			ChatID:      callback.Message.Chat.ID,
			MessageID:   callback.Message.MessageID,
			Caption:     answerText,
			ParseMode:   telegramParseModeMarkdown,
			ReplyMarkup: keyboard,
		})
	}
	return editMessageTextTolerant(editMessageTextRequest{
		ChatID:      callback.Message.Chat.ID,
		MessageID:   callback.Message.MessageID,
		Text:        answerText,
		ParseMode:   telegramParseModeMarkdown,
		ReplyMarkup: keyboard,
	})
}

func ignoredExerciseText(exercise *services.TelegramMessageExercise, t BotTexts) string {
	if exercise.ResultReason == services.ExerciseVocabularyResultReasonDeletedVocabulary {
		return t.ExerciseVocabularyDeleted
	}
	return t.ExerciseOutdated
}
