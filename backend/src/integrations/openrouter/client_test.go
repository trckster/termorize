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
	require.Contains(t, prompt, "Use the supplied translation to identify the specific meaning")
	require.Contains(t, prompt, `{"description": string}`)
}

func TestDescriptionValidationPromptChecksMorphology(t *testing.T) {
	prompt := buildDescriptionValidationSystemPrompt()

	require.Contains(t, prompt, "inflected, conjugated, declined, irregular, derived")
	require.Contains(t, prompt, "same specific sense")
	require.Contains(t, prompt, `{"contains_answer_form": boolean, "matches_translation": boolean}`)
}

type descriptionRoundTripper func(*http.Request) (*http.Response, error)

func (f descriptionRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func setupClientTestConfig(t *testing.T) {
	t.Helper()
	logger.UseNop()
	for _, key := range []string{"SECRET", "DB_USER", "DB_PASSWORD", "TELEGRAM_BOT_TOKEN", "TELEGRAM_LOGIN_CLIENT_ID", "TELEGRAM_LOGIN_CLIENT_SECRET", "GOOGLE_API_KEY"} {
		t.Setenv(key, "test")
	}
	config.LoadEnv()
}

func TestDescriptionRequestsUseSelectedModelAndSupportedSampling(t *testing.T) {
	setupClientTestConfig(t)
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
				if calls == 0 {
					messages := request["messages"].([]any)
					require.Equal(t, "user", messages[1].(map[string]any)["role"])
					require.Equal(t, `Describe the concept represented by "cat" in English, whose translation is "il gatto" in Italian.`, messages[1].(map[string]any)["content"])
				}
				content := `{"description":"A small pet that purrs."}`
				if calls > 0 {
					messages := request["messages"].([]any)
					var validationInput map[string]string
					require.NoError(t, json.Unmarshal([]byte(messages[1].(map[string]any)["content"].(string)), &validationInput))
					require.Equal(t, "cat", validationInput["answer"])
					require.Equal(t, "English", validationInput["language"])
					require.Equal(t, "il gatto", validationInput["translation"])
					require.Equal(t, "Italian", validationInput["translation_language"])
					require.Equal(t, "A small pet that purrs.", validationInput["description"])
					content = `{"contains_answer_form":false,"matches_translation":true}`
				}
				calls++
				payload, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": content}}}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(payload)), Header: make(http.Header)}, nil
			})}}
			result, err := c.GenerateDescription("cat", "English", "il gatto", "Italian", "English")
			require.NoError(t, err)
			validation, err := c.ValidateDescription("cat", "English", "il gatto", "Italian", result.Description)
			require.NoError(t, err)
			require.False(t, validation.ContainsAnswerForm)
			require.True(t, validation.MatchesTranslation)
			require.Equal(t, 2, calls)
		})
	}
}

func TestDescriptionValidationRequiresBothDecisions(t *testing.T) {
	setupClientTestConfig(t)
	for _, test := range []struct {
		name    string
		content string
		valid   bool
	}{
		{"different sense", `{"contains_answer_form":false,"matches_translation":false}`, true},
		{"missing sense decision", `{"contains_answer_form":false}`, false},
		{"missing answer decision", `{"matches_translation":true}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := &client{apiKey: "test", model: "test-model", http: &http.Client{Transport: descriptionRoundTripper(func(*http.Request) (*http.Response, error) {
				payload, err := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": test.content}}}})
				require.NoError(t, err)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(payload)), Header: make(http.Header)}, nil
			})}}
			validation, err := c.ValidateDescription("Run down", "English", "наезжать", "Russian", "To be in poor condition.")
			if !test.valid {
				require.Error(t, err)
				require.Nil(t, validation)
				return
			}
			require.NoError(t, err)
			require.False(t, validation.ContainsAnswerForm)
			require.False(t, validation.MatchesTranslation)
		})
	}
}
