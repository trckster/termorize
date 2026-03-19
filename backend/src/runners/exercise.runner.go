package runners

import (
	"sync"
	"termorize/src/enums"
	"termorize/src/integrations/telegram"
	"termorize/src/logger"
	"termorize/src/services"
	"time"
)

var exerciseRunnerOnce sync.Once

func StartExerciseRunner() {
	exerciseRunnerOnce.Do(func() {
		go runExerciseRunner()
	})
}

func runExerciseRunner() {
	processDueExercises()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		processDueExercises()
	}
}

func processDueExercises() {
	now := time.Now().UTC()

	if err := services.ExpireStaleInProgressExercises(now); err != nil {
		logger.L().Errorw("failed to expire stale in-progress exercises", "error", err)
	}

	if err := processDueExerciseReminders(now); err != nil {
		logger.L().Errorw("failed to process exercise reminders", "error", err)
	}

	exercises, err := services.GetDuePendingExercises(now)
	if err != nil {
		logger.L().Errorw("failed to fetch due pending exercises", "error", err)
		return
	}

	for _, exercise := range exercises {
		if !isSupportedBasicExerciseType(exercise.ExerciseType) {
			continue
		}

		texts := telegram.GetBotTexts(exercise.SystemLanguage)
		questionText := telegram.BuildBasicExerciseQuestion(
			exercise.OriginalWord,
			exercise.TranslationWord,
			exercise.OriginalLanguage,
			exercise.TranslationLanguage,
			exercise.ExerciseType,
			texts,
		)

		messageID, err := telegram.SendExerciseMessage(exercise.TelegramID, questionText, exercise.ExerciseID, texts)
		if err != nil {
			logger.L().Warnw("failed to send scheduled exercise", "error", err, "exercise_id", exercise.ExerciseID, "telegram_id", exercise.TelegramID)
			continue
		}

		if messageID == nil {
			continue
		}

		logger.L().Infow("exercise sent", "username", exercise.Username)

		if err := services.StartTelegramExercise(exercise.ExerciseID, *messageID); err != nil {
			logger.L().Warnw("failed to mark exercise in progress", "error", err, "exercise_id", exercise.ExerciseID)
		}
	}
}

func processDueExerciseReminders(now time.Time) error {
	reminders, err := services.GetDueExerciseReminders(now)
	if err != nil {
		return err
	}

	for _, reminder := range reminders {
		texts := telegram.GetBotTexts(reminder.SystemLanguage)
		if err := telegram.SendReplyMessage(
			reminder.TelegramID,
			telegram.BuildExerciseReminderText(texts),
			reminder.TelegramMessageID,
		); err != nil {
			logger.L().Warnw("failed to send exercise reminder", "error", err, "exercise_id", reminder.ExerciseID, "telegram_id", reminder.TelegramID)
			continue
		}

		updated, err := services.MarkExerciseReminderSent(reminder.ExerciseID, now)
		if err != nil {
			logger.L().Warnw("failed to mark exercise reminder as sent", "error", err, "exercise_id", reminder.ExerciseID)
			continue
		}

		if !updated {
			continue
		}

		logger.L().Infow("exercise reminder sent", "exercise_id", reminder.ExerciseID, "telegram_id", reminder.TelegramID)
	}

	return nil
}

func isSupportedBasicExerciseType(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeBasicDirect, enums.ExerciseTypeBasicReversed:
		return true
	default:
		return false
	}
}
