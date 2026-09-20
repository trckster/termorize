package services

import (
	"strings"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/google"
	"termorize/src/logger"
	"termorize/src/models"

	"github.com/google/uuid"
)

// SaveExerciseDescription keeps the exact clue delivered to the learner.
func SaveExerciseDescription(exerciseID uuid.UUID, description string) error {
	return db.DB.Model(&models.Exercise{}).Where("id = ?", exerciseID).
		Updates(map[string]any{"description": description, "description_translation": ""}).Error
}

// AddDescriptionFeedback is best effort: a translation outage must not hide an
// answer or undo an already recorded result. Successful translations are reused
// when Telegram retries delivery.
func AddDescriptionFeedback(exerciseID uuid.UUID, userID uint, result *VerifyAnswerResult) {
	if result == nil || result.DescriptionFeedback != nil ||
		(result.Result != ExerciseVocabularyResultWrong && result.Result != ExerciseVocabularyResultIgnored) {
		return
	}
	exercise, vocabulary, err := getExerciseWithCorrectVocabulary(exerciseID, userID)
	if err != nil || vocabulary == nil || !isDescriptionExerciseType(exercise.Type) || exercise.Status != enums.ExerciseStatusFailed {
		return
	}
	source, target := vocabulary.OriginalLanguage, vocabulary.TranslationLanguage
	answerTranslation := vocabulary.TranslationWord
	if isReversedExerciseType(exercise.Type) {
		source, target = target, source
		answerTranslation = vocabulary.OriginalWord
	}
	feedback := &DescriptionFeedback{Original: exercise.Description, Language: target, AnswerTranslation: answerTranslation}
	result.DescriptionFeedback = feedback
	if exercise.DescriptionTranslation != "" {
		feedback.Translation = exercise.DescriptionTranslation
		return
	}
	description := exercise.Description
	if description == "" {
		// Exercises sent before snapshots were introduced can use their cached
		// clue. Do not generate a new description for an already answered task.
		var entry models.Vocabulary
		if err := db.DB.Preload("Translation").First(&entry, "id = ?", vocabulary.VocabularyID).Error; err != nil || entry.Translation == nil {
			return
		}
		wordID, translationID := entry.Translation.OriginalID, entry.Translation.TranslationID
		if isReversedExerciseType(exercise.Type) {
			wordID, translationID = translationID, wordID
		}
		var cached models.WordDescription
		if err := descriptionCacheQuery(db.DB, wordID, translationID, config.GetOpenRouterModel()).Take(&cached).Error; err != nil {
			return
		}
		description = cached.Description
	}
	feedback.Original = description
	translated, err := google.NewTranslateClient().Translate(description, string(source), string(target))
	if err != nil {
		logger.L().Warnw("failed to translate exercise description", "exercise_id", exerciseID, "error", err)
		return
	}
	feedback.Translation = strings.TrimSpace(translated)
	if feedback.Translation == "" {
		return
	}
	if err := db.DB.Model(&models.Exercise{}).Where("id = ?", exerciseID).
		Updates(map[string]any{"description": description, "description_translation": feedback.Translation}).Error; err != nil {
		logger.L().Warnw("failed to cache exercise description translation", "exercise_id", exerciseID, "error", err)
	}
}
