package services

import (
	"errors"
	"math/rand"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var rollKnownVocabularyRepetitionDice = func() int {
	return rand.Intn(knownVocabularyRepetitionDiceSides) + 1
}

func SetKnownVocabularyRepetitionDiceRollForTest(roll int) func() {
	previous := rollKnownVocabularyRepetitionDice
	rollKnownVocabularyRepetitionDice = func() int {
		return roll
	}

	return func() {
		rollKnownVocabularyRepetitionDice = previous
	}
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

	if totalMinutes <= 0 {
		logger.L().Errorw("user has no valid daily questions schedule intervals", "user_id", user.ID)
		return 0
	}

	candidateLimit := requestedExercisesCount * 10
	if candidateLimit < requestedExercisesCount {
		candidateLimit = requestedExercisesCount
	}

	vocabularyIDs, err := getEligibleVocabularyIDs(user.ID, candidateLimit)
	if err != nil {
		logger.L().Errorw("failed to select vocabulary for exercises", "user_id", user.ID, "error", err)
		return 0
	}

	generatedCount := 0

	for _, vocabularyID := range vocabularyIDs {
		if generatedCount >= int(requestedExercisesCount) {
			break
		}

		midnightOffset := rand.Intn(totalMinutes)

		realOffsetInMinutes := MapOffsetOnSchedule(user.Settings.Telegram.DailyQuestionsSchedule, midnightOffset)

		exerciseScheduleTime := targetMidnight.Add(time.Duration(realOffsetInMinutes) * time.Minute).UTC()

		if err := generateExercise(user.ID, vocabularyID, exerciseScheduleTime, true); err != nil {
			if errors.Is(err, errNoExerciseTypeAvailable) {
				continue
			}

			logger.L().Errorw("failed to generate exercise", "user_id", user.ID, "scheduled_for", exerciseScheduleTime, "error", err)
			continue
		}

		generatedCount++
	}

	if shouldScheduleKnownVocabularyRepetition(requestedExercisesCount) {
		midnightOffset := rand.Intn(totalMinutes)
		realOffsetInMinutes := MapOffsetOnSchedule(user.Settings.Telegram.DailyQuestionsSchedule, midnightOffset)
		exerciseScheduleTime := targetMidnight.Add(time.Duration(realOffsetInMinutes) * time.Minute).UTC()

		if _, err := CreatePendingKnownVocabularyRepetition(user.ID, exerciseScheduleTime); err != nil {
			if !errors.Is(err, ErrNoVocabularyForExercise) {
				logger.L().Errorw("failed to generate known vocabulary repetition", "user_id", user.ID, "scheduled_for", exerciseScheduleTime, "error", err)
			}
		} else {
			generatedCount++
		}
	}

	return generatedCount
}

func shouldScheduleKnownVocabularyRepetition(dailyQuestionsCount uint) bool {
	return dailyQuestionsCount >= knownVocabularyRepetitionMinimumDailyCount &&
		rollKnownVocabularyRepetitionDice() == knownVocabularyRepetitionWinningRoll
}

func CreatePendingKnownVocabularyRepetition(userID uint, when time.Time) (uuid.UUID, error) {
	vocabularyID, err := getKnownVocabularyID(userID)
	if err != nil {
		return uuid.Nil, err
	}
	if vocabularyID == uuid.Nil {
		return uuid.Nil, ErrNoVocabularyForExercise
	}

	vocabulary, err := loadExerciseVocabulary(vocabularyID)
	if err != nil {
		return uuid.Nil, err
	}

	exerciseTypes := []enums.ExerciseType{
		enums.ExerciseTypeBasicDirect,
		enums.ExerciseTypeBasicReversed,
	}
	exerciseType := exerciseTypes[rand.Intn(len(exerciseTypes))]

	_, _, _, err = buildExerciseQuestionData(vocabulary, exerciseType)
	if err != nil {
		return uuid.Nil, err
	}

	answerWord := vocabulary.Translation.Translation.Word
	if isReversedExerciseType(exerciseType) {
		answerWord = vocabulary.Translation.Original.Word
	}

	exercise := models.Exercise{
		Type:                        exerciseType,
		Status:                      enums.ExerciseStatusPending,
		UserID:                      userID,
		ScheduledFor:                &when,
		IsKnownVocabularyRepetition: true,
	}
	options := []exerciseChoiceCandidate{{
		VocabularyID: vocabularyID,
		AnswerWord:   answerWord,
	}}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&exercise).Error; err != nil {
			return err
		}

		return createExerciseVocabularyLinks(tx, exercise.ID, vocabularyID, options)
	})
	if err != nil {
		return uuid.Nil, err
	}

	return exercise.ID, nil
}

