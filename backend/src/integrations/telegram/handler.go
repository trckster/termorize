package telegram

import (
	"encoding/json"
	"net/http"
	"termorize/src/logger"
	"termorize/src/utils"

	"github.com/gin-gonic/gin"
)

type webhookUpdate struct {
	ID           int                `json:"update_id"`
	Message      *message           `json:"message"`
	MyChatMember *chatMemberUpdated `json:"my_chat_member"`
}

func HandleWebhook(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	logger.L().Infow("telegram webhook payload", "body", string(body))

	var update webhookUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	if update.Message == nil {
		if update.MyChatMember != nil {
			// TODO set botEnabled=true/false for this user
			logger.L().Infow("telegram bot was blocked or unblocked")
			c.Status(http.StatusOK)
			return
		} else {
			c.Status(http.StatusOK)
			return
		}
	}

	if update.Message.Text == "" {
		c.Status(http.StatusOK)
		return
	}

	response, err := sendTelegramMessage(update.Message.Chat.ID, update.Message.Text)

	if err != nil {
		if err.Error() == "blocked" {
			// TODO set botEnabled=false for this user
			logger.L().Warnw("telegram bot blocked", "error", err)
			c.Status(http.StatusOK)
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send telegram message"})
		return
	}

	if !response.OK {
		logger.L().Warnw("telegram response not ok", "response", utils.MustMarshalToString(response))
	}

	c.Status(http.StatusOK)
}
