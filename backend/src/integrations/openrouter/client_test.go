package openrouter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCollectionReturnsOpenRouterUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		var request chatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		assert.Equal(t, "requested/model", request.Model)

		_, _ = w.Write([]byte(`{
			"id":"gen-123",
			"model":"routed/model",
			"choices":[{"message":{"content":"{\"title\":\"Animals\",\"translations\":[]}"}}],
			"usage":{"cost":0.0012345678,"prompt_tokens":100,"completion_tokens":20,"total_tokens":120}
		}`))
	}))
	defer server.Close()
	restoreAPIURL(t, server.URL)

	client := &client{apiKey: "secret", model: "requested/model", http: server.Client()}
	result, err := client.GenerateCollection("animals", []string{"en", "de"})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Animals", result.Collection.Title)
	assert.Equal(t, Usage{
		GenerationID: "gen-123", Model: "routed/model", Cost: 0.0012345678,
		PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120,
	}, result.Usage)
}

func TestGenerateCollectionKeepsUsageWhenContentIsInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"id":"gen-invalid",
			"model":"model",
			"choices":[{"message":{"content":"not json"}}],
			"usage":{"cost":0.004,"prompt_tokens":8,"completion_tokens":2,"total_tokens":10}
		}`))
	}))
	defer server.Close()
	restoreAPIURL(t, server.URL)

	client := &client{apiKey: "secret", model: "model", http: server.Client()}
	result, err := client.GenerateCollection("animals", []string{"en"})

	require.ErrorContains(t, err, "failed to parse")
	require.NotNil(t, result)
	assert.Equal(t, "gen-invalid", result.Usage.GenerationID)
	assert.Equal(t, 0.004, result.Usage.Cost)
}

func restoreAPIURL(t *testing.T, url string) {
	t.Helper()
	previous := apiURL
	apiURL = url
	t.Cleanup(func() { apiURL = previous })
}
