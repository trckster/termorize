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

var ErrExerciseNotAudio = errors.New("exercise is not an audio exercise")

type audioExerciseLanguageDetails struct {
	OriginalLanguage    enums.Language `gorm:"column:original_language"`
	TranslationLanguage enums.Language `gorm:"column:translation_language"`
}

type pendingAudioReplacement struct {
	ExerciseID   uuid.UUID `gorm:"column:exercise_id"`
	ScheduledFor time.Time `gorm:"column:scheduled_for"`
}

func canonicalIgnoredAudioLanguages(languages []enums.Language) []enums.Language {
	settings := models.UserSettings{IgnoredAudioLanguages: languages}.WithDefaults()
	return settings.IgnoredAudioLanguages
}

func containsLanguage(languages []enums.Language, language enums.Language) bool {
	for _, candidate := range languages {
		if candidate == language {
			return true
		}
	}
	return false
}

func newlyIgnoredAudioLanguages(previous, next []enums.Language) []enums.Language {
	newLanguages := make([]enums.Language, 0, len(next))
	for _, language := range next {
		if !containsLanguage(previous, language) {
			newLanguages = append(newLanguages, language)
		}
	}
	return newLanguages
}

func IsAudioLanguageIgnored(userID uint, language enums.Language) (bool, error) {
	var user models.User
	if err := db.DB.Select("settings").Where("id = ?", userID).Take(&user).Error; err != nil {
		return false, err
	}
	return containsLanguage(user.Settings.IgnoredAudioLanguages, language), nil
}

func IgnoreAudioLanguageForExercise(exerciseID uuid.UUID, userID uint) (*models.User, error) {
	var user models.User
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// Settings updates use the same user-then-exercise lock order while
		// replacing pending audio exercises.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).Take(&user).Error; err != nil {
			return err
		}

		var exercise models.Exercise
		if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", exerciseID, userID).
			Take(&exercise).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrExerciseNotFound
			}
			return err
		}
		if !isAudioExerciseType(exercise.Type) {
			return ErrExerciseNotAudio
		}
		if !exercise.DeletedAt.Valid && exercise.Status != enums.ExerciseStatusPending && exercise.Status != enums.ExerciseStatusInProgress {
			return ErrExerciseNotInProgress
		}

		spokenLanguage, err := getAudioExerciseSpokenLanguage(tx, exercise)
		if err != nil {
			return err
		}

		newlyIgnored := !containsLanguage(user.Settings.IgnoredAudioLanguages, spokenLanguage)
		if newlyIgnored {
			settings := user.Settings
			settings.IgnoredAudioLanguages = canonicalIgnoredAudioLanguages(append(settings.IgnoredAudioLanguages, spokenLanguage))
			user.Settings = settings
			if err := tx.Model(&user).Update("settings", settings).Error; err != nil {
				return err
			}
		}

		if !exercise.DeletedAt.Valid {
			if err := tx.Delete(&exercise).Error; err != nil {
				return err
			}
		}

		if newlyIgnored {
			return replacePendingAudioExercisesForLanguages(tx, userID, []enums.Language{spokenLanguage})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func RemoveIgnoredAudioLanguage(userID uint, language enums.Language) (*models.User, error) {
	var user models.User
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).Take(&user).Error; err != nil {
			return err
		}

		filtered := make([]enums.Language, 0, len(user.Settings.IgnoredAudioLanguages))
		for _, ignored := range user.Settings.IgnoredAudioLanguages {
			if ignored != language {
				filtered = append(filtered, ignored)
			}
		}
		settings := user.Settings
		settings.IgnoredAudioLanguages = canonicalIgnoredAudioLanguages(filtered)
		user.Settings = settings
		return tx.Model(&user).Update("settings", settings).Error
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func getAudioExerciseSpokenLanguage(tx *gorm.DB, exercise models.Exercise) (enums.Language, error) {
	var details audioExerciseLanguageDetails
	if err := tx.Raw(`
		SELECT original.language AS original_language, translated.language AS translation_language
		FROM vocabulary_exercises AS ve
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE ve.exercise_id = ? AND ve.is_correct = true
		ORDER BY ve.position ASC
		LIMIT 1
	`, exercise.ID).Scan(&details).Error; err != nil {
		return "", err
	}
	if details.OriginalLanguage == "" || details.TranslationLanguage == "" {
		return "", ErrExerciseVocabularyDeleted
	}
	if exercise.Type == enums.ExerciseTypeAudioReversed {
		return details.TranslationLanguage, nil
	}
	return details.OriginalLanguage, nil
}

func replacePendingAudioExercisesForLanguages(tx *gorm.DB, userID uint, languages []enums.Language) error {
	if len(languages) == 0 {
		return nil
	}

	var replacements []pendingAudioReplacement
	if err := tx.Raw(`
		SELECT e.id AS exercise_id, e.scheduled_for
		FROM exercises AS e
		JOIN vocabulary_exercises AS ve ON ve.exercise_id = e.id AND ve.is_correct = true
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE e.user_id = ?
			AND e.deleted_at IS NULL
			AND e.status = ?
			AND e.scheduled_for IS NOT NULL
			AND ((e.type = ? AND original.language IN ?) OR (e.type = ? AND translated.language IN ?))
		ORDER BY e.scheduled_for ASC, e.created_at ASC
	`, userID, enums.ExerciseStatusPending, enums.ExerciseTypeAudioDirect, languages, enums.ExerciseTypeAudioReversed, languages).
		Scan(&replacements).Error; err != nil {
		return err
	}

	for _, replacement := range replacements {
		result := tx.Where("id = ? AND status = ?", replacement.ExerciseID, enums.ExerciseStatusPending).
			Delete(&models.Exercise{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			continue
		}
		if _, err := createReplacementPendingExercise(tx, userID, replacement.ScheduledFor); err != nil {
			return err
		}
	}
	return nil
}

func ReplacePendingAudioExercise(exerciseID uuid.UUID, nonAudioReplacement bool) (bool, error) {
	replaced := false
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var exercise models.Exercise
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusPending).
			Take(&exercise).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if !isAudioExerciseType(exercise.Type) || exercise.ScheduledFor == nil {
			return nil
		}
		if err := tx.Delete(&exercise).Error; err != nil {
			return err
		}
		var err error
		if nonAudioReplacement {
			replaced, err = createReplacementPendingExerciseWithoutAudio(tx, exercise.UserID, *exercise.ScheduledFor)
		} else {
			replaced, err = createReplacementPendingExercise(tx, exercise.UserID, *exercise.ScheduledFor)
		}
		return err
	})
	return replaced, err
}
