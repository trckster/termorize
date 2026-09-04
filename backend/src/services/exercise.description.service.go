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
	"termorize/src/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrDescriptionGenerationFailed = errors.New("description generation failed")
	ErrExerciseNotDescription      = errors.New("exercise is not a description exercise")
)

var descriptionWordPattern = regexp.MustCompile(`[\pL\pN]+(?:['’][\pL\pN]+)?`)

const maxDescriptionRunes = 300

var descriptionExerciseTypes = []enums.ExerciseType{
	enums.ExerciseTypeDescriptionDirect,
	enums.ExerciseTypeDescriptionReversed,
}

type pendingDescriptionReplacement struct {
	ExerciseID   uuid.UUID `gorm:"column:exercise_id"`
	ScheduledFor time.Time `gorm:"column:scheduled_for"`
}

func IsDescriptionLanguageEligible(userID uint, language enums.Language) (bool, error) {
	var user models.User
	if err := db.DB.Select("settings").Where("id = ?", userID).Take(&user).Error; err != nil {
		return false, err
	}
	return descriptionLanguageEligible(user.Settings, language), nil
}

func lockDescriptionLanguageEligibility(tx *gorm.DB, userID uint, language enums.Language) (bool, error) {
	var user models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("settings").
		Where("id = ?", userID).
		Take(&user).Error; err != nil {
		return false, err
	}
	return descriptionLanguageEligible(user.Settings, language), nil
}

func descriptionLanguageEligible(settings models.UserSettings, language enums.Language) bool {
	return settings.MainLearningLanguage == language &&
		!containsLanguage(settings.IgnoredDescriptionLanguages, language)
}

