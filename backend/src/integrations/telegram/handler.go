package telegram

import (
	"encoding/json"
	"net/http"
	"termorize/src/logger"

	"github.com/gin-gonic/gin"
)

type webhookUpdate struct {
	ID            int                `json:"update_id"`
	Message       *message           `json:"message"`
	MyChatMember  *chatMemberUpdated `json:"my_chat_member"`
	CallbackQuery *callbackQuery     `json:"callback_query"`
}

func HandleWebhook(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	var update webhookUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	logger.L().Debugw("telegram webhook update", "update_id", update.ID)

	if err := handleUpdate(&update); err != nil {
		logger.L().Warnw("failed to process telegram webhook update", "error", err, "update_id", update.ID)
		c.JSON(http.StatusOK, gin.H{"error": "failed to process update"})
		return
	}

	c.Status(http.StatusOK)
}

func handleUpdate(update *webhookUpdate) error {
	if update.MyChatMember != nil {
		handleMyChatMemberUpdate(update.MyChatMember)
		return nil
	}

	if update.CallbackQuery != nil {
		return handleCallbackQuery(update.CallbackQuery)
	}

	if update.Message == nil {
		return nil
	}

	return handleMessage(update.Message)
}
