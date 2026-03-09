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
	case LanguageEs:
		return "Spanish"
	case LanguageFr:
		return "French"
	case LanguagePl:
		return "Polish"
	case LanguageTr:
		return "Turkish"
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
	default:
		return "🏳️"
	}
}
