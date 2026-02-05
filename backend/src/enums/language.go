package enums

type Language string

const (
	LanguageEn Language = "en"
	LanguageRu Language = "ru"
	LanguageIt Language = "it"
	LanguageDe Language = "de"
)

func AllLanguages() []string {
	return []string{
		string(LanguageEn),
		string(LanguageRu),
		string(LanguageIt),
		string(LanguageDe),
	}
}
