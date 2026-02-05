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
