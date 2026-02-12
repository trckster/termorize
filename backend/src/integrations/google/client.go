package google

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"termorize/src/config"
)

type TranslateClient interface {
	Translate(text, sourceLang, targetLang string) (string, error)
}

type translateClient struct {
	apiKey string
}

func NewTranslateClient() TranslateClient {
	return &translateClient{
		apiKey: config.GetGoogleApiKey(),
	}
}

type translateResponse struct {
	Data struct {
		Translations []struct {
			TranslatedText string `json:"translatedText"`
		} `json:"translations"`
	} `json:"data"`
}

func (c *translateClient) Translate(text, sourceLang, targetLang string) (string, error) {
	baseURL := "https://translation.googleapis.com/language/translate/v2"

	params := url.Values{}
	params.Add("key", c.apiKey)
	params.Add("q", text)
	params.Add("source", sourceLang)
	params.Add("target", targetLang)
	params.Add("format", "text")

	resp, err := http.PostForm(baseURL, params)
	if err != nil {
		return "", fmt.Errorf("failed to call Google Translate API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google translate API returned status %d", resp.StatusCode)
	}

	var result translateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Data.Translations) == 0 {
		return "", errors.New("no translation found")
	}

	return result.Data.Translations[0].TranslatedText, nil
}