func getKnownVocabularyID(userID uint) (uuid.UUID, error) {
	var vocabularyIDs []uuid.UUID

	err := db.DB.
		Model(&models.Vocabulary{}).
		Select("id").
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Where(`EXISTS (
			SELECT 1
			FROM jsonb_array_elements(progress) AS p
			WHERE p->>'type' = ? AND (p->>'knowledge')::int = ?
		)`, enums.KnowledgeTypeTranslation, 100).
		Order("RANDOM()").
		Limit(1).
		Pluck("id", &vocabularyIDs).Error
	if err != nil {
		return uuid.Nil, err
	}
	if len(vocabularyIDs) == 0 {
		return uuid.Nil, nil
	}

	return vocabularyIDs[0], nil
}

func getEligibleVocabularyIDs(userID uint, limit uint) ([]uuid.UUID, error) {
	return getEligibleVocabularyIDsWithDB(db.DB, userID, limit)
}

func getEligibleVocabularyIDsWithDB(conn *gorm.DB, userID uint, limit uint) ([]uuid.UUID, error) {
	limitAsInt := int(limit)
	vocabularyIDs := make([]uuid.UUID, 0, limitAsInt)

	err := conn.
		Model(&models.Vocabulary{}).
		Select("id").
		Where("user_id = ?", userID).
		Where("mastered_at IS NULL").
		Where("deleted_at IS NULL").
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

func generateExercise(userID uint, vocabularyID uuid.UUID, when time.Time, includeMatchPairs bool) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		return generateExerciseWithDB(tx, userID, vocabularyID, when, includeMatchPairs)
	})
}

func generateExerciseWithDB(conn *gorm.DB, userID uint, vocabularyID uuid.UUID, when time.Time, includeMatchPairs bool) error {
	return generateExerciseWithDBConfig(conn, userID, vocabularyID, when, includeMatchPairs, false, false)
}

func generateExerciseWithDBConfig(conn *gorm.DB, userID uint, vocabularyID uuid.UUID, when time.Time, includeMatchPairs bool, excludeAudio, excludeDescription bool) error {
	vocabulary, err := loadExerciseVocabularyWithDB(conn, vocabularyID)
	if err != nil {
		return err
	}

	exerciseType, options, err := selectExerciseTypeAndOptionsWithConfig(conn, userID, vocabulary, includeMatchPairs, excludeAudio, excludeDescription)
	if err != nil {
		return err
	}

	exercise := models.Exercise{
		Type:         exerciseType,
		Status:       enums.ExerciseStatusPending,
		UserID:       userID,
		ScheduledFor: &when,
	}

	if err := conn.Create(&exercise).Error; err != nil {
		return err
	}

	correctVocabularyID := vocabularyID
	if exerciseType == enums.ExerciseTypeMatchPairs {
		correctVocabularyID = uuid.Nil
	}

	return createExerciseVocabularyLinks(conn, exercise.ID, correctVocabularyID, options)
}

func createReplacementPendingExercise(tx *gorm.DB, userID uint, when time.Time) (bool, error) {
	return createReplacementPendingExerciseWithConfig(tx, userID, when, false, false)
}

func createReplacementPendingExerciseWithoutAudio(tx *gorm.DB, userID uint, when time.Time) (bool, error) {
	return createReplacementPendingExerciseWithConfig(tx, userID, when, true, false)
}

func createReplacementPendingExerciseWithoutDescription(tx *gorm.DB, userID uint, when time.Time) (bool, error) {
	return createReplacementPendingExerciseWithConfig(tx, userID, when, false, true)
}

