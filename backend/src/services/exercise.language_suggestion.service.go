package services

import (
	"errors"
	"termorize/src/data/db"
	"termorize/src/enums"

	"gorm.io/gorm"
)

const (
	ExerciseLanguageSuggestionFamilyAudio       = "audio"
	ExerciseLanguageSuggestionFamilyDescription = "description"
	exerciseLanguageSuggestionLimit             = 5
)

func ReserveExerciseLanguageSuggestion(userID uint, family string, language enums.Language) (bool, error) {
	return reserveExerciseLanguageSuggestionWithDB(db.DB, userID, family, language)
}

func reserveExerciseLanguageSuggestionWithDB(conn *gorm.DB, userID uint, family string, language enums.Language) (bool, error) {
	if userID == 0 || !validExerciseLanguageSuggestionFamily(family) || !enums.IsSupportedLanguage(language) {
		return false, errors.New("invalid exercise language suggestion")
	}

	var shownCount int
	result := conn.Raw(`
		INSERT INTO exercise_language_suggestion_counts
			(user_id, family, language, shown_count, created_at, updated_at)
		VALUES (?, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, family, language) DO UPDATE
		SET shown_count = exercise_language_suggestion_counts.shown_count + 1,
			updated_at = CURRENT_TIMESTAMP
		WHERE exercise_language_suggestion_counts.shown_count < ?
		RETURNING shown_count
	`, userID, family, language, exerciseLanguageSuggestionLimit).Scan(&shownCount)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func ReleaseExerciseLanguageSuggestion(userID uint, family string, language enums.Language) error {
	if userID == 0 || !validExerciseLanguageSuggestionFamily(family) || !enums.IsSupportedLanguage(language) {
		return errors.New("invalid exercise language suggestion")
	}
	return db.DB.Exec(`
		UPDATE exercise_language_suggestion_counts
		SET shown_count = shown_count - 1, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND family = ? AND language = ? AND shown_count > 0
	`, userID, family, language).Error
}

func validExerciseLanguageSuggestionFamily(family string) bool {
	return family == ExerciseLanguageSuggestionFamilyAudio ||
		family == ExerciseLanguageSuggestionFamilyDescription
}
