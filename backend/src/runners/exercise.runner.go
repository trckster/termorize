package runners

import (
	"fmt"
	"math/rand"
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
		if exercise.ExerciseType != enums.ExerciseTypeBasic {
			continue
		}

		questionText, questionType := buildBasicExerciseQuestion(exercise.OriginalWord, exercise.TranslationWord)

		messageID, err := telegram.SendExerciseMessage(exercise.TelegramID, questionText, exercise.ExerciseID, questionType)
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

func buildBasicExerciseQuestion(originalWord string, translationWord string) (string, string) {
	if rand.Intn(2) == 0 {
		return fmt.Sprintf("Translate this word: %s", originalWord), "o2t"
	}

	return fmt.Sprintf("What is the original word for: %s", translationWord), "t2o"
}
