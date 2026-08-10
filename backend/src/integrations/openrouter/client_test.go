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
