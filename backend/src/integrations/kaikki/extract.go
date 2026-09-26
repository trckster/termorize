package kaikki

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"termorize/src/enums"
	"termorize/src/utils"
)

type classification struct {
	RawTags    []string `json:"raw_tags"`
	Tags       []string `json:"tags"`
	Categories []string `json:"categories"`
}

type entry struct {
	classification
	Word     string           `json:"word"`
	Language string           `json:"lang_code"`
	POS      string           `json:"pos"`
	Redirect string           `json:"redirect"`
	Senses   []classification `json:"senses"`
}

type Idiom struct {
	Word     string
	Language enums.Language
}

func SupportedEdition(edition string) bool {
	switch edition {
	case "enwiktionary", "ruwiktionary", "itwiktionary", "dewiktionary", "enwiktionary-es", "frwiktionary", "plwiktionary", "trwiktionary", "enwiktionary-pt", "ukwiktionary":
		return true
	}
	return false
}

func Extract(edition string, line []byte) (*Idiom, error) {
	if !SupportedEdition(edition) {
		return nil, errors.New("unsupported dictionary edition")
	}
	if !utf8.Valid(line) {
		return nil, errors.New("invalid UTF-8")
	}
	var record entry
	if err := json.Unmarshal(line, &record); err != nil {
		return nil, errors.New("invalid entry JSON or field types")
	}
	// Hard redirects have a title and target instead of lexical word/language fields.
	if record.POS == "hard-redirect" && strings.TrimSpace(record.Redirect) != "" {
		return nil, nil
	}
	if record.Word == "" || record.Language == "" {
		return nil, errors.New("missing word or lang_code")
	}
	if !enums.IsSupportedLanguage(enums.Language(record.Language)) {
		return nil, nil
	}
	// Punctuation, bound morphemes and proverbs are not standalone idioms, even with idiomatic senses.
	if slices.Contains([]string{"punct", "prefix", "suffix", "infix", "interfix", "circumfix", "affix", "root", "proverb"}, record.POS) {
		return nil, nil
	}
	word := strings.TrimSpace(record.Word)
	if word == "" || utf8.RuneCountInString(word) > 500 || strings.ContainsFunc(word, unicode.IsControl) {
		return nil, errors.New("invalid expression text")
	}

	if !isIdiom(edition, record.Language, record.classification) {
		matched := false
		for _, sense := range record.Senses {
			if isIdiom(edition, record.Language, sense) {
				matched = true
				break
			}
		}
		if !matched {
			return nil, nil
		}
	}
	return &Idiom{Word: utils.NormalizeWordCasingForLanguage(word, record.Language), Language: enums.Language(record.Language)}, nil
}

func isIdiom(edition, language string, c classification) bool {
	if slices.Contains(c.Tags, "morpheme") || slices.Contains(c.Tags, "proverb") {
		return false
	}
	if slices.Contains(c.Tags, "idiomatic") {
		return true
	}
	var category string
	switch edition {
	case "enwiktionary":
		category = enums.Language(language).DisplayName() + " idioms"
	case "ruwiktionary":
		category = "Фразеологизмы/" + language
	case "dewiktionary":
		return language == "de" && slices.Contains(c.Categories, "Redewendung (Deutsch)")
	case "frwiktionary":
		if language != "fr" {
			return false
		}
		for _, category := range c.Categories {
			if strings.HasPrefix(category, "Idiotismes ") && strings.HasSuffix(category, " en français") {
				return true
			}
		}
		return false
	case "trwiktionary":
		return language == "tr" && slices.Contains(c.Categories, "Türkçe deyimler")
	case "itwiktionary":
		// Italian keeps this explicit label untranslated; locuzioni and proverbs are broader.
		return slices.Contains(c.RawTags, "idiomatico")
	}
	return slices.Contains(c.Categories, category)
}
