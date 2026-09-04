package services

import (
	"errors"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CancelledTelegramExercise struct {
	ExerciseID     uuid.UUID      `gorm:"column:exercise_id"`
	TelegramID     int64          `gorm:"column:telegram_id"`
	MessageID      int64          `gorm:"column:telegram_message_id"`
	SystemLanguage enums.Language `gorm:"column:system_language"`
}

func cancelInProgressExercisesByVocabularyID(tx *gorm.DB, userID uint, vocabularyID uuid.UUID, now time.Time) ([]CancelledTelegramExercise, error) {
	var cancelled []CancelledTelegramExercise
	if err := tx.Raw(`
		SELECT e.id AS exercise_id, u.telegram_id, e.telegram_message_id,
			u.settings->>'system_language' AS system_language
		FROM exercises e
		JOIN users u ON u.id = e.user_id
		WHERE e.user_id = ? AND e.deleted_at IS NULL AND e.status = ? AND e.telegram_message_id IS NOT NULL
			AND (
				(e.type = ? AND EXISTS (
					SELECT 1 FROM vocabulary_exercises ve
					WHERE ve.exercise_id = e.id AND ve.vocabulary_id = ?
				)) OR (e.type <> ? AND EXISTS (
					SELECT 1 FROM vocabulary_exercises ve
					WHERE ve.exercise_id = e.id AND ve.vocabulary_id = ? AND ve.is_correct = true
				))
			)
	`, userID, enums.ExerciseStatusInProgress, enums.ExerciseTypeMatchPairs, vocabularyID, enums.ExerciseTypeMatchPairs, vocabularyID).Scan(&cancelled).Error; err != nil {
		return nil, err
	}

	if err := tx.Exec(`
		UPDATE vocabulary_exercises ve
		SET result = ?, result_reason = ?, answered_at = ?
		WHERE ve.result IS NULL AND ve.is_correct = true AND ve.exercise_id IN (
			SELECT e.id FROM exercises e
				WHERE e.user_id = ? AND e.deleted_at IS NULL AND e.status = ? AND (
				(e.type = ? AND EXISTS (SELECT 1 FROM vocabulary_exercises x WHERE x.exercise_id = e.id AND x.vocabulary_id = ?))
				OR (e.type <> ? AND EXISTS (SELECT 1 FROM vocabulary_exercises x WHERE x.exercise_id = e.id AND x.vocabulary_id = ? AND x.is_correct = true))
			)
		)
	`, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary, now, userID, enums.ExerciseStatusInProgress, enums.ExerciseTypeMatchPairs, vocabularyID, enums.ExerciseTypeMatchPairs, vocabularyID).Error; err != nil {
		return nil, err
	}

	if err := tx.Exec(`
		UPDATE exercises e SET status = ?, finished_at = ?
		WHERE e.user_id = ? AND e.deleted_at IS NULL AND e.status = ? AND (
			(e.type = ? AND EXISTS (SELECT 1 FROM vocabulary_exercises ve WHERE ve.exercise_id = e.id AND ve.vocabulary_id = ?))
			OR (e.type <> ? AND EXISTS (SELECT 1 FROM vocabulary_exercises ve WHERE ve.exercise_id = e.id AND ve.vocabulary_id = ? AND ve.is_correct = true))
		)
	`, enums.ExerciseStatusIgnored, now, userID, enums.ExerciseStatusInProgress, enums.ExerciseTypeMatchPairs, vocabularyID, enums.ExerciseTypeMatchPairs, vocabularyID).Error; err != nil {
		return nil, err
	}

	return cancelled, nil
}

func DeletePendingExercisesByUserID(tx *gorm.DB, userID uint) error {
	return tx.Where("user_id = ? AND status = ?", userID, enums.ExerciseStatusPending).
		Delete(&models.Exercise{}).Error
}

func DeletePendingExercisesByVocabularyID(tx *gorm.DB, userID uint, vocabularyID uuid.UUID) ([]time.Time, error) {
	var scheduledFor []time.Time
	if err := tx.Model(&models.Exercise{}).
		Where("user_id = ? AND status = ? AND scheduled_for IS NOT NULL", userID, enums.ExerciseStatusPending).
		Where("id IN (?)",
			tx.Table("vocabulary_exercises").
				Select("exercise_id").
				Where("vocabulary_id = ?", vocabularyID),
		).
		Pluck("scheduled_for", &scheduledFor).Error; err != nil {
		return nil, err
	}

	if err := tx.
		Where("user_id = ? AND status = ?", userID, enums.ExerciseStatusPending).
		Where("id IN (?)",
			tx.Table("vocabulary_exercises").
				Select("exercise_id").
				Where("vocabulary_id = ?", vocabularyID),
		).
		Delete(&models.Exercise{}).Error; err != nil {
		return nil, err
	}

	return scheduledFor, nil
}

func IgnoreExercise(exerciseID uuid.UUID) error {
	return db.DB.Model(&models.Exercise{}).
		Where("id = ?", exerciseID).
		Where("status IN ?", []enums.ExerciseStatus{enums.ExerciseStatusPending, enums.ExerciseStatusInProgress}).
		Updates(map[string]any{
			"status":      enums.ExerciseStatusIgnored,
			"finished_at": time.Now().UTC(),
		}).Error
}

func exerciseNotInProgressError(query *gorm.DB, exerciseID uuid.UUID) error {
	var deletedVocabularyResults int64
	if err := query.Model(&models.ExerciseVocabulary{}).
		Where("exercise_id = ? AND is_correct = ? AND result_reason = ?", exerciseID, true, ExerciseVocabularyResultReasonDeletedVocabulary).
		Count(&deletedVocabularyResults).Error; err != nil {
		return err
	}

	if deletedVocabularyResults > 0 {
		return ErrExerciseVocabularyDeleted
	}

	return ErrExerciseNotInProgress
}

func IgnoreUserExercise(exerciseID uuid.UUID, userID uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var exercise models.Exercise
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", exerciseID, userID).
			Take(&exercise).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrExerciseNotFound
			}
			return err
		}

		if exercise.Status != enums.ExerciseStatusPending && exercise.Status != enums.ExerciseStatusInProgress {
			return exerciseNotInProgressError(tx, exercise.ID)
		}

		result := tx.Model(&models.Exercise{}).
			Where("id = ? AND user_id = ?", exerciseID, userID).
			Where("status IN ?", []enums.ExerciseStatus{enums.ExerciseStatusPending, enums.ExerciseStatusInProgress}).
			Updates(map[string]any{
				"status":      enums.ExerciseStatusIgnored,
				"finished_at": time.Now().UTC(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrExerciseNotInProgress
		}

		progressDelta := exerciseProgressDeltasForType(exercise.Type).Wrong
		progressDelta = exerciseProgressDelta(&exercise, progressDelta)

		if isMatchPairsExerciseType(exercise.Type) {
			var vocabularyIDs []uuid.UUID
			if err := tx.Table("vocabulary_exercises").
				Where("exercise_id = ? AND is_correct = ?", exerciseID, true).
				Pluck("vocabulary_id", &vocabularyIDs).Error; err != nil {
				return err
			}

			for _, vocabularyID := range vocabularyIDs {
				if _, _, err := updateVocabularyProgressByID(
					tx,
					exerciseID,
					vocabularyID,
					ExerciseVocabularyResultIgnored,
					ExerciseVocabularyResultReasonSkipped,
					progressDelta,
					exercise.IsKnownVocabularyRepetition,
				); err != nil {
					return err
				}
			}
			return nil
		}

		_, _, err := updateVocabularyProgressByExercise(
			tx,
			exerciseID,
			ExerciseVocabularyResultIgnored,
			ExerciseVocabularyResultReasonSkipped,
			progressDelta,
			exercise.IsKnownVocabularyRepetition,
		)
		return err
	})
}

