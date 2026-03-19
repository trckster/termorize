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
	"gorm.io/gorm/clause"
)

const (
	exerciseProgressStep     = 20
	exerciseReminderPeriod   = 24 * time.Hour
	exerciseExpirationPeriod = 7 * 24 * time.Hour
)

type PendingExercise struct {
	ExerciseID          uuid.UUID          `gorm:"column:exercise_id"`
	ExerciseType        enums.ExerciseType `gorm:"column:exercise_type"`
	UserID              uint               `gorm:"column:user_id"`
	Username            string             `gorm:"column:username"`
	TelegramID          int64              `gorm:"column:telegram_id"`
	OriginalWord        string             `gorm:"column:original_word"`
	OriginalLanguage    enums.Language     `gorm:"column:original_language"`
	TranslationWord     string             `gorm:"column:translation_word"`
	TranslationLanguage enums.Language     `gorm:"column:translation_language"`
	SystemLanguage      enums.Language     `gorm:"column:system_language"`
}

type ExerciseWords struct {
	ExerciseType        enums.ExerciseType `gorm:"column:exercise_type"`
	OriginalWord        string             `gorm:"column:original_word"`
	OriginalLanguage    enums.Language     `gorm:"column:original_language"`
	TranslationWord     string             `gorm:"column:translation_word"`
	TranslationLanguage enums.Language     `gorm:"column:translation_language"`
}

type PendingExerciseReminder struct {
	ExerciseID        uuid.UUID      `gorm:"column:exercise_id"`
	TelegramID        int64          `gorm:"column:telegram_id"`
	TelegramMessageID int64          `gorm:"column:telegram_message_id"`
	SystemLanguage    enums.Language `gorm:"column:system_language"`
}

type TelegramMessageExercise struct {
	ExerciseID          uuid.UUID            `gorm:"column:exercise_id"`
	ExerciseType        enums.ExerciseType   `gorm:"column:exercise_type"`
	Status              enums.ExerciseStatus `gorm:"column:status"`
	OriginalWord        string               `gorm:"column:original_word"`
	OriginalLanguage    enums.Language       `gorm:"column:original_language"`
	TranslationWord     string               `gorm:"column:translation_word"`
	TranslationLanguage enums.Language       `gorm:"column:translation_language"`
}

type ExerciseStatistics struct {
	InProgress int64 `json:"in_progress" gorm:"column:in_progress"`
	Done       int64 `json:"done" gorm:"column:done"`
	Failed     int64 `json:"failed" gorm:"column:failed"`
	Ignored    int64 `json:"ignored" gorm:"column:ignored"`
}

func GenerateDailyExercises() error {
	users, err := GetUsersWithEnabledDailyQuestions()
	if err != nil {
		return err
	}

	targetDate := time.Now().UTC().AddDate(0, 0, 1)
	targetDateString := targetDate.Format("2006-01-02")
	generatedExercisesCount := 0
	usersWithGeneratedExercisesCount := 0

	for _, user := range users {
		generatedCount := GenerateExercises(user, targetDate)
		if generatedCount == 0 {
			continue
		}

		generatedExercisesCount += generatedCount
		usersWithGeneratedExercisesCount++
	}

	logger.L().Infow("daily exercises generated", "date", targetDateString, "exercise_count", generatedExercisesCount, "user_count", usersWithGeneratedExercisesCount)

	return nil
}