func createReplacementPendingExerciseWithConfig(tx *gorm.DB, userID uint, when time.Time, excludeAudio, excludeDescription bool) (bool, error) {
	vocabularyIDs, err := getEligibleVocabularyIDsWithDB(tx, userID, 64)
	if err != nil {
		return false, err
	}

	for _, vocabularyID := range vocabularyIDs {
		err := generateExerciseWithDBConfig(tx, userID, vocabularyID, when, true, excludeAudio, excludeDescription)
		if errors.Is(err, errNoExerciseTypeAvailable) {
			continue
		}
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func CreatePendingMatchExercise(userID uint, when time.Time) (uuid.UUID, error) {
	return generateMatchPairsExercise(userID, when)
}

func CreatePendingCharacterExercise(userID uint, when time.Time) (*RandomExerciseResult, error) {
	vocabularyIDs, err := getEligibleVocabularyIDs(userID, 64)
	if err != nil {
		return nil, err
	}
	if len(vocabularyIDs) == 0 {
		hasVocabulary, hasVocabularyErr := userHasVocabulary(userID)
		if hasVocabularyErr != nil {
			return nil, hasVocabularyErr
		}
		if hasVocabulary {
			return nil, ErrAllVocabularyMastered
		}
		return nil, ErrNoVocabularyForExercise
	}

	vocabulary, err := loadExerciseVocabulary(vocabularyIDs[0])
	if err != nil {
		return nil, err
	}

	exerciseTypes := []enums.ExerciseType{
		enums.ExerciseTypeCharactersDirect,
		enums.ExerciseTypeCharactersReversed,
	}
	exerciseType := exerciseTypes[rand.Intn(len(exerciseTypes))]
	questionWord, language, answerLanguage, err := buildExerciseQuestionData(vocabulary, exerciseType)
	if err != nil {
		return nil, err
	}

	answerWord := vocabulary.Translation.Translation.Word
	if isReversedExerciseType(exerciseType) {
		answerWord = vocabulary.Translation.Original.Word
	}
	if len(AnswerCharacters(answerWord)) == 0 {
		return nil, errNoExerciseTypeAvailable
	}

	exercise := models.Exercise{
		Type:         exerciseType,
		Status:       enums.ExerciseStatusPending,
		UserID:       userID,
		ScheduledFor: &when,
	}
	options := []exerciseChoiceCandidate{{
		VocabularyID: vocabulary.ID,
		AnswerWord:   answerWord,
	}}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Create(&exercise).Error; createErr != nil {
			return createErr
		}
		return createExerciseVocabularyLinks(tx, exercise.ID, vocabulary.ID, options)
	}); err != nil {
		return nil, err
	}

	return &RandomExerciseResult{
		ExerciseID:     exercise.ID,
		Type:           exerciseType,
		QuestionWord:   questionWord,
		Language:       language,
		AnswerLanguage: answerLanguage,
		Options:        ShuffledAnswerCharacters(answerWord),
	}, nil
}

func generateMatchPairsExercise(userID uint, when time.Time) (uuid.UUID, error) {
	seedIDs, err := getEligibleVocabularyIDs(userID, 1)
	if err != nil {
		return uuid.Nil, err
	}
	if len(seedIDs) == 0 {
		return uuid.Nil, errNoExerciseTypeAvailable
	}

	seedVocabulary, err := loadExerciseVocabulary(seedIDs[0])
	if err != nil {
		return uuid.Nil, err
	}

	options, err := buildMatchPairOptions(userID, seedVocabulary)
	if err != nil {
		return uuid.Nil, err
	}
	if len(options) != matchPairsVocabularyCount {
		return uuid.Nil, errNoExerciseTypeAvailable
	}

	exercise := models.Exercise{
		Type:         enums.ExerciseTypeMatchPairs,
		Status:       enums.ExerciseStatusPending,
		UserID:       userID,
		ScheduledFor: &when,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&exercise).Error; err != nil {
			return err
		}

		return createExerciseVocabularyLinks(tx, exercise.ID, uuid.Nil, options)
	})
	if err != nil {
		return uuid.Nil, err
	}

	return exercise.ID, nil
}
