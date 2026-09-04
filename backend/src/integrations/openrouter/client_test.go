package openrouter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectionPromptRequiresPortugueseNounArticles(t *testing.T) {
	prompt := buildSystemPrompt([]string{"en", "pt", "uk"})

	require.Contains(t, prompt, "Spanish, French, or Portuguese")
	require.Contains(t, prompt, "a perna")
}

func TestDescriptionPromptRequiresAClueInTheRequestedLanguage(t *testing.T) {
	prompt := buildDescriptionSystemPrompt("Ukrainian")

	require.Contains(t, prompt, "in Ukrainian")
	require.Contains(t, prompt, "Do not include the given text")
	require.Contains(t, prompt, "a direct translation")
	require.Contains(t, prompt, "inflected, conjugated, declined, and irregular forms")
	require.Contains(t, prompt, `{"description": string, "forbidden_forms": string[]}`)
}
