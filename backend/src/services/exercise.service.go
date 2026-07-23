package services

import (
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"strings"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/models"
	"termorize/src/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNoVocabularyForExercise   = errors.New("no vocabulary found")
	ErrAllVocabularyMastered     = errors.New("all vocabulary is already mastered")
	ErrExerciseNotFound          = errors.New("exercise not found")
	ErrExerciseNotInProgress     = errors.New("exercise is not in progress")
	ErrExerciseVocabularyDeleted = errors.New("exercise vocabulary was deleted")
	ErrInvalidMatchPairResults   = errors.New("invalid match pair results")
	ErrInvalidCharacterResults   = errors.New("invalid character exercise results")
	errNoExerciseTypeAvailable   = errors.New("no exercise type available")
)

var webRussianYoReplacer = strings.NewReplacer("ё", "е", "Ё", "Е")

func normalizeAnswer(value string) string {
	return strings.ToLower(webRussianYoReplacer.Replace(strings.TrimSpace(value)))
}

func almostCorrectThreshold(expected string) int {
	if len([]rune(expected)) > 10 {
		return 2
	}
	return 1
}

const (
	ExerciseCompleteProgressDelta       = 15
	ExerciseAlmostCorrectProgressDelta  = 5
	ExerciseFailProgressDelta           = -20
	ExerciseChoiceCompleteProgressDelta = 5
	ExerciseChoiceFailProgressDelta     = -10
	ExerciseMatchCorrectProgressDelta   = 7
	ExerciseMatchAlmostProgressDelta    = 2
	ExerciseMatchFailProgressDelta      = -10
	exerciseReminderPeriod              = 24 * time.Hour
	telegramExerciseExpirationPeriod    = 7 * 24 * time.Hour
	websiteExerciseExpirationPeriod     = time.Hour
)

const (
	ExerciseVocabularyResultCorrect = "correct"
	ExerciseVocabularyResultAlmost  = "almost"
	ExerciseVocabularyResultWrong   = "wrong"
	ExerciseVocabularyResultIgnored = "ignored"

	ExerciseVocabularyResultReasonTypedAnswer       = "typed_answer"
	ExerciseVocabularyResultReasonCharacterAnswer   = "character_answer"
	ExerciseVocabularyResultReasonChoiceAnswer      = "choice_answer"
	ExerciseVocabularyResultReasonMatchPairs        = "match_pairs"
	ExerciseVocabularyResultReasonSkipped           = "skipped"
	ExerciseVocabularyResultReasonExpired           = "expired"
	ExerciseVocabularyResultReasonDeletedVocabulary = "deleted_vocabulary"
	ExerciseVocabularyResultReasonInvalidOptions    = "invalid_options"
)

const (
	choiceExerciseVocabularyCount = 4
	matchPairsVocabularyCount     = 5
	matchPairCardSideOriginal     = "original"
	matchPairCardSideTranslation  = "translation"

	basicExerciseWeight      = 35
	choiceExerciseWeight     = 35
	characterExerciseWeight  = 20
	matchPairsExerciseWeight = 10
)

const ChoiceExerciseVocabularyCount = choiceExerciseVocabularyCount

const MatchPairsVocabularyCount = matchPairsVocabularyCount

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

type PendingMatchExercise struct {
	ExerciseID     uuid.UUID      `gorm:"column:exercise_id"`
	UserID         uint           `gorm:"column:user_id"`
	Username       string         `gorm:"column:username"`
	TelegramID     int64          `gorm:"column:telegram_id"`
	SystemLanguage enums.Language `gorm:"column:system_language"`
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
	UserID              uint                 `gorm:"column:user_id"`
	Options             []ExerciseOption
	OriginalWord        string         `gorm:"column:original_word"`
	OriginalLanguage    enums.Language `gorm:"column:original_language"`
	TranslationWord     string         `gorm:"column:translation_word"`
	TranslationLanguage enums.Language `gorm:"column:translation_language"`
	Vocabulary          []models.Vocabulary
	CharacterBoard      *CharacterBoardState
}

type ExerciseOption struct {
	VocabularyID uuid.UUID `json:"vocabulary_id"`
	Label        string    `json:"label"`
}

type ExerciseMatchCard struct {
	ID           string         `json:"id"`
	VocabularyID uuid.UUID      `json:"vocabulary_id"`
	Word         string         `json:"word"`
	Language     enums.Language `json:"language"`
	Side         string         `json:"side"`
}

type CharacterBoardState struct {
	Order      []int
	Characters []string
	Chosen     []int
	Answer     string
}

type exerciseChoiceCandidate struct {
	VocabularyID uuid.UUID `gorm:"column:vocabulary_id"`
	AnswerWord   string    `gorm:"column:answer_word"`
}

type exerciseMatchPairCandidate struct {
	VocabularyID    uuid.UUID `gorm:"column:vocabulary_id"`
	OriginalWord    string    `gorm:"column:original_word"`
	TranslationWord string    `gorm:"column:translation_word"`
}

type exerciseVocabularyDetails struct {
	ExerciseID          uuid.UUID      `gorm:"column:exercise_id"`
	VocabularyID        uuid.UUID      `gorm:"column:vocabulary_id"`
	IsCorrect           bool           `gorm:"column:is_correct"`
	Position            int            `gorm:"column:position"`
	Result              *string        `gorm:"column:result"`
	ResultReason        *string        `gorm:"column:result_reason"`
	ProgressDelta       *int           `gorm:"column:progress_delta"`
	KnowledgeAfter      *int           `gorm:"column:knowledge_after"`
	AnsweredAt          *time.Time     `gorm:"column:answered_at"`
	VocabularyDeletedAt *time.Time     `gorm:"column:vocabulary_deleted_at"`
	OriginalWord        string         `gorm:"column:original_word"`
	OriginalLanguage    enums.Language `gorm:"column:original_language"`
	TranslationWord     string         `gorm:"column:translation_word"`
	TranslationLanguage enums.Language `gorm:"column:translation_language"`
}

type ExerciseStatistics struct {
	InProgress         int64                     `json:"in_progress" gorm:"column:in_progress"`
	Done               int64                     `json:"done" gorm:"column:done"`
	Failed             int64                     `json:"failed" gorm:"column:failed"`
	Ignored            int64                     `json:"ignored" gorm:"column:ignored"`
	ExerciseActivity   []ExerciseDailyActivity   `json:"exercise_activity" gorm:"-"`
	VocabularyActivity []VocabularyDailyActivity `json:"vocabulary_activity" gorm:"-"`
}

type ExerciseDailyActivity struct {
	Date      string `json:"date"`
	Completed int64  `json:"completed"`
	Failed    int64  `json:"failed"`
}

type VocabularyDailyActivity struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type ExerciseListExercise struct {
	ID                uuid.UUID                `json:"id"`
	Type              enums.ExerciseType       `json:"type"`
	Status            enums.ExerciseStatus     `json:"status"`
	StartedAt         *time.Time               `json:"starts_at"`
	FinishedAt        *time.Time               `json:"finishes_at"`
	TelegramMessageID *int64                   `json:"telegram_message_id"`
	Vocabulary        []ExerciseListVocabulary `json:"vocabularies"`
	LegacyVocabulary  *ExerciseListVocabulary  `json:"vocabulary,omitempty"`
}

type ExerciseListVocabulary struct {
	ID             uuid.UUID           `json:"id"`
	Translation    *models.Translation `json:"translation,omitempty"`
	ExerciseResult *string             `json:"exercise_result,omitempty"`
	ResultReason   *string             `json:"result_reason,omitempty"`
	ProgressDelta  *int                `json:"progress_delta,omitempty"`
	KnowledgeAfter *int                `json:"knowledge_after,omitempty"`
	AnsweredAt     *time.Time          `json:"answered_at,omitempty"`
	IsCorrect      bool                `json:"is_correct"`
	Position       int                 `json:"position"`
}

