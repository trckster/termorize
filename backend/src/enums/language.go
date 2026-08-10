package enums

type Language string

const (
	LanguageEn Language = "en"
	LanguageRu Language = "ru"
	LanguageIt Language = "it"
	LanguageDe Language = "de"
	LanguageEs Language = "es"
	LanguageFr Language = "fr"
	LanguagePl Language = "pl"
	LanguageTr Language = "tr"
	LanguagePt Language = "pt"
	LanguageUk Language = "uk"
)

func AllLanguages() []string {
	return []string{
		string(LanguageEn),
		string(LanguageRu),
		string(LanguageIt),
		string(LanguageDe),
		string(LanguageEs),
		string(LanguageFr),
		string(LanguagePl),
		string(LanguageTr),
		string(LanguagePt),
		string(LanguageUk),
	}
}

func AllSystemLanguages() []string {
	return []string{
		string(LanguageEn),
		string(LanguageRu),
	}
}

func AllLanguageValues() []Language {
	values := make([]Language, 0, len(AllLanguages()))
	for _, language := range AllLanguages() {
		values = append(values, Language(language))
	}
	return values
}

func AllSystemLanguageValues() []Language {
	values := make([]Language, 0, len(AllSystemLanguages()))
	for _, language := range AllSystemLanguages() {
		values = append(values, Language(language))
	}
	return values
}

func IsSupportedLanguage(language Language) bool {
	for _, supported := range AllLanguageValues() {
		if language == supported {
			return true
		}
	}
	return false
}

func (l Language) DisplayName() string {
	switch l {
	case LanguageEn:
		return "English"
	case LanguageRu:
		return "Russian"
	case LanguageIt:
		return "Italian"
	case LanguageDe:
		return "German"
	case LanguageEs:
		return "Spanish"
	case LanguageFr:
		return "French"
	case LanguagePl:
		return "Polish"
	case LanguageTr:
		return "Turkish"
	case LanguagePt:
		return "Portuguese"
	case LanguageUk:
		return "Ukrainian"
	default:
		return string(l)
	}
}

func (l Language) DisplayNameWithFlag() string {
	return l.Flag() + " " + l.DisplayName()
}

func (l Language) Flag() string {
	switch l {
	case LanguageEn:
		return "🇬🇧"
	case LanguageRu:
		return "🇷🇺"
	case LanguageIt:
		return "🇮🇹"
	case LanguageDe:
		return "🇩🇪"
	case LanguageEs:
		return "🇪🇸"
	case LanguageFr:
		return "🇫🇷"
	case LanguagePl:
		return "🇵🇱"
	case LanguageTr:
		return "🇹🇷"
	case LanguagePt:
		return "🇵🇹"
	case LanguageUk:
		return "🇺🇦"
	default:
		return "🏳️"
	}
}
