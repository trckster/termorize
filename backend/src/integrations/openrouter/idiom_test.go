package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdiomDescriptionRequest(t *testing.T) {
	setupClientTestConfig(t)
	for _, model := range []string{"google/gemini-2.5-flash", "openai/gpt-5.6-sol"} {
		t.Run(model, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := &client{apiKey: "test", model: model, http: &http.Client{Transport: descriptionRoundTripper(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, ctx, r.Context())
				var request chatRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, model, request.Model)
				require.Equal(t, "json_object", request.ResponseFormat.Type)
				require.Len(t, request.Messages, 2)
				require.Equal(t, `Explain the idiom "essere al settimo cielo" in Italian.`, request.Messages[1].Content)
				require.Contains(t, request.Messages[0].Content, "Write in Italian")
				require.Contains(t, request.Messages[0].Content, "figurative meanings")
				require.Contains(t, request.Messages[0].Content, `Separate distinct meanings with "; "`)
				require.Contains(t, request.Messages[0].Content, `Use ", " for enumerations`)
				if model == "openai/gpt-5.6-sol" {
					require.Nil(t, request.Temperature)
				}
				body := `{"choices":[{"message":{"content":"{\"description\":\"felice, entusiasta, euforico\"}"}}]}`
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}}
			result, err := c.GenerateIdiomDescription(ctx, "essere al settimo cielo", "Italian")
			require.NoError(t, err)
			require.Equal(t, "felice, entusiasta, euforico", result.Description)
		})
	}
}

func TestIdiomDescriptionProviderFailures(t *testing.T) {
	setupClientTestConfig(t)
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"unavailable", 503, `{}`},
		{"no choices", 200, `{"choices":[]}`},
		{"invalid envelope", 200, `not json`},
		{"invalid content", 200, `{"choices":[{"message":{"content":"not json"}}]}`},
		{"invalid type", 200, `{"choices":[{"message":{"content":"{\"description\":42}"}}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &client{apiKey: "test", model: "test", http: &http.Client{Transport: descriptionRoundTripper(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})}}
			_, err := c.GenerateIdiomDescription(context.Background(), "under the weather", "English")
			require.Error(t, err)
		})
	}
	c := &client{apiKey: "", http: &http.Client{Transport: descriptionRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Error("unconfigured generation must not make a request")
		return nil, fmt.Errorf("unexpected request")
	})}}
	_, err := c.GenerateIdiomDescription(context.Background(), "under the weather", "English")
	require.ErrorIs(t, err, ErrNotConfigured)
}

func TestIdiomTranslationUsesFigurativePromptAndValidatesResponse(t *testing.T) {
	setupClientTestConfig(t)
	for _, content := range []string{`{"translation":"растопить лёд"}`, `{"translation":" "}`, `{"translation":42}`, `not json`} {
		t.Run(content, func(t *testing.T) {
			ctx := context.Background()
			c := &client{apiKey: "test", model: "test", http: &http.Client{Transport: descriptionRoundTripper(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, apiURL, r.URL.String())
				var request chatRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Contains(t, request.Messages[0].Content, "natural equivalent idiom")
				require.Contains(t, request.Messages[0].Content, "Avoid literal translation")
				require.Contains(t, request.Messages[1].Content, `"source_language":"English"`)
				require.Contains(t, request.Messages[1].Content, `"target_language":"Russian"`)
				body, err := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}}})
				require.NoError(t, err)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
			})}}
			result, err := c.TranslateIdiom(ctx, "break the ice", "English", "Russian")
			if strings.Contains(content, "растопить") {
				require.NoError(t, err)
				require.Equal(t, "растопить лёд", result)
			} else {
				require.Error(t, err)
			}
		})
	}
	_, err := (&client{}).TranslateIdiom(context.Background(), "break the ice", "English", "Russian")
	require.ErrorIs(t, err, ErrNotConfigured)
}