type ExerciseListResponse struct {
	Data       []ExerciseListExercise `json:"data"`
	Pagination Pagination             `json:"pagination"`
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
	vocabulary, err := loadExerciseVocabulary(vocabularyID)
	if err != nil {
		return err
	}

	exerciseType, options, err := selectExerciseTypeAndOptions(userID, vocabulary, includeMatchPairs)
	if err != nil {
		return err
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		exercise := models.Exercise{
			Type:         exerciseType,
			Status:       enums.ExerciseStatusPending,
			UserID:       userID,
			ScheduledFor: &when,
		}

		if err := tx.Create(&exercise).Error; err != nil {
			return err
		}

		correctVocabularyID := vocabularyID
		if exerciseType == enums.ExerciseTypeMatchPairs {
			correctVocabularyID = uuid.Nil
		}

		return createExerciseVocabularyLinks(tx, exercise.ID, correctVocabularyID, options)
	})
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
			translated.word AS translation_word,
			translated.language AS translation_language,
			u.settings->>'system_language' AS system_language
		FROM exercises AS e
		JOIN users AS u ON u.id = e.user_id
		JOIN vocabulary_exercises AS ve ON ve.exercise_id = e.id AND ve.is_correct = true
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE e.status = ?
			AND e.type IN (?, ?, ?, ?, ?, ?)
			AND e.scheduled_for <= ?
			AND u.settings->'telegram'->'bot_enabled' = ?
		ORDER BY e.scheduled_for ASC, e.created_at ASC
	`, enums.ExerciseStatusPending, enums.ExerciseTypeBasicDirect, enums.ExerciseTypeBasicReversed, enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed, enums.ExerciseTypeCharactersDirect, enums.ExerciseTypeCharactersReversed, now, true).Scan(&exercises).Error

	if err != nil {
		return nil, err
	}

	return exercises, nil
}

func GetDuePendingMatchExercises(now time.Time) ([]PendingMatchExercise, error) {
	var exercises []PendingMatchExercise

	err := db.DB.Raw(`
		SELECT
			e.id AS exercise_id,
			e.user_id AS user_id,
			u.username AS username,
			u.telegram_id AS telegram_id,
			u.settings->>'system_language' AS system_language
		FROM exercises AS e
		JOIN users AS u ON u.id = e.user_id
		WHERE e.status = ?
			AND e.type = ?
			AND e.scheduled_for <= ?
			AND u.settings->'telegram'->'bot_enabled' = ?
			AND (
				SELECT COUNT(*)
				FROM vocabulary_exercises AS ve
				JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
				WHERE ve.exercise_id = e.id AND ve.is_correct = true
			) = ?
		ORDER BY e.scheduled_for ASC, e.created_at ASC
	`, enums.ExerciseStatusPending, enums.ExerciseTypeMatchPairs, now, true, matchPairsVocabularyCount).Scan(&exercises).Error

	if err != nil {
		return nil, err
	}

	return exercises, nil
}

func buildCanonicalMatchCards(rows []exerciseVocabularyDetails) []ExerciseMatchCard {
	cards := make([]ExerciseMatchCard, 0, len(rows)*2)
	for _, row := range rows {
		cards = append(cards, ExerciseMatchCard{
			ID:           row.VocabularyID.String() + ":" + matchPairCardSideOriginal,
			VocabularyID: row.VocabularyID,
			Word:         row.OriginalWord,
			Language:     row.OriginalLanguage,
			Side:         matchPairCardSideOriginal,
		})
		cards = append(cards, ExerciseMatchCard{
			ID:           row.VocabularyID.String() + ":" + matchPairCardSideTranslation,
			VocabularyID: row.VocabularyID,
			Word:         row.TranslationWord,
			Language:     row.TranslationLanguage,
			Side:         matchPairCardSideTranslation,
		})
	}

	return cards
}

func BuildMatchBoard(exerciseID uuid.UUID) ([]ExerciseMatchCard, []int, error) {
	rows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, true)
	if err != nil {
		return nil, nil, err
	}
	if len(rows) != matchPairsVocabularyCount {
		return nil, nil, ErrExerciseVocabularyDeleted
	}

	cards := buildCanonicalMatchCards(rows)

	order := make([]int, len(cards))
	for i := range order {
		order[i] = i
	}
	rand.Shuffle(len(order), func(i, j int) {
		order[i], order[j] = order[j], order[i]
	})

	return cards, order, nil
}

func StartMatchExercise(exerciseID uuid.UUID, telegramMessageID int64, order []int) error {
	stateBytes, err := json.Marshal(matchStateJSON{Order: order, Pending: -1, Attempts: [][2]int{}})
	if err != nil {
		return err
	}

	result := db.DB.Model(&models.Exercise{}).
		Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusPending).
		Updates(map[string]any{
			"status":              enums.ExerciseStatusInProgress,
			"telegram_message_id": telegramMessageID,
			"started_at":          time.Now().UTC(),
			"match_state":         string(stateBytes),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrExerciseNotInProgress
	}

	return nil
}

func GetExerciseByTelegramMessage(telegramMessageID int64, telegramID int64) (*TelegramMessageExercise, error) {
	var exercise models.Exercise

	err := db.DB.
		Model(&models.Exercise{}).
		Joins("JOIN users AS u ON u.id = exercises.user_id").
		Where("exercises.telegram_message_id = ?", telegramMessageID).
		Where("u.telegram_id = ?", telegramID).
		First(&exercise).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return buildTelegramMessageExercise(exercise)
}

func GetExerciseByTelegramExerciseID(exerciseID uuid.UUID, telegramID int64) (*TelegramMessageExercise, error) {
	var exercise models.Exercise

	err := db.DB.
		Model(&models.Exercise{}).
		Joins("JOIN users AS u ON u.id = exercises.user_id").
		Where("exercises.id = ?", exerciseID).
		Where("u.telegram_id = ?", telegramID).
		First(&exercise).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return buildTelegramMessageExercise(exercise)
}

func buildTelegramMessageExercise(exercise models.Exercise) (*TelegramMessageExercise, error) {
	correctVocabulary, err := getCorrectExerciseVocabularyDetails(exercise.ID)
	if err != nil {
		return nil, err
	}

	options, err := GetExerciseAnswerOptions(exercise.ID, exercise.Type)
	if err != nil {
		return nil, err
	}

	telegramExercise := TelegramMessageExercise{
		ExerciseID:   exercise.ID,
		ExerciseType: exercise.Type,
		Status:       exercise.Status,
		UserID:       exercise.UserID,
		Options:      options,
	}

	if correctVocabulary != nil {
		telegramExercise.OriginalWord = correctVocabulary.OriginalWord
		telegramExercise.OriginalLanguage = correctVocabulary.OriginalLanguage
		telegramExercise.TranslationWord = correctVocabulary.TranslationWord
		telegramExercise.TranslationLanguage = correctVocabulary.TranslationLanguage
		telegramExercise.Vocabulary = []models.Vocabulary{buildVocabularyFromExerciseDetails(*correctVocabulary)}

		if isCharacterExerciseType(exercise.Type) && exercise.CharacterState != nil {
			answer := correctVocabulary.TranslationWord
			if isReversedExerciseType(exercise.Type) {
				answer = correctVocabulary.OriginalWord
			}
			characters := AnswerCharacters(answer)

			var state characterStateJSON
			if unmarshalErr := json.Unmarshal([]byte(*exercise.CharacterState), &state); unmarshalErr != nil {
				return nil, unmarshalErr
			}
			if !validCharacterState(state, len(characters)) {
				return nil, ErrInvalidCharacterResults
			}
			telegramExercise.CharacterBoard = buildCharacterBoardState(state.Order, characters, state.Chosen)
		}
	}

	return &telegramExercise, nil
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

func AnswerCharacters(answer string) []string {
	trimmed := strings.TrimSpace(answer)
	characters := make([]string, 0, len([]rune(trimmed)))
	for _, character := range []rune(trimmed) {
		characters = append(characters, string(character))
	}

	return characters
}

func ShuffledAnswerCharacters(answer string) []string {
	characters := AnswerCharacters(answer)
	rand.Shuffle(len(characters), func(i, j int) {
		characters[i], characters[j] = characters[j], characters[i]
	})

	return characters
}

func BuildCharacterBoardForAnswer(answer string) *CharacterBoardState {
	characters := AnswerCharacters(answer)
	order := make([]int, characterBoardSlotCount(len(characters)))
	for index := range characters {
		order[index] = index
	}
	for index := len(characters); index < len(order); index++ {
		order[index] = -1
	}
	rand.Shuffle(len(order), func(i, j int) {
		order[i], order[j] = order[j], order[i]
	})

	return buildCharacterBoardState(order, characters, nil)
}

func StartCharacterExercise(exerciseID uuid.UUID, telegramMessageID int64, order []int) error {
	stateBytes, err := json.Marshal(characterStateJSON{
		Order:  append([]int(nil), order...),
		Chosen: []int{},
	})
	if err != nil {
		return err
	}

	result := db.DB.Model(&models.Exercise{}).
		Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusPending).
		Updates(map[string]any{
			"status":              enums.ExerciseStatusInProgress,
			"telegram_message_id": telegramMessageID,
			"started_at":          time.Now().UTC(),
			"character_state":     string(stateBytes),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrExerciseNotInProgress
	}

	return nil
}

func ApplyCharacterTap(exerciseID uuid.UUID, userID uint, tappedIndex int) (*CharacterBoardState, bool, error) {
	var board *CharacterBoardState
	var finished bool
	var vocabularyDeleted bool

	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		var exercise models.Exercise
		if lockErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", exerciseID, userID).
			Take(&exercise).Error; lockErr != nil {
			if errors.Is(lockErr, gorm.ErrRecordNotFound) {
				return ErrExerciseNotFound
			}
			return lockErr
		}

		if !isCharacterExerciseType(exercise.Type) {
			return ErrInvalidCharacterResults
		}
		if exercise.Status != enums.ExerciseStatusInProgress {
			return ErrExerciseNotInProgress
		}

		correctVocabulary, detailErr := getCorrectExerciseVocabularyDetails(exercise.ID)
		if detailErr != nil {
			return detailErr
		}
		if correctVocabulary == nil {
			vocabularyDeleted = true
			return ErrExerciseVocabularyDeleted
		}

		answer := correctVocabulary.TranslationWord
		if isReversedExerciseType(exercise.Type) {
			answer = correctVocabulary.OriginalWord
		}
		characters := AnswerCharacters(answer)
		if len(characters) == 0 || tappedIndex < 0 || tappedIndex >= len(characters) {
			return ErrInvalidCharacterResults
		}

		if exercise.CharacterState == nil {
			return ErrInvalidCharacterResults
		}

		var state characterStateJSON
		if unmarshalErr := json.Unmarshal([]byte(*exercise.CharacterState), &state); unmarshalErr != nil {
			return unmarshalErr
		}
		if !validCharacterState(state, len(characters)) {
			return ErrInvalidCharacterResults
		}

		if len(state.Chosen) >= len(characters) || containsInt(state.Chosen, tappedIndex) {
			board = buildCharacterBoardState(state.Order, characters, state.Chosen)
			finished = len(state.Chosen) == len(characters)
			return nil
		}

		state.Chosen = append(state.Chosen, tappedIndex)
		stateBytes, marshalErr := json.Marshal(state)
		if marshalErr != nil {
			return marshalErr
		}

		updateResult := tx.Model(&models.Exercise{}).
			Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
			Update("character_state", string(stateBytes))
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return ErrExerciseNotInProgress
		}

		board = buildCharacterBoardState(state.Order, characters, state.Chosen)
		finished = len(state.Chosen) == len(characters)
		return nil
	})

	if txErr != nil {
		if vocabularyDeleted {
			_ = MarkExerciseVocabularyResultWithoutProgress(exerciseID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
			_ = IgnoreExercise(exerciseID)
		}
		return nil, false, txErr
	}

	return board, finished, nil
}

func ClearCharacterSelection(exerciseID uuid.UUID, userID uint) (*CharacterBoardState, error) {
	var board *CharacterBoardState
	var vocabularyDeleted bool

	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		var exercise models.Exercise
		if lockErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", exerciseID, userID).
			Take(&exercise).Error; lockErr != nil {
			if errors.Is(lockErr, gorm.ErrRecordNotFound) {
				return ErrExerciseNotFound
			}
			return lockErr
		}

		if !isCharacterExerciseType(exercise.Type) {
			return ErrInvalidCharacterResults
		}
		if exercise.Status != enums.ExerciseStatusInProgress {
			return ErrExerciseNotInProgress
		}

		correctVocabulary, detailErr := getCorrectExerciseVocabularyDetails(exercise.ID)
		if detailErr != nil {
			return detailErr
		}
		if correctVocabulary == nil {
			vocabularyDeleted = true
			return ErrExerciseVocabularyDeleted
		}

		answer := correctVocabulary.TranslationWord
		if isReversedExerciseType(exercise.Type) {
			answer = correctVocabulary.OriginalWord
		}
		characters := AnswerCharacters(answer)

		if exercise.CharacterState == nil {
			return ErrInvalidCharacterResults
		}
		var state characterStateJSON
		if unmarshalErr := json.Unmarshal([]byte(*exercise.CharacterState), &state); unmarshalErr != nil {
			return unmarshalErr
		}
		if !validCharacterState(state, len(characters)) {
			return ErrInvalidCharacterResults
		}

		state.Chosen = []int{}
		stateBytes, marshalErr := json.Marshal(state)
		if marshalErr != nil {
			return marshalErr
		}
		updateResult := tx.Model(&models.Exercise{}).
			Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
			Update("character_state", string(stateBytes))
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return ErrExerciseNotInProgress
		}

		board = buildCharacterBoardState(state.Order, characters, state.Chosen)
		return nil
	})

	if txErr != nil {
		if vocabularyDeleted {
			_ = MarkExerciseVocabularyResultWithoutProgress(exerciseID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
			_ = IgnoreExercise(exerciseID)
		}
		return nil, txErr
	}

	return board, nil
}

func characterBoardSide(characterCount int) int {
	if characterCount <= 0 {
		return 0
	}
	return int(math.Ceil(math.Sqrt(float64(characterCount + 1))))
}

func characterBoardSlotCount(characterCount int) int {
	side := characterBoardSide(characterCount)
	if side == 0 {
		return 0
	}
	return side*side - 1
}

func validCharacterState(state characterStateJSON, characterCount int) bool {
	if len(state.Order) != characterBoardSlotCount(characterCount) || len(state.Chosen) > characterCount {
		return false
	}

	seenOrder := make(map[int]bool, characterCount)
	for _, index := range state.Order {
		if index == -1 {
			continue
		}
		if index < 0 || index >= characterCount || seenOrder[index] {
			return false
		}
		seenOrder[index] = true
	}
	if len(seenOrder) != characterCount {
		return false
	}

	seenChosen := make(map[int]bool, len(state.Chosen))
	for _, index := range state.Chosen {
		if index < 0 || index >= characterCount || seenChosen[index] {
			return false
		}
		seenChosen[index] = true
	}

	return true
}

func containsInt(values []int, expected int) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func buildCharacterBoardState(order []int, characters []string, chosen []int) *CharacterBoardState {
	var answer strings.Builder
	for _, index := range chosen {
		if index >= 0 && index < len(characters) {
			answer.WriteString(characters[index])
		}
	}

	return &CharacterBoardState{
		Order:      append([]int(nil), order...),
		Characters: append([]string(nil), characters...),
		Chosen:     append([]int(nil), chosen...),
		Answer:     answer.String(),
	}
}

func DeletePendingExercisesByUserID(tx *gorm.DB, userID uint) error {
	return tx.Where("user_id = ? AND status = ?", userID, enums.ExerciseStatusPending).
		Delete(&models.Exercise{}).Error
}

func DeletePendingExercisesByVocabularyID(tx *gorm.DB, userID uint, vocabularyID uuid.UUID) error {
	return tx.
		Where("user_id = ? AND status = ?", userID, enums.ExerciseStatusPending).
		Where("id IN (?)",
			tx.Table("vocabulary_exercises").
				Select("exercise_id").
				Where("vocabulary_id = ?", vocabularyID),
		).
		Delete(&models.Exercise{}).Error
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

func IgnoreUserExercise(exerciseID uuid.UUID, userID uint) error {
	result := db.DB.Model(&models.Exercise{}).
		Where("id = ? AND user_id = ?", exerciseID, userID).
		Where("status IN ?", []enums.ExerciseStatus{enums.ExerciseStatusPending, enums.ExerciseStatusInProgress}).
		Updates(map[string]any{
			"status":      enums.ExerciseStatusIgnored,
			"finished_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}

	var count int64
	if err := db.DB.Model(&models.Exercise{}).
		Where("id = ? AND user_id = ?", exerciseID, userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrExerciseNotFound
	}

	return ErrExerciseNotInProgress
}

func IgnoreDuePendingExercisesWithoutActiveVocabulary(now time.Time) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			WITH affected AS (
				SELECT e.id
				FROM exercises AS e
				WHERE e.status = ?
					AND e.type IN (?, ?, ?, ?, ?, ?, ?)
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
			WHERE e.status = ?
				AND e.type IN (?, ?, ?, ?, ?, ?, ?)
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
		`, enums.ExerciseStatusIgnored, now, enums.ExerciseStatusPending, enums.ExerciseTypeBasicDirect, enums.ExerciseTypeBasicReversed, enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed, enums.ExerciseTypeCharactersDirect, enums.ExerciseTypeCharactersReversed, enums.ExerciseTypeMatchPairs, now, enums.ExerciseTypeMatchPairs, enums.ExerciseTypeMatchPairs, matchPairsVocabularyCount).Error
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
		ExerciseCompleteProgressDelta,
	)
}