func IgnoreDuePendingExercisesWithoutActiveVocabulary(now time.Time) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			WITH affected AS (
				SELECT e.id
				FROM exercises AS e
				WHERE e.deleted_at IS NULL
					AND e.status = ?
					AND e.type IN (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
					AND e.scheduled_for <= ?
					AND (
						(e.type <> ? AND NOT EXISTS (
							SELECT 1
							FROM vocabulary_exercises AS ve
							JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
							WHERE ve.exercise_id = e.id
								AND ve.is_correct = true
						))
						OR (e.type = ? AND (
							SELECT COUNT(*)
							FROM vocabulary_exercises AS ve
							JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
							WHERE ve.exercise_id = e.id
								AND ve.is_correct = true
						) <> ?)
					)
			)
			UPDATE vocabulary_exercises AS ve
			SET result = ?, result_reason = ?, answered_at = ?
			FROM affected
			WHERE ve.exercise_id = affected.id
				AND ve.is_correct = true
				AND ve.result IS NULL
		`,
			enums.ExerciseStatusPending,
			enums.ExerciseTypeBasicDirect,
			enums.ExerciseTypeBasicReversed,
			enums.ExerciseTypeChoiceDirect,
			enums.ExerciseTypeChoiceReversed,
			enums.ExerciseTypeCharactersDirect,
			enums.ExerciseTypeCharactersReversed,
			enums.ExerciseTypeAudioDirect,
			enums.ExerciseTypeAudioReversed,
			enums.ExerciseTypeDescriptionDirect,
			enums.ExerciseTypeDescriptionReversed,
			enums.ExerciseTypeMatchPairs,
			now,
			enums.ExerciseTypeMatchPairs,
			enums.ExerciseTypeMatchPairs,
			matchPairsVocabularyCount,
			ExerciseVocabularyResultIgnored,
			ExerciseVocabularyResultReasonDeletedVocabulary,
			now,
		).Error; err != nil {
			return err
		}

		return tx.Exec(`
			UPDATE exercises AS e
			SET status = ?, finished_at = ?
			WHERE e.deleted_at IS NULL
				AND e.status = ?
				AND e.type IN (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				AND e.scheduled_for <= ?
				AND (
					(e.type <> ? AND NOT EXISTS (
						SELECT 1
						FROM vocabulary_exercises AS ve
						JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
						WHERE ve.exercise_id = e.id
							AND ve.is_correct = true
					))
					OR (e.type = ? AND (
						SELECT COUNT(*)
						FROM vocabulary_exercises AS ve
						JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
						WHERE ve.exercise_id = e.id
							AND ve.is_correct = true
					) <> ?)
				)
		`, enums.ExerciseStatusIgnored, now, enums.ExerciseStatusPending, enums.ExerciseTypeBasicDirect, enums.ExerciseTypeBasicReversed, enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed, enums.ExerciseTypeCharactersDirect, enums.ExerciseTypeCharactersReversed, enums.ExerciseTypeAudioDirect, enums.ExerciseTypeAudioReversed, enums.ExerciseTypeDescriptionDirect, enums.ExerciseTypeDescriptionReversed, enums.ExerciseTypeMatchPairs, now, enums.ExerciseTypeMatchPairs, enums.ExerciseTypeMatchPairs, matchPairsVocabularyCount).Error
	})
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
		WHERE e.deleted_at IS NULL
			AND u.deleted_at IS NULL
			AND e.status = ?
			AND e.telegram_message_id IS NOT NULL
			AND e.started_at IS NOT NULL
			AND e.started_at <= ?
			AND e.reminder_sent_at IS NULL
			AND u.settings->'telegram'->'bot_enabled' = ?
			AND (
				(e.type <> ? AND EXISTS (
					SELECT 1 FROM vocabulary_exercises ve
					JOIN vocabulary v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
					WHERE ve.exercise_id = e.id AND ve.is_correct = true
				))
				OR (e.type = ? AND (
					SELECT COUNT(*) FROM vocabulary_exercises ve
					JOIN vocabulary v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
					WHERE ve.exercise_id = e.id AND ve.is_correct = true
				) = ?)
			)
		ORDER BY e.started_at ASC
	`, enums.ExerciseStatusInProgress, remindBefore, true, enums.ExerciseTypeMatchPairs, enums.ExerciseTypeMatchPairs, matchPairsVocabularyCount).Scan(&reminders).Error

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
	telegramExpiresBefore := now.Add(-telegramExerciseExpirationPeriod)
	websiteExpiresBefore := now.Add(-websiteExerciseExpirationPeriod)

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := markExpiredExerciseVocabularyResults(tx, now, true, telegramExpiresBefore); err != nil {
			return err
		}

		return tx.Model(&models.Exercise{}).
			Where("status = ?", enums.ExerciseStatusInProgress).
			Where("started_at IS NOT NULL").
			Where("telegram_message_id IS NOT NULL").
			Where("started_at <= ?", telegramExpiresBefore).
			Updates(map[string]any{
				"status":      enums.ExerciseStatusIgnored,
				"finished_at": now,
			}).Error
	}); err != nil {
		return err
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := markExpiredExerciseVocabularyResults(tx, now, false, websiteExpiresBefore); err != nil {
			return err
		}

		return tx.Model(&models.Exercise{}).
			Where("status = ?", enums.ExerciseStatusInProgress).
			Where("started_at IS NOT NULL").
			Where("telegram_message_id IS NULL").
			Where("started_at <= ?", websiteExpiresBefore).
			Updates(map[string]any{
				"status":      enums.ExerciseStatusIgnored,
				"finished_at": now,
			}).Error
	})
}

