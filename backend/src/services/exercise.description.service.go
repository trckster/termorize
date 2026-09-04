package services

import (
	"errors"
	"regexp"
	"strings"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrDescriptionGenerationFailed = errors.New("description generation failed")

type pendingDescriptionReplacement struct {
	ExerciseID   uuid.UUID `gorm:"column:exercise_id"`
	ScheduledFor time.Time `gorm:"column:scheduled_for"`
}

func IsDescriptionLanguageEligible(userID uint, language enums.Language) (bool, error) {
	var user models.User
	if err := db.DB.Select("settings").Where("id = ?", userID).Take(&user).Error; err != nil {
		return false, err
	}
	return user.Settings.MainLearningLanguage == language &&
		!containsLanguage(user.Settings.IgnoredDescriptionLanguages, language), nil
}

func replacePendingDescriptionExercises(tx *gorm.DB, userID uint) error {
	var replacements []pendingDescriptionReplacement
	if err := tx.Model(&models.Exercise{}).
		Select("id AS exercise_id", "scheduled_for").
		Where("user_id = ? AND status = ? AND type = ? AND scheduled_for IS NOT NULL", userID, enums.ExerciseStatusPending, enums.ExerciseTypeDescriptionReversed).
		Order("scheduled_for ASC, created_at ASC").
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

func ReplacePendingDescriptionExercise(exerciseID uuid.UUID, excludeDescription bool) (bool, error) {
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
		if !isDescriptionExerciseType(exercise.Type) || exercise.ScheduledFor == nil {
			return nil
		}
		if err := tx.Delete(&exercise).Error; err != nil {
			return err
		}
		var err error
		if excludeDescription {
			replaced, err = createReplacementPendingExerciseWithoutDescription(tx, exercise.UserID, *exercise.ScheduledFor)
		} else {
			replaced, err = createReplacementPendingExercise(tx, exercise.UserID, *exercise.ScheduledFor)
		}
		return err
	})
	return replaced, err
}

func GetOrCreateTranslationDescription(translationID uuid.UUID) (*models.TranslationDescription, error) {
	model := config.GetOpenRouterModel()
	var cached models.TranslationDescription
	err := db.DB.Where("translation_id = ? AND model = ?", translationID, model).Take(&cached).Error
	if err == nil {
		return &cached, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var translation models.Translation
	if err := db.DB.
		Preload("Original").
		Preload("Translation").
		Where("id = ?", translationID).
		Take(&translation).Error; err != nil {
		return nil, err
	}
	if translation.Original == nil || translation.Translation == nil {
		return nil, errors.New("translation words are missing")
	}

	generated, err := openrouter.NewClient().GenerateDescription(
		translation.Translation.Word,
		translation.Translation.Language.DisplayName(),
		translation.Original.Language.DisplayName(),
	)
	if err != nil {
		return nil, errors.Join(ErrDescriptionGenerationFailed, err)
	}
	description := strings.TrimSpace(generated.Description)
	if description == "" || descriptionMentionsAnswer(description, translation.Original.Word) {
		return nil, ErrDescriptionGenerationFailed
	}

	created := models.TranslationDescription{
		TranslationID: translation.ID,
		Model:         model,
		Description:   description,
	}
	result := db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "translation_id"}, {Name: "model"}},
		DoNothing: true,
	}).Create(&created)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 1 {
		return &created, nil
	}

	if err := db.DB.Where("translation_id = ? AND model = ?", translationID, model).Take(&cached).Error; err != nil {
		return nil, err
	}
	return &cached, nil
}

func descriptionMentionsAnswer(description, answer string) bool {
	normalizedDescription := normalizeAnswer(description)
	normalizedAnswer := normalizeAnswer(answer)
	if normalizedAnswer == "" {
		return false
	}

	candidates := []string{normalizedAnswer}
	for _, article := range []string{"a ", "an ", "the ", "il ", "lo ", "la ", "l'", "l’", "i ", "gli ", "le ", "el ", "los ", "las ", "un ", "una ", "une ", "der ", "die ", "das ", "ein ", "eine ", "o ", "os ", "as "} {
		if withoutArticle := strings.TrimSpace(strings.TrimPrefix(normalizedAnswer, article)); withoutArticle != normalizedAnswer && withoutArticle != "" {
			candidates = append(candidates, withoutArticle)
			break
		}
	}

	for _, candidate := range candidates {
		pattern := `(^|[^\pL\pN])` + regexp.QuoteMeta(candidate) + `([^\pL\pN]|$)`
		if regexp.MustCompile(pattern).FindStringIndex(normalizedDescription) != nil {
			return true
		}
	}
	return false
}