func FinishExercise(exerciseID uuid.UUID, status enums.ExerciseStatus, result string, reason string, progressDelta int) (bool, int, error) {
	updated := false
	translationKnowledge := 0

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

		var updateErr error
		translationKnowledge, updateErr = updateVocabularyProgressByExercise(tx, exerciseID, result, reason, progressDelta)
		return updateErr
	})

	if err != nil {
		return false, 0, err
	}

	return updated, translationKnowledge, nil
}

func updateVocabularyProgressByExercise(tx *gorm.DB, exerciseID uuid.UUID, result string, reason string, delta int) (int, error) {
	var exerciseLink struct {
		VocabularyID uuid.UUID `gorm:"column:vocabulary_id"`
	}

	if err := tx.Table("vocabulary_exercises").
		Select("vocabulary_id").
		Where("exercise_id = ?", exerciseID).
		Where("is_correct = ?", true).
		Take(&exerciseLink).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}

		return 0, err
	}

	return updateVocabularyProgressByID(tx, exerciseID, exerciseLink.VocabularyID, result, reason, delta)
}

func updateVocabularyProgressByID(tx *gorm.DB, exerciseID uuid.UUID, vocabularyID uuid.UUID, result string, reason string, delta int) (int, error) {
	var vocabulary models.Vocabulary
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", vocabularyID).
		Where("deleted_at IS NULL").
		Take(&vocabulary).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}

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
		return 0, err
	}

	return translationKnowledge, nil
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