func CompleteExercise(exerciseID uuid.UUID) (bool, int, error) {
	return FinishExercise(
		exerciseID,
		enums.ExerciseStatusCompleted,
		ExerciseVocabularyResultCorrect,
		ExerciseVocabularyResultReasonTypedAnswer,
		ExerciseBasicCorrectProgressDelta,
	)
}

func FinishExercise(exerciseID uuid.UUID, status enums.ExerciseStatus, result string, reason string, progressDelta int) (bool, int, error) {
	updated, translationKnowledge, _, err := FinishExerciseWithProgressDelta(exerciseID, status, result, reason, progressDelta)
	return updated, translationKnowledge, err
}

func FinishExerciseWithProgressDelta(exerciseID uuid.UUID, status enums.ExerciseStatus, result string, reason string, progressDelta int) (bool, int, int, error) {
	updated := false
	translationKnowledge := 0
	appliedProgressDelta := progressDelta

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		dbResult := tx.Model(&models.Exercise{}).
			Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
			Updates(map[string]any{
				"status":      status,
				"finished_at": time.Now().UTC(),
			})

		if dbResult.Error != nil {
			return dbResult.Error
		}

		if dbResult.RowsAffected == 0 {
			return nil
		}

		updated = true

		var exercise models.Exercise
		if err := tx.Select("is_known_vocabulary_repetition").
			Where("id = ?", exerciseID).
			Take(&exercise).Error; err != nil {
			return err
		}

		var updateErr error
		translationKnowledge, appliedProgressDelta, updateErr = updateVocabularyProgressByExercise(
			tx,
			exerciseID,
			result,
			reason,
			progressDelta,
			exercise.IsKnownVocabularyRepetition,
		)
		return updateErr
	})

	if err != nil {
		return false, 0, 0, err
	}

	return updated, translationKnowledge, appliedProgressDelta, nil
}

