package openrouter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"termorize/src/config"
	"time"
)

var apiURL = "https://openrouter.ai/api/v1/chat/completions"

var ErrNotConfigured = errors.New("openrouter api key is not configured")

type GeneratedTranslation struct {
	Original            string `json:"original"`
	OriginalLanguage    string `json:"original_language"`
	Translation         string `json:"translation"`
	TranslationLanguage string `json:"translation_language"`
}

type GeneratedCollection struct {
	Title        string                 `json:"title"`
	Translations []GeneratedTranslation `json:"translations"`
}

type Usage struct {
	GenerationID     string
	Model            string
	Cost             float64
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type GenerationResult struct {
	Collection *GeneratedCollection
	Usage      Usage
}

type Client interface {
	GenerateCollection(prompt string, allowedLanguages []string) (*GenerationResult, error)
}

type client struct {
	apiKey  string
	model   string
	referer string
	http    *http.Client
}

var NewClient = func() Client {
	return &client{
		apiKey:  config.GetOpenRouterApiKey(),
		model:   config.GetOpenRouterModel(),
		referer: config.GetPublicURL(),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	ResponseFormat responseFormat `json:"response_format"`
	Temperature    float64        `json:"temperature"`
}

type chatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Usage struct {
		Cost             float64 `json:"cost"`
		PromptTokens     int     `json:"prompt_tokens"`
		CompletionTokens int     `json:"completion_tokens"`
		TotalTokens      int     `json:"total_tokens"`
	} `json:"usage"`
}

func (c *client) GenerateCollection(prompt string, allowedLanguages []string) (*GenerationResult, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, ErrNotConfigured
	}

	systemPrompt := buildSystemPrompt(allowedLanguages)

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		ResponseFormat: responseFormat{Type: "json_object"},
		Temperature:    0.3,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal openrouter request: %w", err)
	}

	response, err := c.doRequest(payload)
	if response == nil {
		return nil, err
	}
	model := response.Model
	if model == "" {
		model = c.model
	}
	result := &GenerationResult{Usage: Usage{
		GenerationID:     response.ID,
		Model:            model,
		Cost:             response.Usage.Cost,
		PromptTokens:     response.Usage.PromptTokens,
		CompletionTokens: response.Usage.CompletionTokens,
		TotalTokens:      response.Usage.TotalTokens,
	}}
	if err != nil {
		return result, err
	}

	var generated GeneratedCollection
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &generated); err != nil {
		return result, fmt.Errorf("failed to parse generated collection json: %w", err)
	}
	result.Collection = &generated
	return result, nil
}

func (c *client) doRequest(payload []byte) (*chatResponse, error) {
	httpReq, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to build openrouter request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", c.referer)
	httpReq.Header.Set("X-Title", "Termorize")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call openrouter: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read openrouter response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter returned status %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode openrouter response: %w", err)
	}

	if parsed.Error != nil {
		return &parsed, fmt.Errorf("openrouter error: %s", parsed.Error.Message)
	}

	if len(parsed.Choices) == 0 {
		return &parsed, errors.New("openrouter returned no choices")
	}

	parsed.Choices[0].Message.Content = strings.TrimSpace(parsed.Choices[0].Message.Content)
	if parsed.Choices[0].Message.Content == "" {
		return &parsed, errors.New("openrouter returned empty content")
	}

	return &parsed, nil
}

func buildSystemPrompt(allowedLanguages []string) string {
	langList := strings.Join(allowedLanguages, ", ")

	return "You generate vocabulary for a language-learning app. " +
		"MANDATORY RULES — never ignore these: " +
		"1) Every noun in Italian, German, Spanish, or French MUST include its definite article: la gamba, das Bein, la pierna, la jambe. " +
		"2) Every verb MUST be infinitive only: прыгать, saltare, springen, sauter. " +
		"3) Never add articles to English nouns. " +
		"4) Output ONLY this JSON shape with no markdown: " +
		`{"title": string, "translations": [{"original": string, "original_language": string, "translation": string, "translation_language": string}]}. ` +
		"5) CRITICAL LANGUAGE RULE: First, analyze the user's prompt and determine exactly which languages they want. " +
		"Use ONLY those languages for EVERY SINGLE translation in the output. " +
		"original_language and translation_language for every item must both be from the set of languages the user requested. " +
		"Never introduce a language the user did not ask for. " +
		"If the user prompt does not mention or imply specific languages, you may use any from this allowed list: " + langList + ". " +
		"original_language != translation_language per item. " +
		"Short descriptive title. Honor count, languages, topic. No extra text."
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
