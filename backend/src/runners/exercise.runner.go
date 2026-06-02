package runners

import (
	"sync"
	"termorize/src/enums"
	"termorize/src/integrations/telegram"
	"termorize/src/logger"
	"termorize/src/monitoring"
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
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.L().Errorw("exercise runner panicked", "panic", recovered)
			monitoring.Recover(nil, recovered)
		}
	}()

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
		monitoring.CaptureException(nil, err)
	}

	if err := processDueExerciseReminders(now); err != nil {
		logger.L().Errorw("failed to process exercise reminders", "error", err)
		monitoring.CaptureException(nil, err)
	}

	if err := services.IgnoreDuePendingExercisesWithoutActiveVocabulary(now); err != nil {
		logger.L().Errorw("failed to ignore invalid pending exercises", "error", err)
		monitoring.CaptureException(nil, err)
	}

	exercises, err := services.GetDuePendingExercises(now)
	if err != nil {
		logger.L().Errorw("failed to fetch due pending exercises", "error", err)
		monitoring.CaptureException(nil, err)
		return
	}

	for _, exercise := range exercises {
		if !isSupportedExerciseType(exercise.ExerciseType) {
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

		var (
			messageID *int64
			err       error
		)

		if isChoiceExerciseType(exercise.ExerciseType) {
			options, loadErr := services.GetExerciseAnswerOptions(exercise.ExerciseID, exercise.ExerciseType)
			if loadErr != nil {
				logger.L().Warnw("failed to load exercise options", "error", loadErr, "exercise_id", exercise.ExerciseID)
				continue
			}
			if len(options) != 4 {
				logger.L().Warnw("ignoring choice exercise with incomplete options", "exercise_id", exercise.ExerciseID, "options_count", len(options))
				if ignoreErr := services.IgnoreExercise(exercise.ExerciseID); ignoreErr != nil {
					logger.L().Warnw("failed to ignore invalid exercise", "error", ignoreErr, "exercise_id", exercise.ExerciseID)
				}
				continue
			}

			messageID, err = telegram.SendChoiceExerciseMessage(exercise.TelegramID, questionText, exercise.ExerciseID, options, texts)
		} else {
			messageID, err = telegram.SendBasicExerciseMessage(exercise.TelegramID, questionText, exercise.ExerciseID, texts)
		}

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

func isSupportedExerciseType(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeBasicDirect, enums.ExerciseTypeBasicReversed, enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed:
		return true
	default:
		return false
	}
}

func isChoiceExerciseType(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed:
		return true
	default:
		return false
	}
}
