package kaikki

import (
	"bufio"
	"os"
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
		{"ruwiktionary", []string{"al dente", "off the top of one's head"}, []string{"it", "en"}},
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
		{name: "ordinary phrase", line: `{"word":"red apple","lang_code":"en","pos":"phrase"}`},
		{name: "only a translation", line: `{"word":"обычный","lang_code":"ru","translations":[{"word":"piece of cake","lang_code":"en","tags":["idiomatic"]}]}`},
		{name: "unrelated category language", line: `{"word":"ordinary","lang_code":"en","categories":["Russian idioms"]}`},
		{name: "unsupported entry language", line: `{"word":"das ist","lang_code":"de","tags":["idiomatic"]}`},
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
