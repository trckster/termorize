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
	default:
		return "🏳️"
	}
}
