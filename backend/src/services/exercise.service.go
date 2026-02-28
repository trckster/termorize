package services

import (
	"math/rand"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PendingExercise struct {
	ExerciseID      uuid.UUID          `gorm:"column:exercise_id"`
	ExerciseType    enums.ExerciseType `gorm:"column:exercise_type"`
	UserID          uint               `gorm:"column:user_id"`
	TelegramID      int64              `gorm:"column:telegram_id"`
	OriginalWord    string             `gorm:"column:original_word"`
	TranslationWord string             `gorm:"column:translation_word"`
}

type ExerciseWords struct {
	OriginalWord    string `gorm:"column:original_word"`
	TranslationWord string `gorm:"column:translation_word"`
}

type TelegramMessageExercise struct {
	ExerciseID      uuid.UUID            `gorm:"column:exercise_id"`
	Status          enums.ExerciseStatus `gorm:"column:status"`
	OriginalWord    string               `gorm:"column:original_word"`
	TranslationWord string               `gorm:"column:translation_word"`
}

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
	requestedExercisesCount := user.Settings.Telegram.DailyQuestionsCount

	if requestedExercisesCount <= 0 {
		return
	}

	vocabularyIDs, err := getEligibleVocabularyIDs(user.ID, requestedExercisesCount)
	if err != nil {
		logger.L().Errorw("failed to select vocabulary for exercises", "user_id", user.ID, "error", err)
		return
	}

	for _, vocabularyID := range vocabularyIDs {
		midnightOffset := rand.Intn(totalMinutes)

		realOffsetInMinutes := MapOffsetOnSchedule(user.Settings.Telegram.DailyQuestionsSchedule, midnightOffset)

		exerciseScheduleTime := nextMidnight.Add(time.Duration(realOffsetInMinutes) * time.Minute).UTC()

		if err := generateExercise(user.ID, vocabularyID, exerciseScheduleTime); err != nil {
			logger.L().Errorw("failed to generate exercise", "user_id", user.ID, "scheduled_for", exerciseScheduleTime, "error", err)
		}
	}
}

func getEligibleVocabularyIDs(userID uint, limit uint) ([]uuid.UUID, error) {
	limitAsInt := int(limit)
	vocabularyIDs := make([]uuid.UUID, 0, limitAsInt)

	err := db.DB.
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
		Limit(limitAsInt).
		Pluck("id", &vocabularyIDs).Error

	if err != nil {
		return nil, err
	}

	return vocabularyIDs, nil
}

func generateExercise(userID uint, vocabularyID uuid.UUID, when time.Time) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
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
			"vocabulary_id": vocabularyID,
		}).Error; err != nil {
			return err
		}

		return nil
	})
}

func GetDuePendingExercises(now time.Time) ([]PendingExercise, error) {
	var exercises []PendingExercise

	err := db.DB.Raw(`
		SELECT
			e.id AS exercise_id,
			e.type AS exercise_type,
			e.user_id AS user_id,
			u.telegram_id AS telegram_id,
			original.word AS original_word,
			translation.word AS translation_word
		FROM exercises AS e
		JOIN users AS u ON u.id = e.user_id
		JOIN vocabulary_exercises AS ve ON ve.exercise_id = e.id
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translation ON translation.id = t.translation_id
		WHERE e.status = ?
			AND e.type = ?
			AND e.scheduled_for <= ?
			AND u.settings->'telegram'->'bot_enabled' = ?
		ORDER BY e.scheduled_for ASC, e.created_at ASC
	`, enums.ExerciseStatusPending, enums.ExerciseTypeBasic, now, true).Scan(&exercises).Error

	if err != nil {
		return nil, err
	}

	return exercises, nil
}

func GetExerciseByTelegramMessage(telegramMessageID int64, telegramID int64) (*TelegramMessageExercise, error) {
	var exercise TelegramMessageExercise

	err := db.DB.Raw(`
		SELECT
			e.id AS exercise_id,
			e.status AS status,
			original.word AS original_word,
			translation.word AS translation_word
		FROM exercises AS e
		JOIN users AS u ON u.id = e.user_id
		JOIN vocabulary_exercises AS ve ON ve.exercise_id = e.id
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translation ON translation.id = t.translation_id
		WHERE e.telegram_message_id = ?
			AND u.telegram_id = ?
		LIMIT 1
	`, telegramMessageID, telegramID).Scan(&exercise).Error

	if err != nil {
		return nil, err
	}

	if exercise.ExerciseID == uuid.Nil {
		return nil, nil
	}

	return &exercise, nil
}

func StartTelegramExercise(exerciseID uuid.UUID, telegramMessageID int64) error {
	return db.DB.Model(&models.Exercise{}).
		Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusPending).
		Updates(map[string]any{
			"status":              enums.ExerciseStatusInProgress,
			"telegram_message_id": telegramMessageID,
			"started_at":          time.Now().UTC(),
		}).Error
}

func CompleteExercise(exerciseID uuid.UUID) error {
	return db.DB.Model(&models.Exercise{}).
		Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
		Updates(map[string]any{
			"status":      enums.ExerciseStatusCompleted,
			"finished_at": time.Now().UTC(),
		}).Error
}

func FailExercise(exerciseID uuid.UUID) (bool, error) {
	result := db.DB.Model(&models.Exercise{}).
		Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
		Updates(map[string]any{
			"status":      enums.ExerciseStatusFailed,
			"finished_at": time.Now().UTC(),
		})

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func GetExerciseWordsByTelegram(exerciseID uuid.UUID, telegramID int64) (*ExerciseWords, error) {
	var words ExerciseWords

	err := db.DB.Raw(`
		SELECT
			original.word AS original_word,
			translation.word AS translation_word
		FROM exercises AS e
		JOIN users AS u ON u.id = e.user_id
		JOIN vocabulary_exercises AS ve ON ve.exercise_id = e.id
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translation ON translation.id = t.translation_id
		WHERE e.id = ?
			AND u.telegram_id = ?
		LIMIT 1
	`, exerciseID, telegramID).Scan(&words).Error

	if err != nil {
		return nil, err
	}

	if words.OriginalWord == "" && words.TranslationWord == "" {
		return nil, nil
	}

	return &words, nil
}