func updateVocabularyProgressByExercise(tx *gorm.DB, exerciseID uuid.UUID, result string, reason string, delta int, isKnownVocabularyRepetition bool) (int, int, error) {
	var exerciseLink struct {
		VocabularyID uuid.UUID `gorm:"column:vocabulary_id"`
	}

	if err := tx.Table("vocabulary_exercises").
		Select("vocabulary_id").
		Where("exercise_id = ?", exerciseID).
		Where("is_correct = ?", true).
		Take(&exerciseLink).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, delta, nil
		}

		return 0, 0, err
	}

	return updateVocabularyProgressByID(tx, exerciseID, exerciseLink.VocabularyID, result, reason, delta, isKnownVocabularyRepetition)
}

func updateVocabularyProgressByID(tx *gorm.DB, exerciseID uuid.UUID, vocabularyID uuid.UUID, result string, reason string, delta int, isKnownVocabularyRepetition bool) (int, int, error) {
	var vocabulary models.Vocabulary
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", vocabularyID).
		Where("deleted_at IS NULL").
		Take(&vocabulary).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, delta, nil
		}

		return 0, 0, err
	}

	translationKnowledge := 0
	found := false
	for index := range vocabulary.Progress {
		if vocabulary.Progress[index].Type != enums.KnowledgeTypeTranslation {
			continue
		}

		if isKnownVocabularyRepetition {
			delta = knownVocabularyRepetitionProgressDelta(vocabulary.Progress[index].Knowledge, delta)
		}
		vocabulary.Progress[index].Knowledge = clampProgress(vocabulary.Progress[index].Knowledge + delta)
		translationKnowledge = vocabulary.Progress[index].Knowledge
		found = true
		break
	}

	if !found {
		if isKnownVocabularyRepetition {
			delta = knownVocabularyRepetitionProgressDelta(0, delta)
		}
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
		return 0, 0, err
	}

	now := time.Now().UTC()
	if err := tx.Model(&models.ExerciseVocabulary{}).
		Where("exercise_id = ?", exerciseID).
		Where("vocabulary_id = ?", vocabulary.ID).
		Updates(map[string]any{
			"result":          result,
			"result_reason":   reason,
			"progress_delta":  delta,
			"knowledge_after": translationKnowledge,
			"answered_at":     now,
		}).Error; err != nil {
		return 0, 0, err
	}

	return translationKnowledge, delta, nil
}

