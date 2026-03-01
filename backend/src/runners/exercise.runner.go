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

	exercises, err := services.GetDuePendingExercises(now)
	if err != nil {
		logger.L().Errorw("failed to fetch due pending exercises", "error", err)
		return
	}

	for _, exercise := range exercises {
		if !isSupportedBasicExerciseType(exercise.ExerciseType) {
			continue
		}

		questionText := telegram.BuildBasicExerciseQuestion(
			exercise.OriginalWord,
			exercise.TranslationWord,
			exercise.OriginalLanguage,
			exercise.TranslationLanguage,
			exercise.ExerciseType,
		)

		messageID, err := telegram.SendExerciseMessage(exercise.TelegramID, questionText, exercise.ExerciseID)
		if err != nil {
			logger.L().Warnw("failed to send scheduled exercise", "error", err, "exercise_id", exercise.ExerciseID, "telegram_id", exercise.TelegramID)
			continue
		}

		if messageID == nil {
			continue
		}

		if err := services.StartTelegramExercise(exercise.ExerciseID, *messageID); err != nil {
			logger.L().Warnw("failed to mark exercise in progress", "error", err, "exercise_id", exercise.ExerciseID)
		}
	}
}

func isSupportedBasicExerciseType(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeBasicDirect, enums.ExerciseTypeBasicReversed:
		return true
	default:
		return false
	}
}
