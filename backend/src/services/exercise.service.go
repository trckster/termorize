package services

import (
	"errors"
	"math/rand"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/models"
	"time"

	"gorm.io/gorm"
)

func GenerateDailyExercises() error {
	users, err := GetUsersWithEnabledDailyQuestions()
	if err != nil {
		return err
	}

	for _, user := range users {
		GenerateExercises(user)
	}

	return nil
}

func GenerateExercises(user models.User) {
	location, _ := time.LoadLocation(user.Settings.TimeZone)

	now := time.Now().In(location)
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	nextMidnight := midnight.AddDate(0, 0, 1)

	totalMinutes := CountTotalMinutesInSchedule(user.Settings.Telegram.DailyQuestionsSchedule)

	for range user.Settings.Telegram.DailyQuestionsCount {
		midnightOffset := rand.Intn(totalMinutes)

		realOffsetInMinutes := MapOffsetOnSchedule(user.Settings.Telegram.DailyQuestionsSchedule, midnightOffset)

		exerciseScheduleTime := nextMidnight.Add(time.Duration(realOffsetInMinutes) * time.Minute).UTC()

		if err := generateExercise(user.ID, exerciseScheduleTime); err != nil {
			logger.L().Errorw("failed to generate exercise", "user_id", user.ID, "scheduled_for", exerciseScheduleTime, "error", err)
		}
	}
}

func generateExercise(userID uint, when time.Time) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var vocabulary models.Vocabulary
		if err := tx.
			Model(&models.Vocabulary{}).
			Select("id").
			Where("user_id = ?", userID).
			Where("mastered_at IS NULL").
			Where(`EXISTS (
				SELECT 1
				FROM jsonb_array_elements(progress) AS p
				WHERE p->>'type' = ? AND (p->>'knowledge')::int < ?
			)`, enums.KnowledgeTypeTranslation, 100).
			Order("RANDOM()").
			Take(&vocabulary).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("no eligible vocabulary for exercise")
			}

			return err
		}

		exercise := models.Exercise{
			Type:         enums.ExerciseTypeBasic,
			Status:       enums.ExerciseStatusPending,
			UserID:       userID,
			ScheduledFor: &when,
		}

		if err := tx.Create(&exercise).Error; err != nil {
			return err
		}

		if err := tx.Table("vocabulary_exercises").Create(map[string]any{
			"exercise_id":   exercise.ID,
			"vocabulary_id": vocabulary.ID,
		}).Error; err != nil {
			return err
		}

		return nil
	})
}