func GenerateExercises(user models.User, targetDate time.Time) int {
	location, _ := time.LoadLocation(user.Settings.TimeZone)
	targetMidnight := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, location)

	totalMinutes := CountTotalMinutesInSchedule(user.Settings.Telegram.DailyQuestionsSchedule)
	requestedExercisesCount := user.Settings.Telegram.DailyQuestionsCount

	if requestedExercisesCount <= 0 {
		return 0
	}

	vocabularyIDs, err := getEligibleVocabularyIDs(user.ID, requestedExercisesCount)
	if err != nil {
		logger.L().Errorw("failed to select vocabulary for exercises", "user_id", user.ID, "error", err)
		return 0
	}

	generatedCount := 0

	for _, vocabularyID := range vocabularyIDs {
		midnightOffset := rand.Intn(totalMinutes)

		realOffsetInMinutes := MapOffsetOnSchedule(user.Settings.Telegram.DailyQuestionsSchedule, midnightOffset)

		exerciseScheduleTime := targetMidnight.Add(time.Duration(realOffsetInMinutes) * time.Minute).UTC()

		if err := generateExercise(user.ID, vocabularyID, exerciseScheduleTime); err != nil {
			logger.L().Errorw("failed to generate exercise", "user_id", user.ID, "scheduled_for", exerciseScheduleTime, "error", err)
			continue
		}

		generatedCount++
	}

	return generatedCount
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
		exerciseType := enums.ExerciseTypeBasicDirect
		if rand.Intn(2) == 0 {
			exerciseType = enums.ExerciseTypeBasicReversed
		}

		exercise := models.Exercise{
			Type:         exerciseType,
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
			u.username AS username,
			u.telegram_id AS telegram_id,
			original.word AS original_word,
			original.language AS original_language,
			translation.word AS translation_word,
			translation.language AS translation_language,
			u.settings->>'system_language' AS system_language
		FROM exercises AS e
		JOIN users AS u ON u.id = e.user_id
		JOIN vocabulary_exercises AS ve ON ve.exercise_id = e.id
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translation ON translation.id = t.translation_id
		WHERE e.status = ?
			AND e.type IN (?, ?)
			AND e.scheduled_for <= ?
			AND u.settings->'telegram'->'bot_enabled' = ?
		ORDER BY e.scheduled_for ASC, e.created_at ASC
	`, enums.ExerciseStatusPending, enums.ExerciseTypeBasicDirect, enums.ExerciseTypeBasicReversed, now, true).Scan(&exercises).Error

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
			e.type AS exercise_type,
			e.status AS status,
			original.word AS original_word,
			original.language AS original_language,
			translation.word AS translation_word,
			translation.language AS translation_language
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

func DeletePendingExercisesByUserID(tx *gorm.DB, userID uint) error {
	return tx.Where("user_id = ? AND status = ?", userID, enums.ExerciseStatusPending).
		Delete(&models.Exercise{}).Error
}

func GetDueExerciseReminders(now time.Time) ([]PendingExerciseReminder, error) {
	var reminders []PendingExerciseReminder
	remindBefore := now.Add(-exerciseReminderPeriod)

	err := db.DB.Raw(`
		SELECT
			e.id AS exercise_id,
			u.telegram_id AS telegram_id,
			e.telegram_message_id AS telegram_message_id,
			u.settings->>'system_language' AS system_language
		FROM exercises AS e
		JOIN users AS u ON u.id = e.user_id
		WHERE e.status = ?
			AND e.telegram_message_id IS NOT NULL
			AND e.started_at IS NOT NULL
			AND e.started_at <= ?
			AND e.reminder_sent_at IS NULL
			AND u.settings->'telegram'->'bot_enabled' = ?
		ORDER BY e.started_at ASC
	`, enums.ExerciseStatusInProgress, remindBefore, true).Scan(&reminders).Error

	if err != nil {
		return nil, err
	}

	return reminders, nil
}

func MarkExerciseReminderSent(exerciseID uuid.UUID, reminderSentAt time.Time) (bool, error) {
	result := db.DB.Model(&models.Exercise{}).
		Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
		Where("reminder_sent_at IS NULL").
		Update("reminder_sent_at", reminderSentAt)

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func ExpireStaleInProgressExercises(now time.Time) error {
	expiresBefore := now.Add(-exerciseExpirationPeriod)

	return db.DB.Model(&models.Exercise{}).
		Where("status = ?", enums.ExerciseStatusInProgress).
		Where("started_at IS NOT NULL").
		Where("started_at <= ?", expiresBefore).
		Updates(map[string]any{
			"status":      enums.ExerciseStatusIgnored,
			"finished_at": now,
		}).Error
}

func CompleteExercise(exerciseID uuid.UUID) (bool, int, error) {
	updated := false
	translationKnowledge := 0

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Exercise{}).
			Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
			Updates(map[string]any{
				"status":      enums.ExerciseStatusCompleted,
				"finished_at": time.Now().UTC(),
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return nil
		}

		updated = true

		var updateErr error
		translationKnowledge, updateErr = updateVocabularyProgressByExercise(tx, exerciseID, exerciseProgressStep)
		return updateErr
	})

	if err != nil {
		return false, 0, err
	}

	return updated, translationKnowledge, nil
}

func FailExercise(exerciseID uuid.UUID) (bool, int, error) {
	updated := false
	translationKnowledge := 0

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Exercise{}).
			Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
			Updates(map[string]any{
				"status":      enums.ExerciseStatusFailed,
				"finished_at": time.Now().UTC(),
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return nil
		}

		updated = true

		var updateErr error
		translationKnowledge, updateErr = updateVocabularyProgressByExercise(tx, exerciseID, -exerciseProgressStep)
		return updateErr
	})

	if err != nil {
		return false, 0, err
	}

	return updated, translationKnowledge, nil
}

func updateVocabularyProgressByExercise(tx *gorm.DB, exerciseID uuid.UUID, delta int) (int, error) {
	var exerciseLink struct {
		VocabularyID uuid.UUID `gorm:"column:vocabulary_id"`
	}

	if err := tx.Table("vocabulary_exercises").
		Select("vocabulary_id").
		Where("exercise_id = ?", exerciseID).
		Take(&exerciseLink).Error; err != nil {
		return 0, err
	}

	var vocabulary models.Vocabulary
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", exerciseLink.VocabularyID).
		Take(&vocabulary).Error; err != nil {
		return 0, err
	}

	translationKnowledge := 0
	found := false
	for index := range vocabulary.Progress {
		if vocabulary.Progress[index].Type != enums.KnowledgeTypeTranslation {
			continue
		}

		vocabulary.Progress[index].Knowledge = clampProgress(vocabulary.Progress[index].Knowledge + delta)
		translationKnowledge = vocabulary.Progress[index].Knowledge
		found = true
		break
	}

	if !found {
		translationKnowledge = clampProgress(delta)
		vocabulary.Progress = append(vocabulary.Progress, models.ProgressEntry{
			Knowledge: translationKnowledge,
			Type:      enums.KnowledgeTypeTranslation,
		})
	}

	var masteredAt *time.Time
	if translationKnowledge >= 100 {
		if vocabulary.MasteredAt != nil {
			masteredAt = vocabulary.MasteredAt
		} else {
			now := time.Now().UTC()
			masteredAt = &now
		}
	}

	err := tx.Model(&models.Vocabulary{}).
		Where("id = ?", vocabulary.ID).
		Updates(map[string]any{
			"progress":    vocabulary.Progress,
			"mastered_at": masteredAt,
		}).Error
	if err != nil {
		return 0, err
	}

	return translationKnowledge, nil
}

func clampProgress(progress int) int {
	if progress < 0 {
		return 0
	}

	if progress > 100 {
		return 100
	}

	return progress
}

func GetExerciseWordsByTelegram(exerciseID uuid.UUID, telegramID int64) (*ExerciseWords, error) {
	var words ExerciseWords

	err := db.DB.Raw(`
		SELECT
			e.type AS exercise_type,
			original.word AS original_word,
			original.language AS original_language,
			translation.word AS translation_word,
			translation.language AS translation_language
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

func GetExerciseStatistics(userID uint) (*ExerciseStatistics, error) {
	var statistics ExerciseStatistics

	err := db.DB.Raw(`
		SELECT
			COUNT(*) FILTER (WHERE status = ?) AS in_progress,
			COUNT(*) FILTER (WHERE status = ?) AS done,
			COUNT(*) FILTER (WHERE status = ?) AS failed,
			COUNT(*) FILTER (WHERE status = ?) AS ignored
		FROM exercises
		WHERE user_id = ?
	`, enums.ExerciseStatusInProgress, enums.ExerciseStatusCompleted, enums.ExerciseStatusFailed, enums.ExerciseStatusIgnored, userID).Scan(&statistics).Error
	if err != nil {
		return nil, err
	}

	return &statistics, nil
}