func GetExerciseWordsByTelegram(exerciseID uuid.UUID, telegramID int64) (*ExerciseWords, error) {
	var words ExerciseWords

	err := db.DB.Raw(`
		SELECT
			e.type AS exercise_type,
			original.word AS original_word,
			original.language AS original_language,
			translated.word AS translation_word,
			translated.language AS translation_language
		FROM exercises AS e
		JOIN users AS u ON u.id = e.user_id
		JOIN vocabulary_exercises AS ve ON ve.exercise_id = e.id AND ve.is_correct = true
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
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

	location := time.UTC
	var user models.User
	if err := db.DB.Select("settings").First(&user, userID).Error; err != nil {
		return nil, err
	}
	if user.Settings.TimeZone != "" {
		if configuredLocation, locationErr := time.LoadLocation(user.Settings.TimeZone); locationErr == nil {
			location = configuredLocation
		}
	}

	now := time.Now().In(location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	weekStart := today.AddDate(0, 0, -7)
	activityStart := time.Date(today.Year(), today.Month()-5, 1, 0, 0, 0, 0, location)
	rangeEnd := today.AddDate(0, 0, 1)

	type exerciseActivityRow struct {
		Status     enums.ExerciseStatus
		FinishedAt time.Time
	}
	var exerciseRows []exerciseActivityRow
	if err := db.DB.Model(&models.Exercise{}).
		Select("status, finished_at").
		Where("user_id = ?", userID).
		Where("status IN ?", []enums.ExerciseStatus{enums.ExerciseStatusCompleted, enums.ExerciseStatusFailed}).
		Where("finished_at >= ? AND finished_at < ?", weekStart.UTC(), rangeEnd.UTC()).
		Find(&exerciseRows).Error; err != nil {
		return nil, err
	}

	exerciseByDate := make(map[string]int, 8)
	statistics.ExerciseActivity = make([]ExerciseDailyActivity, 0, 8)
	for day := weekStart; !day.After(today); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		statistics.ExerciseActivity = append(statistics.ExerciseActivity, ExerciseDailyActivity{Date: date})
		exerciseByDate[date] = len(statistics.ExerciseActivity) - 1
	}
	for _, row := range exerciseRows {
		index, ok := exerciseByDate[row.FinishedAt.In(location).Format("2006-01-02")]
		if !ok {
			continue
		}
		if row.Status == enums.ExerciseStatusCompleted {
			statistics.ExerciseActivity[index].Completed++
		} else {
			statistics.ExerciseActivity[index].Failed++
		}
	}

	type vocabularyActivityRow struct {
		CreatedAt time.Time
	}
	var vocabularyRows []vocabularyActivityRow
	if err := db.DB.Raw(`
		SELECT created_at
		FROM vocabulary
		WHERE user_id = ? AND created_at >= ? AND created_at < ?
	`, userID, activityStart.UTC(), rangeEnd.UTC()).Scan(&vocabularyRows).Error; err != nil {
		return nil, err
	}

	vocabularyByDate := make(map[string]int)
	statistics.VocabularyActivity = make([]VocabularyDailyActivity, 0, 186)
	for day := activityStart; !day.After(today); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		statistics.VocabularyActivity = append(statistics.VocabularyActivity, VocabularyDailyActivity{Date: date})
		vocabularyByDate[date] = len(statistics.VocabularyActivity) - 1
	}
	for _, row := range vocabularyRows {
		index, ok := vocabularyByDate[row.CreatedAt.In(location).Format("2006-01-02")]
		if ok {
			statistics.VocabularyActivity[index].Count++
		}
	}

	return &statistics, nil
}

func GetExercises(userID uint, page, pageSize int) (*ExerciseListResponse, error) {
	if page <= 0 {
		return nil, ErrInvalidPage
	}

	if pageSize < 1 || pageSize > 1000 {
		return nil, ErrInvalidPageSize
	}

	totalQuery := db.DB.Model(&models.Exercise{}).
		Where("user_id = ?", userID).
		Where("started_at IS NOT NULL")

	var total int64
	if err := totalQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	exercises := make([]models.Exercise, 0, pageSize)

	if err := db.DB.
		Model(&models.Exercise{}).
		Where("user_id = ?", userID).
		Where("started_at IS NOT NULL").
		Order("started_at DESC, id DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&exercises).Error; err != nil {
		return nil, err
	}

	vocabularyByExerciseID, err := loadListVocabularyByExerciseIDs(collectExerciseIDs(exercises))
	if err != nil {
		return nil, err
	}

	data := make([]ExerciseListExercise, 0, len(exercises))
	for _, exerciseModel := range exercises {
		exercise := ExerciseListExercise{
			ID:                exerciseModel.ID,
			Type:              exerciseModel.Type,
			Status:            exerciseModel.Status,
			StartedAt:         exerciseModel.StartedAt,
			FinishedAt:        exerciseModel.FinishedAt,
			TelegramMessageID: exerciseModel.TelegramMessageID,
		}

		if vocabulary, ok := vocabularyByExerciseID[exerciseModel.ID]; ok {
			exercise.Vocabulary = vocabulary
			if len(vocabulary) > 0 {
				legacyVocabulary := vocabulary[0]
				exercise.LegacyVocabulary = &legacyVocabulary
			}
		}

		data = append(data, exercise)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return &ExerciseListResponse{
		Data: data,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

type RandomExerciseResult struct {
	ExerciseID     uuid.UUID
	Type           enums.ExerciseType
	QuestionWord   string
	Language       enums.Language
	AnswerLanguage enums.Language
	Options        []string
	Cards          []ExerciseMatchCard
}

type MatchPairAttempt struct {
	FirstCardID  string
	SecondCardID string
}

type MatchPairsCompleteResult struct {
	Status  enums.ExerciseStatus     `json:"status"`
	Results []ExerciseListVocabulary `json:"results"`
}

type matchStateJSON struct {
	Order    []int    `json:"order"`
	Pending  int      `json:"pending"`
	Attempts [][2]int `json:"attempts"`
}

type characterStateJSON struct {
	Order  []int `json:"order"`
	Chosen []int `json:"chosen"`
}

type MatchBoardState struct {
	Order        []int                // display permutation of canonical card indices
	Cards        []ExerciseMatchCard  // canonical order; index == canonical card index
	Pending      int                  // canonical index of held first pick, or -1
	Resolved     map[uuid.UUID]string // vocabulary -> result (correct/almost/wrong)
	CardWrong    map[string]int       // card.ID -> wrong attempts
	MatchedCount int                  // number of resolved vocabularies
}

func CreateRandomExercise(userID uint) (*RandomExerciseResult, error) {
	ids, err := getEligibleVocabularyIDs(userID, 64)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		hasVocabulary, err := userHasVocabulary(userID)
		if err != nil {
			return nil, err
		}

		if !hasVocabulary {
			return nil, ErrNoVocabularyForExercise
		}

		return nil, ErrAllVocabularyMastered
	}

	for _, vocabularyID := range ids {
		result, createErr := createRandomExerciseForVocabulary(userID, vocabularyID)
		if errors.Is(createErr, errNoExerciseTypeAvailable) {
			continue
		}

		if createErr != nil {
			return nil, createErr
		}

		return result, nil
	}

	return nil, ErrNoVocabularyForExercise
}

func userHasVocabulary(userID uint) (bool, error) {
	var count int64

	err := db.DB.
		Model(&models.Vocabulary{}).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

type VerifyAnswerResult struct {
	Result        string
	CorrectAnswer string
	Knowledge     int
	ProgressDelta int
}

func VerifyExerciseAnswer(exerciseID uuid.UUID, userID uint, answer string) (*VerifyAnswerResult, error) {
	exercise, correctVocabulary, err := getExerciseWithCorrectVocabulary(exerciseID, userID)
	if err != nil {
		return nil, err
	}

	if exercise.Status != enums.ExerciseStatusInProgress {
		return nil, ErrExerciseNotInProgress
	}

	if isMatchPairsExerciseType(exercise.Type) {
		return nil, ErrInvalidMatchPairResults
	}

	if correctVocabulary == nil {
		_ = MarkExerciseVocabularyResultWithoutProgress(exercise.ID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
		_ = IgnoreExercise(exercise.ID)
		return nil, ErrExerciseVocabularyDeleted
	}

	expectedAnswer := correctVocabulary.TranslationWord
	if isReversedExerciseType(exercise.Type) {
		expectedAnswer = correctVocabulary.OriginalWord
	}

	normalizedAnswer := normalizeAnswer(answer)
	normalizedExpectedAnswer := normalizeAnswer(expectedAnswer)

	var updated bool
	var knowledge int
	var progressDelta int
	var resultType string

	if isChoiceExerciseType(exercise.Type) {
		options, optionsErr := GetExerciseAnswerOptions(exercise.ID, exercise.Type)
		if optionsErr != nil {
			return nil, optionsErr
		}
		if len(options) != 4 {
			_ = MarkExerciseVocabularyResultWithoutProgress(exercise.ID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonInvalidOptions)
			_ = IgnoreExercise(exercise.ID)
			return nil, ErrExerciseVocabularyDeleted
		}

		if normalizedAnswer == normalizedExpectedAnswer {
			progressDelta = ExerciseChoiceCompleteProgressDelta
			updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultCorrect, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
			resultType = "correct"
		} else if exerciseOptionsContainAnswer(options, normalizedAnswer) {
			progressDelta = ExerciseChoiceFailProgressDelta
			updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusFailed, ExerciseVocabularyResultWrong, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
			resultType = "wrong"
		} else {
			progressDelta = ExerciseChoiceFailProgressDelta
			updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusFailed, ExerciseVocabularyResultWrong, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
			resultType = "wrong"
		}
	} else {
		answerReason := ExerciseVocabularyResultReasonTypedAnswer
		if isCharacterExerciseType(exercise.Type) {
			answerReason = ExerciseVocabularyResultReasonCharacterAnswer
		}

		if normalizedAnswer == normalizedExpectedAnswer {
			progressDelta = ExerciseCompleteProgressDelta
			updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultCorrect, answerReason, progressDelta)
			resultType = "correct"
		} else {
			distance := utils.LevenshteinDistance(normalizedAnswer, normalizedExpectedAnswer)
			threshold := almostCorrectThreshold(normalizedExpectedAnswer)
			if distance <= threshold {
				progressDelta = ExerciseAlmostCorrectProgressDelta
				updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultAlmost, answerReason, progressDelta)
				resultType = "almost"
			} else {
				progressDelta = ExerciseFailProgressDelta
				updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusFailed, ExerciseVocabularyResultWrong, answerReason, progressDelta)
				resultType = "wrong"
			}
		}
	}

	if err != nil {
		return nil, err
	}

	if !updated {
		return nil, ErrExerciseNotInProgress
	}

	return &VerifyAnswerResult{
		Result:        resultType,
		CorrectAnswer: expectedAnswer,
		Knowledge:     knowledge,
		ProgressDelta: progressDelta,
	}, nil
}

func VerifyExerciseChoice(exerciseID uuid.UUID, userID uint, selectedVocabularyID uuid.UUID) (*VerifyAnswerResult, error) {
	exercise, correctVocabulary, err := getExerciseWithCorrectVocabulary(exerciseID, userID)
	if err != nil {
		return nil, err
	}

	if exercise.Status != enums.ExerciseStatusInProgress {
		return nil, ErrExerciseNotInProgress
	}

	if isMatchPairsExerciseType(exercise.Type) {
		return nil, ErrInvalidMatchPairResults
	}

	if correctVocabulary == nil {
		_ = MarkExerciseVocabularyResultWithoutProgress(exercise.ID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
		_ = IgnoreExercise(exercise.ID)
		return nil, ErrExerciseVocabularyDeleted
	}

	options, err := GetExerciseAnswerOptions(exercise.ID, exercise.Type)
	if err != nil {
		return nil, err
	}
	if len(options) != 4 {
		_ = MarkExerciseVocabularyResultWithoutProgress(exercise.ID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonInvalidOptions)
		_ = IgnoreExercise(exercise.ID)
		return nil, ErrExerciseVocabularyDeleted
	}

	correctAnswer := correctVocabulary.TranslationWord
	if isReversedExerciseType(exercise.Type) {
		correctAnswer = correctVocabulary.OriginalWord
	}

	var updated bool
	var knowledge int
	var progressDelta int
	var resultType string

	if selectedVocabularyID == correctVocabulary.VocabularyID {
		progressDelta = ExerciseChoiceCompleteProgressDelta
		updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultCorrect, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
		resultType = "correct"
	} else {
		progressDelta = ExerciseChoiceFailProgressDelta
		updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusFailed, ExerciseVocabularyResultWrong, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
		resultType = "wrong"
	}

	if err != nil {
		return nil, err
	}

	if !updated {
		return nil, ErrExerciseNotInProgress
	}

	return &VerifyAnswerResult{
		Result:        resultType,
		CorrectAnswer: correctAnswer,
		Knowledge:     knowledge,
		ProgressDelta: progressDelta,
	}, nil
}

func replayMatchAttempts(
	cardByID map[string]ExerciseMatchCard,
	expectedVocabularyIDs []uuid.UUID,
	attempts []MatchPairAttempt,
) (states map[uuid.UUID]string, cardWrong map[string]int, err error) {
	states = make(map[uuid.UUID]string, len(expectedVocabularyIDs))
	cardWrong = make(map[string]int, len(expectedVocabularyIDs)*2)
	for _, vocabularyID := range expectedVocabularyIDs {
		states[vocabularyID] = ""
	}

	for _, attempt := range attempts {
		if attempt.FirstCardID == attempt.SecondCardID {
			return nil, nil, ErrInvalidMatchPairResults
		}

		firstCard, ok := cardByID[attempt.FirstCardID]
		if !ok {
			return nil, nil, ErrInvalidMatchPairResults
		}
		secondCard, ok := cardByID[attempt.SecondCardID]
		if !ok {
			return nil, nil, ErrInvalidMatchPairResults
		}

		if states[firstCard.VocabularyID] != "" || states[secondCard.VocabularyID] != "" {
			return nil, nil, ErrInvalidMatchPairResults
		}

		isCorrectPair := firstCard.VocabularyID == secondCard.VocabularyID && firstCard.Side != secondCard.Side
		if isCorrectPair {
			result := ExerciseVocabularyResultCorrect
			if cardWrong[firstCard.ID] > 0 || cardWrong[secondCard.ID] > 0 {
				result = ExerciseVocabularyResultAlmost
			}
			states[firstCard.VocabularyID] = result
			continue
		}

		for _, card := range []ExerciseMatchCard{firstCard, secondCard} {
			cardWrong[card.ID]++
			if cardWrong[card.ID] >= 2 {
				states[card.VocabularyID] = ExerciseVocabularyResultWrong
			}
		}
	}

	return states, cardWrong, nil
}

func CompleteMatchPairsExercise(exerciseID uuid.UUID, userID uint, attempts []MatchPairAttempt) (*MatchPairsCompleteResult, error) {
	if len(attempts) == 0 {
		return nil, ErrInvalidMatchPairResults
	}

	var exercise models.Exercise
	if err := db.DB.
		Where("id = ? AND user_id = ?", exerciseID, userID).
		Take(&exercise).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExerciseNotFound
		}

		return nil, err
	}

	if exercise.Type != enums.ExerciseTypeMatchPairs {
		return nil, ErrInvalidMatchPairResults
	}

	if exercise.Status != enums.ExerciseStatusInProgress {
		return nil, ErrExerciseNotInProgress
	}

	rows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, true)
	if err != nil {
		return nil, err
	}
	if len(rows) != matchPairsVocabularyCount {
		_ = MarkExerciseVocabularyResultWithoutProgress(exercise.ID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
		_ = IgnoreExercise(exercise.ID)
		return nil, ErrExerciseVocabularyDeleted
	}

	expected := make(map[uuid.UUID]exerciseVocabularyDetails, len(rows))
	for _, row := range rows {
		expected[row.VocabularyID] = row
	}

	cardByID := make(map[string]ExerciseMatchCard, len(rows)*2)
	for _, row := range rows {
		originalCard := ExerciseMatchCard{
			ID:           row.VocabularyID.String() + ":" + matchPairCardSideOriginal,
			VocabularyID: row.VocabularyID,
			Word:         row.OriginalWord,
			Language:     row.OriginalLanguage,
			Side:         matchPairCardSideOriginal,
		}
		translationCard := ExerciseMatchCard{
			ID:           row.VocabularyID.String() + ":" + matchPairCardSideTranslation,
			VocabularyID: row.VocabularyID,
			Word:         row.TranslationWord,
			Language:     row.TranslationLanguage,
			Side:         matchPairCardSideTranslation,
		}
		cardByID[originalCard.ID] = originalCard
		cardByID[translationCard.ID] = translationCard
	}

	expectedVocabularyIDs := make([]uuid.UUID, 0, len(expected))
	for vocabularyID := range expected {
		expectedVocabularyIDs = append(expectedVocabularyIDs, vocabularyID)
	}

	states, _, err := replayMatchAttempts(cardByID, expectedVocabularyIDs, attempts)
	if err != nil {
		return nil, err
	}

	submitted := make(map[uuid.UUID]string, len(expected))
	hasWrong := false
	for vocabularyID, result := range states {
		if result == "" {
			return nil, ErrInvalidMatchPairResults
		}

		switch result {
		case ExerciseVocabularyResultCorrect, ExerciseVocabularyResultAlmost:
		case ExerciseVocabularyResultWrong:
			hasWrong = true
		default:
			return nil, ErrInvalidMatchPairResults
		}

		submitted[vocabularyID] = result
	}

	status := enums.ExerciseStatusCompleted
	if hasWrong {
		status = enums.ExerciseStatusFailed
	}

	var completedRows []ExerciseListVocabulary
	err = db.DB.Transaction(func(tx *gorm.DB) error {
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
			return ErrExerciseNotInProgress
		}

		for vocabularyID, result := range submitted {
			delta := ExerciseMatchCorrectProgressDelta
			if result == ExerciseVocabularyResultAlmost {
				delta = ExerciseMatchAlmostProgressDelta
			}
			if result == ExerciseVocabularyResultWrong {
				delta = ExerciseMatchFailProgressDelta
			}

			if _, updateErr := updateVocabularyProgressByID(tx, exerciseID, vocabularyID, result, ExerciseVocabularyResultReasonMatchPairs, delta); updateErr != nil {
				return updateErr
			}
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, ErrExerciseNotInProgress) {
			return nil, ErrExerciseNotInProgress
		}

		return nil, err
	}

	updatedRows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, false)
	if err != nil {
		return nil, err
	}
	completedRows = make([]ExerciseListVocabulary, 0, len(updatedRows))
	for _, row := range updatedRows {
		completedRows = append(completedRows, buildListVocabularyFromExerciseDetails(row))
	}

	return &MatchPairsCompleteResult{
		Status:  status,
		Results: completedRows,
	}, nil
}

func GetCompletedMatchPairsResult(exerciseID uuid.UUID, userID uint) (*MatchPairsCompleteResult, error) {
	var exercise models.Exercise
	if err := db.DB.
		Where("id = ? AND user_id = ?", exerciseID, userID).
		Take(&exercise).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExerciseNotFound
		}
		return nil, err
	}

	if exercise.Type != enums.ExerciseTypeMatchPairs {
		return nil, ErrInvalidMatchPairResults
	}
	if exercise.Status != enums.ExerciseStatusCompleted && exercise.Status != enums.ExerciseStatusFailed {
		return nil, ErrExerciseNotInProgress
	}

	rows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, false)
	if err != nil {
		return nil, err
	}

	results := make([]ExerciseListVocabulary, 0, len(rows))
	for _, row := range rows {
		results = append(results, buildListVocabularyFromExerciseDetails(row))
	}

	return &MatchPairsCompleteResult{
		Status:  exercise.Status,
		Results: results,
	}, nil
}

func ApplyMatchTap(exerciseID uuid.UUID, userID uint, tappedIdx int) (
	board *MatchBoardState, wasWrong bool, finished bool, finalizeAttempts []MatchPairAttempt, err error,
) {
	var vocabularyDeleted bool

	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		var exercise models.Exercise
		if lockErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", exerciseID, userID).
			Take(&exercise).Error; lockErr != nil {
			if errors.Is(lockErr, gorm.ErrRecordNotFound) {
				return ErrExerciseNotFound
			}
			return lockErr
		}

		if exercise.Type != enums.ExerciseTypeMatchPairs {
			return ErrInvalidMatchPairResults
		}
		if exercise.Status != enums.ExerciseStatusInProgress {
			return ErrExerciseNotInProgress
		}

		rows, detailErr := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, true)
		if detailErr != nil {
			return detailErr
		}
		if len(rows) != matchPairsVocabularyCount {
			vocabularyDeleted = true
			return ErrExerciseVocabularyDeleted
		}

		cards := buildCanonicalMatchCards(rows)
		if tappedIdx < 0 || tappedIdx >= len(cards) {
			return ErrInvalidMatchPairResults
		}

		cardByID := make(map[string]ExerciseMatchCard, len(cards))
		for _, card := range cards {
			cardByID[card.ID] = card
		}

		expectedVocabularyIDs := make([]uuid.UUID, 0, len(rows))
		for _, row := range rows {
			expectedVocabularyIDs = append(expectedVocabularyIDs, row.VocabularyID)
		}

		var state matchStateJSON
		if exercise.MatchState == nil {
			return ErrInvalidMatchPairResults
		}
		if unmarshalErr := json.Unmarshal([]byte(*exercise.MatchState), &state); unmarshalErr != nil {
			return unmarshalErr
		}

		attemptsToPairs := func(attempts [][2]int) []MatchPairAttempt {
			pairs := make([]MatchPairAttempt, 0, len(attempts))
			for _, attempt := range attempts {
				if attempt[0] < 0 || attempt[0] >= len(cards) || attempt[1] < 0 || attempt[1] >= len(cards) {
					continue
				}
				pairs = append(pairs, MatchPairAttempt{
					FirstCardID:  cards[attempt[0]].ID,
					SecondCardID: cards[attempt[1]].ID,
				})
			}
			return pairs
		}

		states, cardWrong, replayErr := replayMatchAttempts(cardByID, expectedVocabularyIDs, attemptsToPairs(state.Attempts))
		if replayErr != nil {
			return replayErr
		}

		tappedCard := cards[tappedIdx]

		wasWrong = false
		if states[tappedCard.VocabularyID] != "" {
			board = buildMatchBoardState(state.Order, cards, state.Pending, states, cardWrong)
			finished = isMatchFinished(states)
			if finished {
				finalizeAttempts = attemptsToPairs(state.Attempts)
			}
			return nil
		}

		switch {
		case state.Pending == -1:
			state.Pending = tappedIdx
		case state.Pending == tappedIdx:
			state.Pending = -1
		default:
			resolvedCorrectBefore := countResolvedNonWrong(states)
			state.Attempts = append(state.Attempts, [2]int{state.Pending, tappedIdx})
			state.Pending = -1

			states, cardWrong, replayErr = replayMatchAttempts(cardByID, expectedVocabularyIDs, attemptsToPairs(state.Attempts))
			if replayErr != nil {
				return replayErr
			}
			wasWrong = countResolvedNonWrong(states) == resolvedCorrectBefore
		}

		stateBytes, marshalErr := json.Marshal(state)
		if marshalErr != nil {
			return marshalErr
		}

		updateResult := tx.Model(&models.Exercise{}).
			Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
			Update("match_state", string(stateBytes))
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return ErrExerciseNotInProgress
		}

		board = buildMatchBoardState(state.Order, cards, state.Pending, states, cardWrong)
		finished = isMatchFinished(states)
		if finished {
			finalizeAttempts = attemptsToPairs(state.Attempts)
		}

		return nil
	})

	if txErr != nil {
		if vocabularyDeleted {
			_ = MarkExerciseVocabularyResultWithoutProgress(exerciseID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
			_ = IgnoreExercise(exerciseID)
		}
		return nil, false, false, nil, txErr
	}

	return board, wasWrong, finished, finalizeAttempts, nil
}

func buildMatchBoardState(order []int, cards []ExerciseMatchCard, pending int, states map[uuid.UUID]string, cardWrong map[string]int) *MatchBoardState {
	resolved := make(map[uuid.UUID]string)
	for vocabularyID, result := range states {
		if result != "" {
			resolved[vocabularyID] = result
		}
	}

	return &MatchBoardState{
		Order:        order,
		Cards:        cards,
		Pending:      pending,
		Resolved:     resolved,
		CardWrong:    cardWrong,
		MatchedCount: len(resolved),
	}
}

func isMatchFinished(states map[uuid.UUID]string) bool {
	if len(states) == 0 {
		return false
	}
	for _, result := range states {
		if result == "" {
			return false
		}
	}
	return true
}

func countResolvedNonWrong(states map[uuid.UUID]string) int {
	count := 0
	for _, result := range states {
		if result == ExerciseVocabularyResultCorrect || result == ExerciseVocabularyResultAlmost {
			count++
		}
	}
	return count
}

func createRandomExerciseForVocabulary(userID uint, vocabularyID uuid.UUID) (*RandomExerciseResult, error) {
	vocabulary, err := loadExerciseVocabulary(vocabularyID)
	if err != nil {
		return nil, err
	}

	exerciseType, options, err := selectExerciseTypeAndOptions(userID, vocabulary, true)
	if err != nil {
		return nil, err
	}

	if exerciseType == enums.ExerciseTypeMatchPairs {
		return createRandomMatchPairsExercise(userID, options)
	}

	questionWord, language, answerLanguage, err := buildExerciseQuestionData(vocabulary, exerciseType)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	exercise := models.Exercise{
		Type:      exerciseType,
		Status:    enums.ExerciseStatusInProgress,
		UserID:    userID,
		StartedAt: &now,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&exercise).Error; err != nil {
			return err
		}

		return createExerciseVocabularyLinks(tx, exercise.ID, vocabularyID, options)
	})
	if err != nil {
		return nil, err
	}

	resultOptions := collectExerciseOptionLabels(options)
	if isCharacterExerciseType(exerciseType) && len(options) > 0 {
		resultOptions = ShuffledAnswerCharacters(options[0].AnswerWord)
	}

	return &RandomExerciseResult{
		ExerciseID:     exercise.ID,
		Type:           exerciseType,
		QuestionWord:   questionWord,
		Language:       language,
		AnswerLanguage: answerLanguage,
		Options:        resultOptions,
	}, nil
}

func createRandomMatchPairsExercise(userID uint, options []exerciseChoiceCandidate) (*RandomExerciseResult, error) {
	if len(options) != matchPairsVocabularyCount {
		return nil, errNoExerciseTypeAvailable
	}

	now := time.Now().UTC()
	exercise := models.Exercise{
		Type:      enums.ExerciseTypeMatchPairs,
		Status:    enums.ExerciseStatusInProgress,
		UserID:    userID,
		StartedAt: &now,
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&exercise).Error; err != nil {
			return err
		}

		return createExerciseVocabularyLinks(tx, exercise.ID, uuid.Nil, options)
	})
	if err != nil {
		return nil, err
	}

	cards, err := GetExerciseMatchCards(exercise.ID)
	if err != nil {
		return nil, err
	}

	return &RandomExerciseResult{
		ExerciseID: exercise.ID,
		Type:       enums.ExerciseTypeMatchPairs,
		Cards:      cards,
	}, nil
}

func loadExerciseVocabulary(vocabularyID uuid.UUID) (*models.Vocabulary, error) {
	var vocabulary models.Vocabulary

	err := db.DB.
		Where("id = ?", vocabularyID).
		Where("deleted_at IS NULL").
		Preload("Translation").
		Preload("Translation.Original").
		Preload("Translation.Translation").
		Take(&vocabulary).Error
	if err != nil {
		return nil, err
	}

	return &vocabulary, nil
}

func selectExerciseTypeAndOptions(userID uint, vocabulary *models.Vocabulary, includeMatchPairs bool) (enums.ExerciseType, []exerciseChoiceCandidate, error) {
	availableTypes, err := buildExerciseOptionsByType(userID, vocabulary, includeMatchPairs)
	if err != nil {
		return "", nil, err
	}

	type exerciseTypeGroup struct {
		weight int
		types  []enums.ExerciseType
	}

	groups := []exerciseTypeGroup{
		{
			weight: basicExerciseWeight,
			types: []enums.ExerciseType{
				enums.ExerciseTypeBasicDirect,
				enums.ExerciseTypeBasicReversed,
			},
		},
		{
			weight: choiceExerciseWeight,
			types: []enums.ExerciseType{
				enums.ExerciseTypeChoiceDirect,
				enums.ExerciseTypeChoiceReversed,
			},
		},
		{
			weight: characterExerciseWeight,
			types: []enums.ExerciseType{
				enums.ExerciseTypeCharactersDirect,
				enums.ExerciseTypeCharactersReversed,
			},
		},
	}
	if includeMatchPairs {
		groups = append(groups, exerciseTypeGroup{
			weight: matchPairsExerciseWeight,
			types:  []enums.ExerciseType{enums.ExerciseTypeMatchPairs},
		})
	}

	availableGroups := make([]exerciseTypeGroup, 0, len(groups))
	totalWeight := 0
	for _, group := range groups {
		availableTypesInGroup := make([]enums.ExerciseType, 0, len(group.types))
		for _, exerciseType := range group.types {
			options := availableTypes[exerciseType]
			if (isChoiceExerciseType(exerciseType) && len(options) == choiceExerciseVocabularyCount) ||
				(isMatchPairsExerciseType(exerciseType) && len(options) == matchPairsVocabularyCount) ||
				(!isChoiceExerciseType(exerciseType) && !isMatchPairsExerciseType(exerciseType) && len(options) > 0) {
				availableTypesInGroup = append(availableTypesInGroup, exerciseType)
			}
		}
		if len(availableTypesInGroup) > 0 {
			group.types = availableTypesInGroup
			availableGroups = append(availableGroups, group)
			totalWeight += group.weight
		}
	}

	if len(availableGroups) == 0 {
		return "", nil, errNoExerciseTypeAvailable
	}

	roll := rand.Intn(totalWeight)
	selectedGroup := availableGroups[len(availableGroups)-1]
	for _, group := range availableGroups {
		if roll < group.weight {
			selectedGroup = group
			break
		}
		roll -= group.weight
	}

	exerciseType := selectedGroup.types[rand.Intn(len(selectedGroup.types))]
	return exerciseType, append([]exerciseChoiceCandidate(nil), availableTypes[exerciseType]...), nil
}

func buildExerciseOptionsByType(userID uint, vocabulary *models.Vocabulary, includeMatchPairs bool) (map[enums.ExerciseType][]exerciseChoiceCandidate, error) {
	optionsByType := make(map[enums.ExerciseType][]exerciseChoiceCandidate, 7)

	directOptions, err := buildExerciseOptionsForType(userID, vocabulary, enums.ExerciseTypeBasicDirect)
	if err != nil {
		return nil, err
	}
	if len(directOptions) > 0 {
		optionsByType[enums.ExerciseTypeBasicDirect] = append([]exerciseChoiceCandidate(nil), directOptions[:1]...)
		if strings.TrimSpace(directOptions[0].AnswerWord) != "" {
			optionsByType[enums.ExerciseTypeCharactersDirect] = append([]exerciseChoiceCandidate(nil), directOptions[:1]...)
		}
	}
	if len(directOptions) == choiceExerciseVocabularyCount {
		optionsByType[enums.ExerciseTypeChoiceDirect] = shuffledExerciseOptions(directOptions)
	}

	reversedOptions, err := buildExerciseOptionsForType(userID, vocabulary, enums.ExerciseTypeBasicReversed)
	if err != nil {
		return nil, err
	}
	if len(reversedOptions) > 0 {
		optionsByType[enums.ExerciseTypeBasicReversed] = append([]exerciseChoiceCandidate(nil), reversedOptions[:1]...)
		if strings.TrimSpace(reversedOptions[0].AnswerWord) != "" {
			optionsByType[enums.ExerciseTypeCharactersReversed] = append([]exerciseChoiceCandidate(nil), reversedOptions[:1]...)
		}
	}
	if len(reversedOptions) == choiceExerciseVocabularyCount {
		optionsByType[enums.ExerciseTypeChoiceReversed] = shuffledExerciseOptions(reversedOptions)
	}

	if includeMatchPairs {
		matchOptions, err := buildMatchPairOptions(userID, vocabulary)
		if err != nil {
			return nil, err
		}
		if len(matchOptions) == matchPairsVocabularyCount {
			optionsByType[enums.ExerciseTypeMatchPairs] = shuffledExerciseOptions(matchOptions)
		}
	}

	return optionsByType, nil
}

func buildMatchPairOptions(userID uint, vocabulary *models.Vocabulary) ([]exerciseChoiceCandidate, error) {
	if vocabulary == nil || vocabulary.Translation == nil {
		return nil, errors.New("vocabulary has no translation")
	}

	translation := vocabulary.Translation
	if translation.Original == nil {
		return nil, errors.New("vocabulary has no original word")
	}
	if translation.Translation == nil {
		return nil, errors.New("vocabulary has no translation word")
	}

	query := `
		SELECT
			v.id AS vocabulary_id,
			original.word AS original_word,
			translated.word AS translation_word
		FROM vocabulary AS v
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE v.user_id = ?
			AND v.deleted_at IS NULL
			AND v.mastered_at IS NULL
			AND original.language = ?
			AND translated.language = ?
			AND EXISTS (
				SELECT 1
				FROM jsonb_array_elements(v.progress) AS p
				WHERE p->>'type' = ? AND (p->>'knowledge')::int < ?
			)
		ORDER BY RANDOM()
	`

	var rows []exerciseMatchPairCandidate
	if err := db.DB.Raw(
		query,
		userID,
		translation.Original.Language,
		translation.Translation.Language,
		enums.KnowledgeTypeTranslation,
		100,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]exerciseMatchPairCandidate, len(rows))
	for _, row := range rows {
		byID[row.VocabularyID] = row
	}

	selected, ok := byID[vocabulary.ID]
	if !ok {
		return []exerciseChoiceCandidate{}, nil
	}

	seenOriginal := map[string]struct{}{
		normalizeAnswer(selected.OriginalWord): {},
	}
	seenTranslation := map[string]struct{}{
		normalizeAnswer(selected.TranslationWord): {},
	}
	selectedRows := []exerciseMatchPairCandidate{selected}

	for _, row := range rows {
		if row.VocabularyID == vocabulary.ID {
			continue
		}

		normalizedOriginal := normalizeAnswer(row.OriginalWord)
		normalizedTranslation := normalizeAnswer(row.TranslationWord)
		if strings.TrimSpace(normalizedOriginal) == "" || strings.TrimSpace(normalizedTranslation) == "" {
			continue
		}
		if _, exists := seenOriginal[normalizedOriginal]; exists {
			continue
		}
		if _, exists := seenTranslation[normalizedTranslation]; exists {
			continue
		}

		seenOriginal[normalizedOriginal] = struct{}{}
		seenTranslation[normalizedTranslation] = struct{}{}
		selectedRows = append(selectedRows, row)
		if len(selectedRows) == matchPairsVocabularyCount {
			break
		}
	}

	options := make([]exerciseChoiceCandidate, 0, len(selectedRows))
	for _, row := range selectedRows {
		options = append(options, exerciseChoiceCandidate{VocabularyID: row.VocabularyID})
	}

	return options, nil
}

func buildExerciseOptionsForType(userID uint, vocabulary *models.Vocabulary, exerciseType enums.ExerciseType) ([]exerciseChoiceCandidate, error) {
	if vocabulary == nil || vocabulary.Translation == nil {
		return nil, errors.New("vocabulary has no translation")
	}

	translation := vocabulary.Translation
	if translation.Original == nil {
		return nil, errors.New("vocabulary has no original word")
	}
	if translation.Translation == nil {
		return nil, errors.New("vocabulary has no translation word")
	}

	answerOption := exerciseChoiceCandidate{
		VocabularyID: vocabulary.ID,
		AnswerWord:   translation.Translation.Word,
	}
	queryColumn := "translated.word"

	if isReversedExerciseType(exerciseType) {
		answerOption.AnswerWord = translation.Original.Word
		queryColumn = "original.word"
	}

	distractors, err := getExerciseDistractorWords(
		userID,
		translation.Original.Language,
		translation.Translation.Language,
		queryColumn,
		vocabulary.ID,
		answerOption.AnswerWord,
	)
	if err != nil {
		return nil, err
	}

	requiredDistractors := choiceExerciseVocabularyCount - 1
	if len(distractors) < requiredDistractors {
		return []exerciseChoiceCandidate{answerOption}, nil
	}

	options := make([]exerciseChoiceCandidate, 0, choiceExerciseVocabularyCount)
	options = append(options, answerOption)
	options = append(options, distractors[:requiredDistractors]...)

	return options, nil
}

func shuffledExerciseOptions(options []exerciseChoiceCandidate) []exerciseChoiceCandidate {
	shuffled := append([]exerciseChoiceCandidate(nil), options...)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled
}

func getExerciseDistractorWords(
	userID uint,
	originalLanguage enums.Language,
	translationLanguage enums.Language,
	answerColumn string,
	correctVocabularyID uuid.UUID,
	correctAnswer string,
) ([]exerciseChoiceCandidate, error) {
	query := `
		SELECT
			v.id AS vocabulary_id,
			` + answerColumn + ` AS answer_word
		FROM vocabulary AS v
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE v.user_id = ?
			AND v.deleted_at IS NULL
			AND original.language = ?
			AND translated.language = ?
			AND v.id <> ?
	`

	var rows []exerciseChoiceCandidate
	if err := db.DB.Raw(query, userID, originalLanguage, translationLanguage, correctVocabularyID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	seen := map[string]struct{}{
		normalizeAnswer(correctAnswer): {},
	}
	options := make([]exerciseChoiceCandidate, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.AnswerWord) == "" {
			continue
		}

		normalized := normalizeAnswer(row.AnswerWord)
		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		options = append(options, row)
	}

	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	return options, nil
}

func buildExerciseQuestionData(vocabulary *models.Vocabulary, exerciseType enums.ExerciseType) (string, enums.Language, enums.Language, error) {
	if vocabulary == nil || vocabulary.Translation == nil {
		return "", "", "", errors.New("vocabulary has no translation")
	}

	translation := vocabulary.Translation
	if translation.Original == nil {
		return "", "", "", errors.New("vocabulary has no original word")
	}
	if translation.Translation == nil {
		return "", "", "", errors.New("vocabulary has no translation word")
	}

	if isReversedExerciseType(exerciseType) {
		return translation.Translation.Word, translation.Translation.Language, translation.Original.Language, nil
	}

	return translation.Original.Word, translation.Original.Language, translation.Translation.Language, nil
}

func isReversedExerciseType(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeBasicReversed, enums.ExerciseTypeChoiceReversed, enums.ExerciseTypeCharactersReversed:
		return true
	default:
		return false
	}
}

func isCharacterExerciseType(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeCharactersDirect, enums.ExerciseTypeCharactersReversed:
		return true
	default:
		return false
	}
}

func isChoiceExerciseType(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed:
		return true
	default:
		return false
	}
}

func isMatchPairsExerciseType(exerciseType enums.ExerciseType) bool {
	return exerciseType == enums.ExerciseTypeMatchPairs
}

func GetExercisesByIDs(userID uint, ids []uuid.UUID) ([]ExerciseListExercise, error) {
	if len(ids) == 0 {
		return []ExerciseListExercise{}, nil
	}

	exercises := make([]models.Exercise, 0, len(ids))

	if err := db.DB.
		Where("user_id = ? AND id IN ?", userID, ids).
		Find(&exercises).Error; err != nil {
		return nil, err
	}

	vocabularyByExerciseID, err := loadListVocabularyByExerciseIDs(collectExerciseIDs(exercises))
	if err != nil {
		return nil, err
	}

	data := make([]ExerciseListExercise, 0, len(exercises))
	for _, exercise := range exercises {
		response := ExerciseListExercise{
			ID:                exercise.ID,
			Type:              exercise.Type,
			Status:            exercise.Status,
			StartedAt:         exercise.StartedAt,
			FinishedAt:        exercise.FinishedAt,
			TelegramMessageID: exercise.TelegramMessageID,
		}

		if vocabulary, ok := vocabularyByExerciseID[exercise.ID]; ok {
			response.Vocabulary = vocabulary
			if len(vocabulary) > 0 {
				legacyVocabulary := vocabulary[0]
				response.LegacyVocabulary = &legacyVocabulary
			}
		}

		data = append(data, response)
	}

	return data, nil
}

func createExerciseVocabularyLinks(tx *gorm.DB, exerciseID uuid.UUID, correctVocabularyID uuid.UUID, options []exerciseChoiceCandidate) error {
	links := make([]map[string]any, 0, len(options))
	for index, option := range options {
		links = append(links, map[string]any{
			"exercise_id":   exerciseID,
			"vocabulary_id": option.VocabularyID,
			"is_correct":    correctVocabularyID == uuid.Nil || option.VocabularyID == correctVocabularyID,
			"position":      index,
		})
	}

	return tx.Table("vocabulary_exercises").Create(&links).Error
}

func GetExerciseMatchCards(exerciseID uuid.UUID) ([]ExerciseMatchCard, error) {
	rows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, true)
	if err != nil {
		return nil, err
	}

	cards := make([]ExerciseMatchCard, 0, len(rows)*2)
	for _, row := range rows {
		cards = append(cards, ExerciseMatchCard{
			ID:           row.VocabularyID.String() + ":" + matchPairCardSideOriginal,
			VocabularyID: row.VocabularyID,
			Word:         row.OriginalWord,
			Language:     row.OriginalLanguage,
			Side:         matchPairCardSideOriginal,
		})
		cards = append(cards, ExerciseMatchCard{
			ID:           row.VocabularyID.String() + ":" + matchPairCardSideTranslation,
			VocabularyID: row.VocabularyID,
			Word:         row.TranslationWord,
			Language:     row.TranslationLanguage,
			Side:         matchPairCardSideTranslation,
		})
	}

	rand.Shuffle(len(cards), func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})

	return cards, nil
}

func GetExerciseAnswerOptions(exerciseID uuid.UUID, exerciseType enums.ExerciseType) ([]ExerciseOption, error) {
	answerColumn := "translated.word"
	if isReversedExerciseType(exerciseType) {
		answerColumn = "original.word"
	}

	query := `
		SELECT
			v.id AS vocabulary_id,
			` + answerColumn + ` AS answer_word
		FROM vocabulary_exercises AS ve
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE ve.exercise_id = ?
	`

	var rows []exerciseChoiceCandidate
	if err := db.DB.Raw(query, exerciseID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	options := make([]ExerciseOption, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.AnswerWord) == "" {
			continue
		}

		options = append(options, ExerciseOption{
			VocabularyID: row.VocabularyID,
			Label:        row.AnswerWord,
		})
	}

	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	return options, nil
}

func getExerciseWithCorrectVocabulary(exerciseID uuid.UUID, userID uint) (*models.Exercise, *exerciseVocabularyDetails, error) {
	var exercise models.Exercise

	err := db.DB.
		Where("id = ? AND user_id = ?", exerciseID, userID).
		Take(&exercise).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrExerciseNotFound
		}

		return nil, nil, err
	}

	correctVocabulary, err := getCorrectExerciseVocabularyDetails(exercise.ID)
	if err != nil {
		return nil, nil, err
	}

	return &exercise, correctVocabulary, nil
}

func getCorrectExerciseVocabularyDetails(exerciseID uuid.UUID) (*exerciseVocabularyDetails, error) {
	rows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, true)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		return &row, nil
	}

	return nil, nil
}

func loadListVocabularyByExerciseIDs(exerciseIDs []uuid.UUID) (map[uuid.UUID][]ExerciseListVocabulary, error) {
	rows, err := getExerciseVocabularyDetails(exerciseIDs, false, false)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]ExerciseListVocabulary, len(rows))
	for _, row := range rows {
		result[row.ExerciseID] = append(result[row.ExerciseID], buildListVocabularyFromExerciseDetails(row))
	}

	return result, nil
}

func getExerciseVocabularyDetails(exerciseIDs []uuid.UUID, onlyCorrect bool, onlyActive bool) ([]exerciseVocabularyDetails, error) {
	if len(exerciseIDs) == 0 {
		return []exerciseVocabularyDetails{}, nil
	}

	query := `
		SELECT
			ve.exercise_id AS exercise_id,
			ve.vocabulary_id AS vocabulary_id,
			ve.is_correct AS is_correct,
			ve.position AS position,
			ve.result AS result,
			ve.result_reason AS result_reason,
			ve.progress_delta AS progress_delta,
			ve.knowledge_after AS knowledge_after,
			ve.answered_at AS answered_at,
			v.deleted_at AS vocabulary_deleted_at,
			original.word AS original_word,
			original.language AS original_language,
			translated.word AS translation_word,
			translated.language AS translation_language
		FROM vocabulary_exercises AS ve
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE ve.exercise_id IN ?
	`
	args := []any{exerciseIDs}
	if onlyCorrect {
		query += ` AND ve.is_correct = ?`
		args = append(args, true)
	}
	if onlyActive {
		query += ` AND v.deleted_at IS NULL`
	} else {
		query += ` AND (ve.is_correct = true OR ve.result IS NOT NULL)`
	}

	query += ` ORDER BY ve.exercise_id, ve.position ASC, ve.id ASC`

	var rows []exerciseVocabularyDetails
	if err := db.DB.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

func buildListVocabularyFromExerciseDetails(details exerciseVocabularyDetails) ExerciseListVocabulary {
	vocabulary := buildVocabularyFromExerciseDetails(details)
	translation := vocabulary.Translation
	if details.VocabularyDeletedAt != nil {
		translation = nil
	}

	return ExerciseListVocabulary{
		ID:             details.VocabularyID,
		Translation:    translation,
		ExerciseResult: details.Result,
		ResultReason:   details.ResultReason,
		ProgressDelta:  details.ProgressDelta,
		KnowledgeAfter: details.KnowledgeAfter,
		AnsweredAt:     details.AnsweredAt,
		IsCorrect:      details.IsCorrect,
		Position:       details.Position,
	}
}

func buildVocabularyFromExerciseDetails(details exerciseVocabularyDetails) models.Vocabulary {
	return models.Vocabulary{
		ID: details.VocabularyID,
		Translation: &models.Translation{
			Original: &models.Word{
				Word:     details.OriginalWord,
				Language: details.OriginalLanguage,
			},
			Translation: &models.Word{
				Word:     details.TranslationWord,
				Language: details.TranslationLanguage,
			},
		},
	}
}

func collectExerciseIDs(exercises []models.Exercise) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(exercises))
	for _, exercise := range exercises {
		ids = append(ids, exercise.ID)
	}

	return ids
}

func collectExerciseOptionLabels(options []exerciseChoiceCandidate) []string {
	labels := make([]string, 0, len(options))
	for _, option := range options {
		labels = append(labels, option.AnswerWord)
	}

	return labels
}

func exerciseOptionsContainAnswer(options []ExerciseOption, normalizedAnswer string) bool {
	for _, option := range options {
		if normalizeAnswer(option.Label) == normalizedAnswer {
			return true
		}
	}

	return false
}
