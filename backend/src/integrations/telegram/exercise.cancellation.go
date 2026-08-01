package telegram

import (
	"termorize/src/logger"
	"termorize/src/services"
)

func init() {
	services.RegisterCancelledTelegramExercisesNotifier(notifyCancelledTelegramExercises)
}

func notifyCancelledTelegramExercises(exercises []services.CancelledTelegramExercise) {
	for _, exercise := range exercises {
		texts := GetBotTexts(exercise.SystemLanguage)
		if err := EditMessageTextWithInlineKeyboard(
			exercise.TelegramID,
			exercise.MessageID,
			texts.ExerciseCancelledVocabularyDeleted,
			[][]inlineKeyboardButton{},
		); err != nil {
			logger.L().Warnw(
				"failed to update cancelled exercise message",
				"error", err,
				"exercise_id", exercise.ExerciseID,
				"telegram_id", exercise.TelegramID,
				"message_id", exercise.MessageID,
			)
		}
	}
}

func sendIgnoredExerciseMessage(chatID, messageID, responseChatID int64, exercise *services.TelegramMessageExercise, texts BotTexts) error {
	if exercise.ResultReason == services.ExerciseVocabularyResultReasonDeletedVocabulary {
		removeDeletedExerciseKeyboard(chatID, messageID)
	}

	return SendMessage(responseChatID, ignoredExerciseText(exercise, texts))
}

func sendDeletedVocabularyMessage(chatID, messageID, responseChatID int64, text string) error {
	removeDeletedExerciseKeyboard(chatID, messageID)
	return SendMessage(responseChatID, text)
}

func removeDeletedExerciseKeyboard(chatID, messageID int64) {
	if err := removeMessageInlineKeyboard(chatID, messageID); err != nil {
		logger.L().Warnw("failed to remove inline keyboard", "error", err, "chat_id", chatID, "message_id", messageID)
	}
}
