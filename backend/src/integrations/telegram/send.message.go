package telegram

type sendMessageRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type sendMessageResponse struct {
	OK          bool    `json:"ok"`
	Result      message `json:"result"`
	Description string  `json:"description,omitempty"`
	ErrorCode   int     `json:"error_code,omitempty"`
}

func sendTelegramMessage(chatID int64, text string) (*sendMessageResponse, error) {
	messageRequest := sendMessageRequest{ChatID: chatID, Text: text}
	return CallAPI[sendMessageResponse]("sendMessage", messageRequest)
}
