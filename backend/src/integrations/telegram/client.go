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

func CallAPI[Response any](action string, requestBody any) (*Response, error) {
	encodedBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", config.GetTelegramBotToken(), action)

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
		return nil, errors.New("blocked")
	}

	if resp.StatusCode >= nethttp.StatusBadRequest {
		return nil, fmt.Errorf("telegram api returned status %d", resp.StatusCode)
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
