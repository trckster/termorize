package enums

type KnowledgeType string

const (
	KnowledgeTypeTranslation KnowledgeType = "translation"
)

type TranslationSource string

const (
	TranslationSourceUser       TranslationSource = "user"
	TranslationSourceDictionary TranslationSource = "dictionary"
	TranslationSourceGoogle     TranslationSource = "google"
)

type ExerciseType string

const (
	ExerciseTypeBasic ExerciseType = "basic"
)

type ExerciseStatus string

const (
	ExerciseStatusPending    ExerciseStatus = "pending"
	ExerciseStatusInProgress ExerciseStatus = "inProgress"
	ExerciseStatusIgnored    ExerciseStatus = "ignored"
	ExerciseStatusCompleted  ExerciseStatus = "completed"
	ExerciseStatusFailed     ExerciseStatus = "failed"
)

type TelegramState string

const (
	TelegramStateNone                   TelegramState = ""
	TelegramStateDeletingVocabulary     TelegramState = "deletingVocabulary"
	TelegramStateAddingVocabularyFirst  TelegramState = "addingVocabularyFirst"
	TelegramStateAddingVocabularySecond TelegramState = "addingVocabularySecond"
)
