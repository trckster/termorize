package kaikki

import (
	"bufio"
	"os"
	"termorize/src/enums"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractEditionFixtures(t *testing.T) {
	for _, tt := range []struct {
		edition   string
		words     []string
		languages []string
	}{
		{"enwiktionary", []string{"rain cats and dogs", "rompere il ghiaccio", "бить баклуши"}, []string{"en", "it", "ru"}},
		{"ruwiktionary", []string{"al dente", "off the top of one's head", "бить баклуши"}, []string{"it", "en", "ru"}},
		{"itwiktionary", []string{"itsy bitsy"}, []string{"en"}},
	} {
		t.Run(tt.edition, func(t *testing.T) {
			file, err := os.Open("testdata/" + tt.edition + ".jsonl")
			require.NoError(t, err)
			defer file.Close()
			scanner := bufio.NewScanner(file)
			var words, languages []string
			for scanner.Scan() {
				idiom, err := Extract(tt.edition, scanner.Bytes())
				require.NoError(t, err)
				if idiom != nil {
					words = append(words, idiom.Word)
					languages = append(languages, string(idiom.Language))
				}
			}
			require.NoError(t, scanner.Err())
			assert.Equal(t, tt.words, words)
			assert.Equal(t, tt.languages, languages)
		})
	}
}

func TestExtractClassificationBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name, edition, line, word string
		invalid                   bool
	}{
		{name: "missing optional fields", line: `{"word":"piece of cake","lang_code":"en","tags":["idiomatic"]}`, word: "piece of cake"},
		{name: "space punctuation", line: `{"word":" ","lang_code":"en","pos":"punct"}`},
		{name: "punctuation with idiomatic tag", line: `{"word":"!","lang_code":"en","pos":"punct","tags":["idiomatic"]}`},
		{name: "blank idiom", line: `{"word":" ","lang_code":"en","pos":"phrase","tags":["idiomatic"]}`, invalid: true},
		{name: "ordinary phrase", line: `{"word":"red apple","lang_code":"en","pos":"phrase"}`},
		{name: "hard redirect", line: `{"title":"grain of salt","redirect":"with a grain of salt","pos":"hard-redirect"}`},
		{name: "redirect missing target", line: `{"title":"alias","pos":"hard-redirect"}`, invalid: true},
		{name: "redirect empty target", line: `{"title":"alias","redirect":" ","pos":"hard-redirect"}`, invalid: true},
		{name: "redirect wrong field type", line: `{"title":"alias","redirect":{},"pos":"hard-redirect"}`, invalid: true},
		{name: "redirect without classification", line: `{"title":"alias","redirect":"target"}`, invalid: true},
		{name: "only a translation", line: `{"word":"обычный","lang_code":"ru","translations":[{"word":"piece of cake","lang_code":"en","tags":["idiomatic"]}]}`},
		{name: "unrelated category language", line: `{"word":"ordinary","lang_code":"en","categories":["Russian idioms"]}`},
		{name: "unsupported entry language", line: `{"word":"unsupported","lang_code":"xx","tags":["idiomatic"]}`},
		{name: "sense category", edition: "ruwiktionary", line: `{"word":"бить баклуши","lang_code":"ru","senses":[{"categories":["Фразеологизмы/ru"]}]}`, word: "бить баклуши"},
		{name: "italian raw tag only in italian edition", line: `{"word":"itsy bitsy","lang_code":"en","raw_tags":["idiomatico"]}`},
		{name: "proverb", line: `{"word":"time is money","lang_code":"en","pos":"proverb","tags":["idiomatic"]}`},
		{name: "preserve phrase casing", line: `{"word":"  Piece of Cake  ","lang_code":"en","tags":["idiomatic"]}`, word: "Piece of Cake"},
		{name: "italian normalization", line: `{"word":"La Vita","lang_code":"it","tags":["idiomatic"]}`, word: "la vita"},
		{name: "missing text", line: `{"lang_code":"en"}`, invalid: true},
		{name: "missing language", line: `{"word":"hello"}`, invalid: true},
		{name: "null entry", line: `null`, invalid: true},
		{name: "invalid json", line: `{`, invalid: true},
		{name: "wrong field type", line: `{"word":"hello","lang_code":"en","senses":{}}`, invalid: true},
		{name: "control character", line: `{"word":"bad\u0000text","lang_code":"en","tags":["idiomatic"]}`, invalid: true},
		{name: "invalid utf8", line: string([]byte{0xff}), invalid: true},
		{name: "unknown edition", edition: "other", line: `{}`, invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			edition := tt.edition
			if edition == "" {
				edition = "enwiktionary"
			}
			idiom, err := Extract(edition, []byte(tt.line))
			if tt.invalid {
				require.Error(t, err)
				require.Nil(t, idiom)
				return
			}
			require.NoError(t, err)
			if tt.word == "" {
				require.Nil(t, idiom)
			} else {
				require.NotNil(t, idiom)
				assert.Equal(t, tt.word, idiom.Word)
			}
		})
	}
}

func TestExtractSupportsEveryApplicationLanguage(t *testing.T) {
	for _, language := range enums.AllLanguageValues() {
		for _, classification := range []string{`"tags":["idiomatic"]`, `"categories":["` + language.DisplayName() + ` idioms"]`} {
			t.Run(string(language)+classification, func(t *testing.T) {
				idiom, err := Extract("enwiktionary", []byte(`{"word":"example phrase","lang_code":"`+string(language)+`",`+classification+`}`))
				require.NoError(t, err)
				require.NotNil(t, idiom)
				require.Equal(t, language, idiom.Language)
			})
		}
	}

}

func TestExtractNewSourceClassifications(t *testing.T) {
	for _, tc := range []struct {
		edition, line string
		accepted      bool
	}{
		{"dewiktionary", `{"word":"Tacheles reden","lang_code":"de","pos":"phrase","categories":["Redewendung (Deutsch)"]}`, true},
		{"dewiktionary", `{"word":"a proverb","lang_code":"de","tags":["proverb"],"categories":["Redewendung (Deutsch)"]}`, false},
		{"dewiktionary", `{"word":"test","lang_code":"fr","categories":["Redewendung (Deutsch)"]}`, false},
		{"frwiktionary", `{"word":"donner sa langue au chat","lang_code":"fr","pos":"verb","senses":[{"categories":["Idiotismes animaliers en français"]}]}`, true},
		{"frwiktionary", `{"word":"pomme de terre","lang_code":"fr","pos":"noun","categories":["Locutions nominales en français"]}`, false},
		{"plwiktionary", `{"word":"złote usta","lang_code":"pl","senses":[{"tags":["idiomatic"]}]}`, true},
		{"trwiktionary", `{"word":"göze girmek","lang_code":"tr","categories":["Türkçe deyimler"]}`, true},
		{"enwiktionary-es", `{"word":"tirar la toalla","lang_code":"es","tags":["idiomatic"]}`, true},
		{"enwiktionary-pt", `{"word":"quebrar o gelo","lang_code":"pt","tags":["idiomatic"]}`, true},
		{"ukwiktionary", `{"word":"бити байдики","lang_code":"uk","tags":["idiomatic"]}`, true},
		{"trwiktionary", `{"word":"test proverb","lang_code":"tr","pos":"proverb","tags":["idiomatic"]}`, false},
	} {
		t.Run(tc.edition+tc.line, func(t *testing.T) {
			value, err := Extract(tc.edition, []byte(tc.line))
			require.NoError(t, err)
			assert.Equal(t, tc.accepted, value != nil)
		})
	}
}
