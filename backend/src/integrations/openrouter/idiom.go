package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (c *client) GenerateIdiomDescription(ctx context.Context, idiom, language string) (*GeneratedDescription, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, ErrNotConfigured
	}
	payload, err := json.Marshal(chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: fmt.Sprintf(`Describe the idiom's established figurative meanings, not its literal image. Write in %s.
Use brief, lowercase words or phrases. Separate distinct meanings with "; ".
Use ", " for enumerations of related words or phrases within one meaning.
Do not repeat the idiom, add examples, or end the description with a period.
Return only JSON in this shape: {"description": "..."}.`, language)},
			{Role: "user", Content: fmt.Sprintf("Explain the idiom %q in %s.", idiom, language)},
		},
		ResponseFormat: responseFormat{Type: "json_object"},
		Temperature:    c.temperature(0.3),
	})
	if err != nil {
		return nil, err
	}
	content, err := c.doRequestWithContext(ctx, payload)
	if err != nil {
		return nil, err
	}
	var generated GeneratedDescription
	if err := json.Unmarshal([]byte(content), &generated); err != nil {
		return nil, fmt.Errorf("failed to parse idiom description json: %w", err)
	}
	return &generated, nil
}
