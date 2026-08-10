package telegram

import (
	"errors"
	"testing"

	"termorize/src/enums"
	"termorize/src/integrations/google"
	"termorize/src/models"

	"github.com/stretchr/testify/require"
)

type languageDetectionFake struct {
	detected string
	err      error
}

func (f *languageDetectionFake) DetectLanguage(string) (string, error) {
	return f.detected, f.err
}

func (f *languageDetectionFake) Translate(string, string, string) (string, error) {
	return "", errors.New("unexpected translation call")
}

func TestDetectMessageTranslationLanguages(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		detected       string
		detectionError error
		source         enums.Language
		target         enums.Language
		wantSource     enums.Language
		wantTarget     enums.Language
	}{
		{
			name:       "detected configured source keeps direction",
			text:       "dog",
			detected:   "en",
			source:     enums.LanguageEn,
			target:     enums.LanguageRu,
			wantSource: enums.LanguageEn,
			wantTarget: enums.LanguageRu,
		},
		{
			name:       "detected configured target reverses direction",
			text:       "dog",
			detected:   "en",
			source:     enums.LanguageRu,
			target:     enums.LanguageEn,
			wantSource: enums.LanguageEn,
			wantTarget: enums.LanguageRu,
		},
		{
			name:       "third language detection uses Latin script to reverse Russian source",
			text:       "frugal",
			detected:   "pt",
			source:     enums.LanguageRu,
			target:     enums.LanguageEn,
			wantSource: enums.LanguageEn,
			wantTarget: enums.LanguageRu,
		},
		{
			name:           "detection failure uses Latin script to reverse Russian source",
			text:           "frugal",
			detectionError: errors.New("detection unavailable"),
			source:         enums.LanguageRu,
			target:         enums.LanguageEn,
			wantSource:     enums.LanguageEn,
			wantTarget:     enums.LanguageRu,
		},
		{
			name:       "third language detection uses Cyrillic script to reverse Russian target",
			text:       "слово",
			detected:   "uk",
			source:     enums.LanguageEn,
			target:     enums.LanguageRu,
			wantSource: enums.LanguageRu,
			wantTarget: enums.LanguageEn,
		},
		{
			name:           "detection failure uses Cyrillic script for Ukrainian source",
			text:           "привіт",
			detectionError: errors.New("detection unavailable"),
			source:         enums.LanguageUk,
			target:         enums.LanguagePt,
			wantSource:     enums.LanguageUk,
			wantTarget:     enums.LanguagePt,
		},
		{
			name:           "detection failure uses Latin script to reverse Ukrainian source",
			text:           "olá",
			detectionError: errors.New("detection unavailable"),
			source:         enums.LanguageUk,
			target:         enums.LanguagePt,
			wantSource:     enums.LanguagePt,
			wantTarget:     enums.LanguageUk,
		},
		{
			name:           "detection failure keeps direction for two Cyrillic languages",
			text:           "слово",
			detectionError: errors.New("detection unavailable"),
			source:         enums.LanguageUk,
			target:         enums.LanguageRu,
			wantSource:     enums.LanguageUk,
			wantTarget:     enums.LanguageRu,
		},
		{
			name:       "Cyrillic majority wins for mixed-script text",
			text:       "gородская улица",
			detected:   "uk",
			source:     enums.LanguageRu,
			target:     enums.LanguageEn,
			wantSource: enums.LanguageRu,
			wantTarget: enums.LanguageEn,
		},
		{
			name:       "Latin majority wins for mixed-script text",
			text:       "helloя",
			detected:   "uk",
			source:     enums.LanguageRu,
			target:     enums.LanguageEn,
			wantSource: enums.LanguageEn,
			wantTarget: enums.LanguageRu,
		},
		{
			name:       "third language detection falls back to English in a Latin pair",
			text:       "sale",
			detected:   "fi",
			source:     enums.LanguageIt,
			target:     enums.LanguageEn,
			wantSource: enums.LanguageEn,
			wantTarget: enums.LanguageIt,
		},
		{
			name:           "detection failure falls back to English in a Latin pair",
			text:           "sale",
			detectionError: errors.New("detection unavailable"),
			source:         enums.LanguageIt,
			target:         enums.LanguageEn,
			wantSource:     enums.LanguageEn,
			wantTarget:     enums.LanguageIt,
		},
		{
			name:       "third language keeps configured direction when English is unavailable",
			text:       "sale",
			detected:   "fi",
			source:     enums.LanguageIt,
			target:     enums.LanguageDe,
			wantSource: enums.LanguageIt,
			wantTarget: enums.LanguageDe,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalFactory := google.NewTranslateClient
			t.Cleanup(func() { google.NewTranslateClient = originalFactory })
			google.NewTranslateClient = func() google.TranslateClient {
				return &languageDetectionFake{
					detected: test.detected,
					err:      test.detectionError,
				}
			}

			user := &models.User{Settings: models.UserSettings{
				TranslationSourceLanguage: test.source,
				TranslationTargetLanguage: test.target,
			}}

			source, target, err := detectMessageTranslationLanguages(user, test.text)

			require.NoError(t, err)
			require.Equal(t, test.wantSource, source)
			require.Equal(t, test.wantTarget, target)
		})
	}
}
