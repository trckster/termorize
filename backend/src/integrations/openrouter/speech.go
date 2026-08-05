package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"termorize/src/config"
	"time"
)

const defaultSpeechAPIURL = "https://openrouter.ai/api/v1/audio/speech"
const defaultGenerationAPIURL = "https://openrouter.ai/api/v1/generation"

var speechAPIURL = defaultSpeechAPIURL
var generationAPIURL = defaultGenerationAPIURL

type SpeechResult struct {
	Audio []byte
	Usage Usage
}

type SpeechClient interface {
	GenerateSpeech(input string) (*SpeechResult, error)
}

type speechClient struct {
	apiKey    string
	model     string
	voice     string
	format    string
	referer   string
	http      *http.Client
	encodePCM func([]byte) ([]byte, error)
}

var NewSpeechClient = func(model, voice, responseFormat string) SpeechClient {
	return &speechClient{
		apiKey:    config.GetOpenRouterApiKey(),
		model:     model,
		voice:     voice,
		format:    responseFormat,
		referer:   config.GetPublicURL(),
		http:      &http.Client{Timeout: 30 * time.Second},
		encodePCM: encodePCMToMP3,
	}
}

type speechRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Voice          string `json:"voice"`
	ResponseFormat string `json:"response_format"`
}

func (c *speechClient) GenerateSpeech(input string) (*SpeechResult, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, ErrNotConfigured
	}

	payload, err := json.Marshal(speechRequest{
		Model:          c.model,
		Input:          input,
		Voice:          c.voice,
		ResponseFormat: c.format,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal openrouter speech request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, speechAPIURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to build openrouter speech request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", c.referer)
	req.Header.Set("X-Title", "Termorize")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call openrouter speech: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read openrouter speech response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter speech returned status %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	if len(body) == 0 {
		return nil, errors.New("openrouter speech returned empty audio")
	}
	audio := body
	if c.format != "mp3" {
		encodePCM := c.encodePCM
		if encodePCM == nil {
			encodePCM = encodePCMToMP3
		}

		audio, err = encodePCM(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode openrouter speech PCM as MP3: %w", err)
		}
		if len(audio) == 0 {
			return nil, errors.New("PCM encoder returned empty MP3 audio")
		}
	}

	generationID := strings.TrimSpace(resp.Header.Get("X-Generation-Id"))
	result := &SpeechResult{
		Audio: audio,
		Usage: Usage{GenerationID: generationID, Model: c.model},
	}
	if generationID == "" {
		return result, nil
	}

	usage, err := c.getGenerationUsage(generationID)
	if err != nil {
		// The audio is still usable. Keep the generation ID so the request is
		// represented in the local ledger even if metadata retrieval fails.
		return result, nil
	}
	result.Usage = usage
	return result, nil
}

func (c *speechClient) getGenerationUsage(generationID string) (Usage, error) {
	req, err := http.NewRequest(http.MethodGet, generationAPIURL, nil)
	if err != nil {
		return Usage{}, fmt.Errorf("failed to build openrouter generation request: %w", err)
	}
	query := req.URL.Query()
	query.Set("id", generationID)
	req.URL.RawQuery = query.Encode()
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return Usage{}, fmt.Errorf("failed to fetch openrouter generation usage: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Usage{}, fmt.Errorf("failed to read openrouter generation usage: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return Usage{}, fmt.Errorf("openrouter generation usage returned status %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var parsed struct {
		Data struct {
			ID                     string  `json:"id"`
			Model                  string  `json:"model"`
			TotalCost              float64 `json:"total_cost"`
			NativeTokensPrompt     int     `json:"native_tokens_prompt"`
			NativeTokensCompletion int     `json:"native_tokens_completion"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Usage{}, fmt.Errorf("failed to decode openrouter generation usage: %w", err)
	}
	model := parsed.Data.Model
	if model == "" {
		model = c.model
	}
	return Usage{
		GenerationID:     generationID,
		Model:            model,
		Cost:             parsed.Data.TotalCost,
		PromptTokens:     parsed.Data.NativeTokensPrompt,
		CompletionTokens: parsed.Data.NativeTokensCompletion,
		TotalTokens:      parsed.Data.NativeTokensPrompt + parsed.Data.NativeTokensCompletion,
	}, nil
}

func encodePCMToMP3(pcm []byte) ([]byte, error) {
	if len(pcm) == 0 {
		return nil, errors.New("cannot encode empty PCM audio")
	}
	if len(pcm)%2 != 0 {
		return nil, errors.New("PCM audio has an incomplete 16-bit sample")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	command := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-f", "s16le",
		"-ar", "24000",
		"-ac", "1",
		"-i", "pipe:0",
		"-codec:a", "libmp3lame",
		"-b:a", "64k",
		"-f", "mp3",
		"pipe:1",
	)
	command.Stdin = bytes.NewReader(pcm)

	var output bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &output
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("ffmpeg timed out: %w", ctx.Err())
		}
		if details := strings.TrimSpace(stderr.String()); details != "" {
			return nil, fmt.Errorf("ffmpeg failed: %w: %s", err, truncate(details, 300))
		}
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}

	return output.Bytes(), nil
}