func IgnoreDescriptionLanguageForExercise(exerciseID uuid.UUID, userID uint) (*models.User, error) {
	var user models.User
	err := db.DB.Transaction(func(tx *gorm.DB) error {
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
		if !isDescriptionExerciseType(exercise.Type) {
			return ErrExerciseNotDescription
		}
		if !exercise.DeletedAt.Valid && exercise.Status != enums.ExerciseStatusPending && exercise.Status != enums.ExerciseStatusInProgress {
			return ErrExerciseNotInProgress
		}

		descriptionLanguage, err := getDescriptionExerciseLanguage(tx, exercise)
		if err != nil {
			return err
		}
		newlyIgnored := !containsLanguage(user.Settings.IgnoredDescriptionLanguages, descriptionLanguage)
		if newlyIgnored {
			settings := user.Settings
			settings.IgnoredDescriptionLanguages = canonicalIgnoredDescriptionLanguages(
				append(settings.IgnoredDescriptionLanguages, descriptionLanguage),
			)
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
			if err := cancelInProgressDescriptionExercises(tx, userID); err != nil {
				return err
			}
			return replacePendingDescriptionExercises(tx, userID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func RemoveIgnoredDescriptionLanguage(userID uint, language enums.Language) (*models.User, error) {
	var user models.User
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).Take(&user).Error; err != nil {
			return err
		}

		filtered := make([]enums.Language, 0, len(user.Settings.IgnoredDescriptionLanguages))
		for _, ignored := range user.Settings.IgnoredDescriptionLanguages {
			if ignored != language {
				filtered = append(filtered, ignored)
			}
		}
		settings := user.Settings
		settings.IgnoredDescriptionLanguages = canonicalIgnoredDescriptionLanguages(filtered)
		user.Settings = settings
		return tx.Model(&user).Update("settings", settings).Error
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func canonicalIgnoredDescriptionLanguages(languages []enums.Language) []enums.Language {
	settings := models.UserSettings{IgnoredDescriptionLanguages: languages}.WithDefaults()
	return settings.IgnoredDescriptionLanguages
}

func getDescriptionExerciseLanguage(tx *gorm.DB, exercise models.Exercise) (enums.Language, error) {
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
	if exercise.Type == enums.ExerciseTypeDescriptionReversed {
		return details.TranslationLanguage, nil
	}
	return details.OriginalLanguage, nil
}

func IsPendingDescriptionExercise(exerciseID uuid.UUID) (bool, error) {
	var count int64
	if err := db.DB.Model(&models.Exercise{}).
		Where("id = ? AND status = ? AND type IN ?", exerciseID, enums.ExerciseStatusPending, descriptionExerciseTypes).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == 1, nil
}

func replacePendingDescriptionExercises(tx *gorm.DB, userID uint) error {
	var replacements []pendingDescriptionReplacement
	if err := tx.Model(&models.Exercise{}).
		Select("id AS exercise_id", "scheduled_for").
		Where("user_id = ? AND status = ? AND type IN ? AND scheduled_for IS NOT NULL", userID, enums.ExerciseStatusPending, descriptionExerciseTypes).
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

func cancelInProgressDescriptionExercises(tx *gorm.DB, userID uint) error {
	return tx.Where(
		"user_id = ? AND status = ? AND type IN ?",
		userID,
		enums.ExerciseStatusInProgress,
		descriptionExerciseTypes,
	).Delete(&models.Exercise{}).Error
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

func GetOrCreateWordDescription(wordID uuid.UUID) (*models.WordDescription, error) {
	model := config.GetOpenRouterModel()
	var cached models.WordDescription
	err := db.DB.Where("word_id = ? AND model = ?", wordID, model).Take(&cached).Error
	if err == nil {
		return &cached, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var description *models.WordDescription
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		lockKey := "word-description:" + wordID.String() + ":" + model
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		if err := tx.Where("word_id = ? AND model = ?", wordID, model).Take(&cached).Error; err == nil {
			description = &cached
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var word models.Word
		if err := tx.Where("id = ?", wordID).Take(&word).Error; err != nil {
			return err
		}

		generated, err := openrouter.NewClient().GenerateDescription(
			word.Word,
			word.Language.DisplayName(),
			word.Language.DisplayName(),
		)
		if err != nil {
			return errors.Join(ErrDescriptionGenerationFailed, err)
		}
		generatedText := strings.TrimSpace(generated.Description)
		if generatedText == "" || len([]rune(generatedText)) > maxDescriptionRunes ||
			descriptionMentionsAnswer(generatedText, word.Word) {
			return ErrDescriptionGenerationFailed
		}
		for _, forbiddenForm := range generated.ForbiddenForms {
			if descriptionMentionsAnswer(generatedText, forbiddenForm) {
				return ErrDescriptionGenerationFailed
			}
		}

		detectedLanguage, supported, err := DetectLanguage(generatedText)
		if err != nil {
			return errors.Join(ErrDescriptionGenerationFailed, err)
		}
		if !supported || detectedLanguage != word.Language {
			return ErrDescriptionGenerationFailed
		}

		created := models.WordDescription{
			WordID:      word.ID,
			Model:       model,
			Description: generatedText,
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		description = &created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return description, nil
}

func descriptionMentionsAnswer(description, answer string) bool {
	normalizedDescription := normalizeAnswer(description)
	normalizedAnswer := normalizeAnswer(answer)
	if normalizedAnswer == "" {
		return false
	}

	candidates := []string{normalizedAnswer}
	for _, prefix := range []string{
		"de la ", "de l'", "de l’",
		"a ", "an ", "the ", "to ",
		"il ", "lo ", "la ", "l'", "l’", "i ", "gli ", "le ", "un ", "uno ", "una ",
		"el ", "los ", "las ", "unos ", "unas ",
		"les ", "des ", "du ", "une ",
		"der ", "die ", "das ", "den ", "dem ", "ein ", "eine ", "einen ", "einem ", "einer ", "eines ",
		"o ", "os ", "as ", "um ", "uma ", "uns ", "umas ",
	} {
		if withoutPrefix := strings.TrimSpace(strings.TrimPrefix(normalizedAnswer, prefix)); withoutPrefix != normalizedAnswer && withoutPrefix != "" {
			candidates = append(candidates, withoutPrefix)
			break
		}
	}

	descriptionWords := descriptionWordPattern.FindAllString(normalizedDescription, -1)
	for _, candidate := range candidates {
		pattern := `(^|[^\pL\pN])` + regexp.QuoteMeta(candidate) + `([^\pL\pN]|$)`
		if regexp.MustCompile(pattern).FindStringIndex(normalizedDescription) != nil {
			return true
		}
		if descriptionContainsPhraseVariant(descriptionWords, candidate) {
			return true
		}
	}
	return false
}

func descriptionContainsPhraseVariant(descriptionWords []string, candidate string) bool {
	candidateWords := descriptionWordPattern.FindAllString(candidate, -1)
	if len(candidateWords) == 0 || len(candidateWords) > len(descriptionWords) {
		return false
	}
	for start := 0; start <= len(descriptionWords)-len(candidateWords); start++ {
		matches := true
		for index, candidateWord := range candidateWords {
			if !descriptionWordVariant(candidateWord, descriptionWords[start+index]) {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func descriptionWordVariant(candidate, word string) bool {
	if candidate == word {
		return true
	}
	candidateRunes := []rune(candidate)
	wordRunes := []rune(word)
	if len(candidateRunes) >= 3 && strings.HasPrefix(word, candidate) {
		suffix := strings.TrimPrefix(word, candidate)
		if suffix == "s" || suffix == "es" || suffix == "ed" || suffix == "ing" {
			return true
		}
	}
	return len(candidateRunes) >= 4 &&
		(utils.DamerauLevenshteinDistance(candidate, word) <= 1 ||
			(len(wordRunes) >= len(candidateRunes) && commonRunePrefix(candidateRunes, wordRunes) >= len(candidateRunes)-1))
}

func commonRunePrefix(left, right []rune) int {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for index := 0; index < limit; index++ {
		if left[index] != right[index] {
			return index
		}
	}
	return limit
}