func knownVocabularyRepetitionProgressDelta(currentKnowledge int, defaultDelta int) int {
	if defaultDelta < 0 {
		return KnownVocabularyRepetitionFailProgressDelta
	}
	if currentKnowledge >= 100 {
		return 0
	}

	return defaultDelta
}

func MarkExerciseVocabularyResultWithoutProgress(exerciseID uuid.UUID, result string, reason string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		return markExerciseVocabularyResultWithoutProgress(tx, exerciseID, result, reason)
	})
}

func markExerciseVocabularyResultWithoutProgress(tx *gorm.DB, exerciseID uuid.UUID, result string, reason string) error {
	return tx.Exec(`
		UPDATE vocabulary_exercises AS ve
		SET result = ?, result_reason = ?, answered_at = ?
		FROM exercises AS e
		WHERE e.id = ve.exercise_id
			AND e.id = ?
			AND e.deleted_at IS NULL
			AND e.status IN (?, ?)
			AND ve.is_correct = true
			AND ve.result IS NULL
	`,
		result,
		reason,
		time.Now().UTC(),
		exerciseID,
		enums.ExerciseStatusPending,
		enums.ExerciseStatusInProgress,
	).Error
}

func markExpiredExerciseVocabularyResults(tx *gorm.DB, now time.Time, telegramMessageIDPresent bool, startedBefore time.Time) error {
	messageIDPredicate := "telegram_message_id IS NULL"
	if telegramMessageIDPresent {
		messageIDPredicate = "telegram_message_id IS NOT NULL"
	}

	return tx.Exec(`
		UPDATE vocabulary_exercises AS ve
		SET result = ?, result_reason = ?, answered_at = ?
		FROM exercises AS e
		WHERE e.id = ve.exercise_id
			AND e.deleted_at IS NULL
			AND e.status = ?
			AND e.started_at IS NOT NULL
			AND e.`+messageIDPredicate+`
			AND e.started_at <= ?
			AND ve.is_correct = true
			AND ve.result IS NULL
	`, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonExpired, now, enums.ExerciseStatusInProgress, startedBefore).Error
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
