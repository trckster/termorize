package services

import (
	"errors"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
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

const (
	ExerciseBasicCorrectProgressDelta          = 15
	ExerciseBasicAlmostProgressDelta           = 5
	ExerciseBasicWrongProgressDelta            = -15
	ExerciseCharacterCorrectProgressDelta      = 10
	ExerciseCharacterAlmostProgressDelta       = 5
	ExerciseCharacterWrongProgressDelta        = -20
	ExerciseChoiceCorrectProgressDelta         = 5
	ExerciseChoiceWrongProgressDelta           = -25
	ExerciseMatchCorrectProgressDelta          = 5
	ExerciseMatchAlmostProgressDelta           = 2
	ExerciseMatchWrongProgressDelta            = -15
	KnownVocabularyRepetitionFailProgressDelta = -25
	exerciseReminderPeriod                     = 24 * time.Hour
	telegramExerciseExpirationPeriod           = 7 * 24 * time.Hour
	websiteExerciseExpirationPeriod            = time.Hour
)

type exerciseProgressDeltas struct {
	Correct int
	Almost  int
	Wrong   int
}

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

	basicExerciseWeight      = 30
	choiceExerciseWeight     = 30
	characterExerciseWeight  = 30
	matchPairsExerciseWeight = 10

	knownVocabularyRepetitionMinimumDailyCount = 3
	knownVocabularyRepetitionDiceSides         = 6
	knownVocabularyRepetitionWinningRoll       = 6
)

const ChoiceExerciseVocabularyCount = choiceExerciseVocabularyCount

const MatchPairsVocabularyCount = matchPairsVocabularyCount

type PendingExercise struct {
	ExerciseID                  uuid.UUID          `gorm:"column:exercise_id"`
	ExerciseType                enums.ExerciseType `gorm:"column:exercise_type"`
	IsKnownVocabularyRepetition bool               `gorm:"column:is_known_vocabulary_repetition"`
	UserID                      uint               `gorm:"column:user_id"`
	Username                    string             `gorm:"column:username"`
	TelegramID                  int64              `gorm:"column:telegram_id"`
	OriginalWord                string             `gorm:"column:original_word"`
	OriginalLanguage            enums.Language     `gorm:"column:original_language"`
	TranslationWord             string             `gorm:"column:translation_word"`
	TranslationLanguage         enums.Language     `gorm:"column:translation_language"`
	SystemLanguage              enums.Language     `gorm:"column:system_language"`
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
	ResultReason        string               `gorm:"-"`
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
