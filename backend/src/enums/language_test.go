package enums

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPortugueseAndUkrainianLanguageMetadata(t *testing.T) {
	tests := []struct {
		language Language
		code     string
		name     string
		flag     string
	}{
		{LanguagePt, "pt", "Portuguese", "🇵🇹"},
		{LanguageUk, "uk", "Ukrainian", "🇺🇦"},
	}

	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			assert.True(t, IsSupportedLanguage(test.language))
			assert.Contains(t, AllLanguages(), test.code)
			assert.Equal(t, test.name, test.language.DisplayName())
			assert.Equal(t, test.flag, test.language.Flag())
			assert.Equal(t, test.flag+" "+test.name, test.language.DisplayNameWithFlag())
		})
	}
}
