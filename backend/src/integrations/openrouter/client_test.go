package openrouter

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"termorize/src/config"
	"termorize/src/logger"
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
	require.Contains(t, prompt, `{"description": string}`)
}

func TestDescriptionValidationPromptChecksMorphology(t *testing.T) {
	prompt := buildDescriptionValidationSystemPrompt()

	require.Contains(t, prompt, "inflected, conjugated, declined, irregular, derived")
	require.Contains(t, prompt, `{"contains_answer_form": boolean}`)
}

type descriptionRoundTripper func(*http.Request) (*http.Response, error)

func (f descriptionRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDescriptionRequestsUseSelectedModelAndSupportedSampling(t *testing.T) {
	logger.UseNop()
	for _, key := range []string{"SECRET", "DB_USER", "DB_PASSWORD", "TELEGRAM_BOT_TOKEN", "TELEGRAM_LOGIN_CLIENT_ID", "TELEGRAM_LOGIN_CLIENT_SECRET", "GOOGLE_API_KEY"} {
		t.Setenv(key, "test")
	}
	config.LoadEnv()
	for _, model := range []string{"google/gemini-2.5-flash", "moonshotai/kimi-k2.6", "openai/gpt-5.6-sol"} {
		t.Run(model, func(t *testing.T) {
			calls := 0
			c := &client{apiKey: "test", model: model, http: &http.Client{Transport: descriptionRoundTripper(func(r *http.Request) (*http.Response, error) {
				var request map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, model, request["model"])
				if model == "openai/gpt-5.6-sol" {
					require.NotContains(t, request, "temperature")
				} else {
					require.Contains(t, request, "temperature")
				}
				content := `{"description":"A small pet that purrs."}`
				if calls > 0 {
					content = `{"contains_answer_form":false}`
				}
				calls++
				payload, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": content}}}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(payload)), Header: make(http.Header)}, nil
			})}}
			result, err := c.GenerateDescription("cat", "English", "English")
			require.NoError(t, err)
			contains, err := c.DescriptionContainsAnswerForm("cat", "English", result.Description)
			require.NoError(t, err)
			require.False(t, contains)
			require.Equal(t, 2, calls)
		})
	}
}
