package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	nethttp "net/http"
	"termorize/src/config"
	"time"
)

var ErrBlocked = errors.New("blocked")

var apiBaseURL = "https://api.telegram.org"

type apiError struct {
	StatusCode  int
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

func (e *apiError) Error() string {
	if e.Description == "" {
		return fmt.Sprintf("telegram api returned status %d", e.StatusCode)
	}

	return fmt.Sprintf("telegram api returned status %d: %s", e.StatusCode, e.Description)
}

func parseAPIError(statusCode int, body []byte) error {
	apiErr := &apiError{StatusCode: statusCode}
	if err := json.Unmarshal(body, apiErr); err != nil {
		apiErr.Description = string(bytes.TrimSpace(body))
	}
	return apiErr
}

func SetAPIBaseURLForTest(url string) (restore func()) {
	previous := apiBaseURL
	apiBaseURL = url
	return func() { apiBaseURL = previous }
}

func CallAPI[Response any](action string, requestBody any) (*Response, error) {
	encodedBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/bot%s/%s", apiBaseURL, config.GetTelegramBotToken(), action)

	req, err := nethttp.NewRequest(nethttp.MethodPost, url, bytes.NewReader(encodedBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &nethttp.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, ErrBlocked
	}

	if resp.StatusCode >= nethttp.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return nil, parseAPIError(resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
