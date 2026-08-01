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
