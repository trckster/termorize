package enums

type PartOfSpeech string

const (
	PartOfSpeechNoun         PartOfSpeech = "noun"
	PartOfSpeechVerb         PartOfSpeech = "verb"
	PartOfSpeechAdjective    PartOfSpeech = "adjective"
	PartOfSpeechAdverb       PartOfSpeech = "adverb"
	PartOfSpeechPronoun      PartOfSpeech = "pronoun"
	PartOfSpeechPreposition  PartOfSpeech = "preposition"
	PartOfSpeechConjunction  PartOfSpeech = "conjunction"
	PartOfSpeechNumeral      PartOfSpeech = "numeral"
	PartOfSpeechInterjection PartOfSpeech = "interjection"
	PartOfSpeechPhrase       PartOfSpeech = "phrase"
	PartOfSpeechUnknown      PartOfSpeech = "unknown"
)

func (p PartOfSpeech) Valid() bool {
	switch p {
	case PartOfSpeechNoun, PartOfSpeechVerb, PartOfSpeechAdjective, PartOfSpeechAdverb,
		PartOfSpeechPronoun, PartOfSpeechPreposition, PartOfSpeechConjunction,
		PartOfSpeechNumeral, PartOfSpeechInterjection, PartOfSpeechPhrase, PartOfSpeechUnknown:
		return true
	}
	return false
}
