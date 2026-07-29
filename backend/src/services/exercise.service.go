package services

import (
	"encoding/json"
	"errors"
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
	ID                 uuid.UUID                   `json:"id"`
	Type               enums.ExerciseType          `json:"type"`
	Status             enums.ExerciseStatus        `json:"status"`
	StartedAt          *time.Time                  `json:"starts_at"`
	FinishedAt         *time.Time                  `json:"finishes_at"`
	TelegramMessageID  *int64                      `json:"telegram_message_id"`
	Vocabulary         []ExerciseListVocabulary    `json:"vocabularies"`
	LegacyVocabulary   *ExerciseListVocabulary     `json:"vocabulary,omitempty"`
	CollectionPractice *ExerciseCollectionPractice `json:"collection_practice,omitempty"`
}

type ExerciseCollectionPractice struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
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

type characterStateJSON struct {
	Order  []int `json:"order"`
	Chosen []int `json:"chosen"`
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
			progressDelta = exerciseProgressDelta(exercise, ExerciseChoiceCompleteProgressDelta)
			updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultCorrect, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
			resultType = "correct"
		} else if exerciseOptionsContainAnswer(options, normalizedAnswer) {
			progressDelta = exerciseProgressDelta(exercise, ExerciseChoiceFailProgressDelta)
			updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusFailed, ExerciseVocabularyResultWrong, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
			resultType = "wrong"
		} else {
			progressDelta = exerciseProgressDelta(exercise, ExerciseChoiceFailProgressDelta)
			updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusFailed, ExerciseVocabularyResultWrong, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
			resultType = "wrong"
		}
	} else {
		answerReason := ExerciseVocabularyResultReasonTypedAnswer
		if isCharacterExerciseType(exercise.Type) {
			answerReason = ExerciseVocabularyResultReasonCharacterAnswer
		}

		if normalizedAnswer == normalizedExpectedAnswer {
			progressDelta = exerciseProgressDelta(exercise, ExerciseCompleteProgressDelta)
			updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultCorrect, answerReason, progressDelta)
			resultType = "correct"
		} else {
			distance := utils.LevenshteinDistance(normalizedAnswer, normalizedExpectedAnswer)
			threshold := almostCorrectThreshold(normalizedExpectedAnswer)
			if distance <= threshold {
				progressDelta = exerciseProgressDelta(exercise, ExerciseAlmostCorrectProgressDelta)
				updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultAlmost, answerReason, progressDelta)
				resultType = "almost"
			} else {
				progressDelta = exerciseProgressDelta(exercise, ExerciseFailProgressDelta)
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
		progressDelta = exerciseProgressDelta(exercise, ExerciseChoiceCompleteProgressDelta)
		updated, knowledge, err = FinishExercise(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultCorrect, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
		resultType = "correct"
	} else {
		progressDelta = exerciseProgressDelta(exercise, ExerciseChoiceFailProgressDelta)
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

func exerciseProgressDelta(exercise *models.Exercise, regularDelta int) int {
	if exercise != nil && exercise.PracticeCollectionTitle != nil {
		return 0
	}

	return regularDelta
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
