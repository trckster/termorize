package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	nethttp "net/http"
	"net/textproto"
	"strconv"
	"termorize/src/config"
	"time"
)

type sendAudioRequest struct {
	ChatID int64  `json:"chat_id"`
	Audio  string `json:"audio"`
	Title  string `json:"title,omitempty"`
}

type sendAudioResponse struct {
	OK          bool    `json:"ok"`
	Result      message `json:"result"`
	Description string  `json:"description,omitempty"`
	ErrorCode   int     `json:"error_code,omitempty"`
}

func SendAudioByFileID(chatID int64, fileID, title string) (string, error) {
	response, err := CallAPI[sendAudioResponse]("sendAudio", sendAudioRequest{ChatID: chatID, Audio: fileID, Title: title})
	if err != nil {
		return "", err
	}

	return parseSendAudioResponse(response)
}

func SendAudioMP3(chatID int64, audio []byte, mimeType, title string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("chat_id", strconv.FormatInt(chatID, 10)); err != nil {
		return "", err
	}
	if title != "" {
		if err := writer.WriteField("title", title); err != nil {
			return "", err
		}
	}

	if mimeType == "" {
		mimeType = "audio/mpeg"
	}
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="audio"; filename="pronunciation.mp3"`)
	partHeader.Set("Content-Type", mimeType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(audio); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/bot%s/sendAudio", apiBaseURL, config.GetTelegramBotToken())
	req, err := nethttp.NewRequest(nethttp.MethodPost, url, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &nethttp.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == nethttp.StatusForbidden {
		return "", ErrBlocked
	}
	if resp.StatusCode >= nethttp.StatusBadRequest {
		return "", parseAPIError(resp.StatusCode, responseBody)
	}

	var response sendAudioResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", err
	}

	return parseSendAudioResponse(&response)
}

func parseSendAudioResponse(response *sendAudioResponse) (string, error) {
	if response == nil {
		return "", errors.New("telegram sendAudio returned no response")
	}
	if !response.OK {
		return "", &apiError{
			StatusCode:  nethttp.StatusOK,
			ErrorCode:   response.ErrorCode,
			Description: response.Description,
		}
	}
	if response.Result.Audio == nil || response.Result.Audio.FileID == "" {
		return "", errors.New("telegram sendAudio response is missing audio file_id")
	}

	return response.Result.Audio.FileID, nil
}
