package telegram

import (
	"errors"
	"termorize/src/logger"
	"termorize/src/services"

	"github.com/google/uuid"
)

type sendMessageRequest struct {
	ChatID      int64                 `json:"chat_id"`
	Text        string                `json:"text"`
	ReplyMarkup *inlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type sendMessageResponse struct {
	OK          bool    `json:"ok"`
	Result      message `json:"result"`
	Description string  `json:"description,omitempty"`
	ErrorCode   int     `json:"error_code,omitempty"`
}

type inlineKeyboardMarkup struct {
	InlineKeyboard [][]inlineKeyboardButton `json:"inline_keyboard"`
}

type inlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

type answerCallbackQueryRequest struct {
	CallbackQueryID string `json:"callback_query_id"`
}

type answerCallbackQueryResponse struct {
	OK bool `json:"ok"`
}

type editMessageReplyMarkupRequest struct {
	ChatID      int64                 `json:"chat_id"`
	MessageID   int64                 `json:"message_id"`
	ReplyMarkup *inlineKeyboardMarkup `json:"reply_markup"`
}

type editMessageReplyMarkupResponse struct {
	OK bool `json:"ok"`
}

func SendMessage(chatID int64, text string) error {
	messageRequest := sendMessageRequest{ChatID: chatID, Text: text}
	return sendMessage(messageRequest)
}

func SendExerciseMessage(chatID int64, text string, exerciseID uuid.UUID, questionType string) error {
	messageRequest := sendMessageRequest{
		ChatID: chatID,
		Text:   text,
		ReplyMarkup: &inlineKeyboardMarkup{InlineKeyboard: [][]inlineKeyboardButton{{
			{Text: "IDK", CallbackData: "exercise:idk:" + exerciseID.String() + ":" + questionType},
		}}},
	}

	return sendMessage(messageRequest)
}

func answerTelegramCallbackQuery(callbackQueryID string) error {
	_, err := CallAPI[answerCallbackQueryResponse]("answerCallbackQuery", answerCallbackQueryRequest{CallbackQueryID: callbackQueryID})
	return err
}

func sendMessage(messageRequest sendMessageRequest) error {
	response, err := CallAPI[sendMessageResponse]("sendMessage", messageRequest)
	if err != nil {
		if errors.Is(err, ErrBlocked) {
			if updateErr := services.UpdateUserTelegramBotEnabled(messageRequest.ChatID, false); updateErr != nil {
				logger.L().Warnw("failed to disable telegram bot for blocked user", "error", updateErr, "telegram_id", messageRequest.ChatID)
			}
			return nil
		}

		return err
	}

	if !response.OK {
		return errors.New("telegram response not ok")
	}

	return nil
}

func removeMessageInlineKeyboard(chatID int64, messageID int64) error {
	request := editMessageReplyMarkupRequest{
		ChatID:      chatID,
		MessageID:   messageID,
		ReplyMarkup: &inlineKeyboardMarkup{InlineKeyboard: [][]inlineKeyboardButton{}},
	}

	response, err := CallAPI[editMessageReplyMarkupResponse]("editMessageReplyMarkup", request)
	if err != nil {
		return err
	}

	if !response.OK {
		return errors.New("telegram editMessageReplyMarkup response not ok")
	}

	return nil
}
