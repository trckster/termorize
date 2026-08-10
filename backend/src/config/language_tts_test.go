package config_test

import (
	"testing"

	"termorize/src/config"
	"termorize/src/enums"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEverySupportedLanguageHasDedicatedFallbackTTSVoice(t *testing.T) {
	voices := make(map[string]string)
	for _, language := range enums.AllLanguages() {
		configs := config.GetOpenRouterTTSConfigs(language)
		require.Len(t, configs, 2, "language %s", language)
		assert.NotEmpty(t, configs[0].Voice, "primary voice for %s", language)
		assert.NotEmpty(t, configs[1].Voice, "fallback voice for %s", language)
		if language != string(enums.LanguageEn) {
			assert.NotEqual(t, "en-US-Harper:MAI-Voice-2", configs[1].Voice, "language %s must not silently use the English fallback", language)
		}
		voices[language] = configs[1].Voice
	}

	assert.Equal(t, "pt-BR-FranciscaNeural", voices[string(enums.LanguagePt)])
	assert.Equal(t, "uk-UA-PolinaNeural", voices[string(enums.LanguageUk)])
}
