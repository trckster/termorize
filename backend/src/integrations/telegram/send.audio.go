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
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/models"
	"termorize/src/services"
	"time"

	"github.com/google/uuid"
)

type sendAudioRequest struct {
	ChatID      int64                 `json:"chat_id"`
	Audio       string                `json:"audio"`
	Title       string                `json:"title,omitempty"`
	Caption     string                `json:"caption,omitempty"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *inlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type sendAudioResponse struct {
	OK          bool    `json:"ok"`
	Result      message `json:"result"`
	Description string  `json:"description,omitempty"`
	ErrorCode   int     `json:"error_code,omitempty"`
}

type audioSendResult struct {
	FileID    string
	MessageID int64
}

type audioSendOptions struct {
	Title       string
	Caption     string
	ParseMode   string
	ReplyMarkup *inlineKeyboardMarkup
}

func SendAudioByFileID(chatID int64, fileID, title string) (string, error) {
	result, err := sendAudioByFileID(chatID, fileID, audioSendOptions{Title: title})
	if err != nil {
		return "", err
	}
	return result.FileID, nil
}

func SendAudioMP3(chatID int64, audio []byte, mimeType, title string) (string, error) {
	result, err := sendAudioMP3(chatID, audio, mimeType, audioSendOptions{Title: title})
	if err != nil {
		return "", err
	}
	return result.FileID, nil
}

func sendAudioByFileID(chatID int64, fileID string, options audioSendOptions) (*audioSendResult, error) {
	response, err := CallAPI[sendAudioResponse]("sendAudio", sendAudioRequest{
		ChatID:      chatID,
		Audio:       fileID,
		Title:       options.Title,
		Caption:     options.Caption,
		ParseMode:   options.ParseMode,
		ReplyMarkup: options.ReplyMarkup,
	})
	if err != nil {
		return nil, err
	}
	return parseSendAudioResponse(response)
}

func sendAudioMP3(chatID int64, audio []byte, mimeType string, options audioSendOptions) (*audioSendResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("chat_id", strconv.FormatInt(chatID, 10)); err != nil {
		return nil, err
	}
	if options.Title != "" {
		if err := writer.WriteField("title", options.Title); err != nil {
			return nil, err
		}
	}
	if options.Caption != "" {
		if err := writer.WriteField("caption", options.Caption); err != nil {
			return nil, err
		}
	}
	if options.ParseMode != "" {
		if err := writer.WriteField("parse_mode", options.ParseMode); err != nil {
			return nil, err
		}
	}
	if options.ReplyMarkup != nil {
		replyMarkup, err := json.Marshal(options.ReplyMarkup)
		if err != nil {
			return nil, err
		}
		if err := writer.WriteField("reply_markup", string(replyMarkup)); err != nil {
			return nil, err
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
		return nil, err
	}
	if _, err := part.Write(audio); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/bot%s/sendAudio", apiBaseURL, config.GetTelegramBotToken())
	req, err := nethttp.NewRequest(nethttp.MethodPost, url, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &nethttp.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == nethttp.StatusForbidden {
		return nil, ErrBlocked
	}
	if resp.StatusCode >= nethttp.StatusBadRequest {
		return nil, parseAPIError(resp.StatusCode, responseBody)
	}

	var response sendAudioResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, err
	}

	return parseSendAudioResponse(&response)
}

func parseSendAudioResponse(response *sendAudioResponse) (*audioSendResult, error) {
	if response == nil {
		return nil, errors.New("telegram sendAudio returned no response")
	}
	if !response.OK {
		return nil, &apiError{
			StatusCode:  nethttp.StatusOK,
			ErrorCode:   response.ErrorCode,
			Description: response.Description,
		}
	}
	if response.Result.Audio == nil || response.Result.Audio.FileID == "" {
		return nil, errors.New("telegram sendAudio response is missing audio file_id")
	}

	return &audioSendResult{FileID: response.Result.Audio.FileID, MessageID: response.Result.MessageID}, nil
}

func SendAudioExerciseMessage(
	chatID int64,
	exerciseID uuid.UUID,
	pronunciation *models.WordPronunciation,
	spokenLanguage enums.Language,
	answerLanguage enums.Language,
	showIgnoreLanguageSuggestion bool,
	texts BotTexts,
) (*int64, error) {
	if pronunciation == nil {
		return nil, errors.New("audio exercise pronunciation is missing")
	}
	options := audioSendOptions{
		Title:   texts.AudioExerciseTitle,
		Caption: buildAudioExerciseCaption(answerLanguage, texts, false),
		ReplyMarkup: &inlineKeyboardMarkup{InlineKeyboard: buildAudioExerciseKeyboard(
			exerciseID,
			spokenLanguage,
			showIgnoreLanguageSuggestion,
			texts,
		)},
	}

	var result *audioSendResult
	var err error
	if pronunciation.TelegramFileID != nil {
		result, err = sendAudioByFileID(chatID, *pronunciation.TelegramFileID, options)
		if err != nil {
			logger.L().Warnw("cached telegram audio exercise file id was rejected", "error", err, "pronunciation_id", pronunciation.ID)
		}
	}
	if result == nil {
		if len(pronunciation.Audio) == 0 {
			pronunciation.Audio, pronunciation.MIMEType, err = services.GetWordPronunciationAudio(pronunciation.ID)
			if err != nil {
				return nil, err
			}
		}
		result, err = sendAudioMP3(chatID, pronunciation.Audio, pronunciation.MIMEType, options)
		if err != nil {
			return nil, err
		}
		if err := services.SetWordPronunciationTelegramFileID(pronunciation.ID, result.FileID); err != nil {
			logger.L().Warnw("failed to cache telegram audio exercise file id", "error", err, "pronunciation_id", pronunciation.ID)
		}
	}

	return &result.MessageID, nil
}

func buildAudioExerciseCaption(answerLanguage enums.Language, texts BotTexts, cancelled bool) string {
	captionFormat := texts.AudioExerciseCaptionFormat
	if cancelled {
		captionFormat = texts.AudioExerciseCancelledCaptionFormat
	}

	return fmt.Sprintf(captionFormat, answerLanguage.Flag()+" "+localizedLanguageName(answerLanguage, texts))
}

func editCancelledAudioExerciseMessage(
	chatID int64,
	messageID int64,
	answerLanguage enums.Language,
	exerciseID uuid.UUID,
	spokenLanguage enums.Language,
	texts BotTexts,
) error {
	return editMessageCaptionTolerant(editMessageCaptionRequest{
		ChatID:    chatID,
		MessageID: messageID,
		Caption:   buildAudioExerciseCaption(answerLanguage, texts, true),
		ReplyMarkup: &inlineKeyboardMarkup{InlineKeyboard: buildAudioUndoKeyboard(
			exerciseID,
			spokenLanguage,
			texts,
		)},
	})
}
