package telegram

import (
	"encoding/json"
	"net/http"
	"termorize/src/logger"
	"termorize/src/services"
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

	if update.MyChatMember != nil {
		handleMyChatMemberUpdate(update.MyChatMember)
		c.Status(http.StatusOK)
		return
	}

	if update.Message == nil {
		c.Status(http.StatusOK)
		return
	}

	if update.Message.Chat.Type == Private {
		telegramID, username, firstName, lastName := extractMessageUser(update.Message)

		if err := services.EnsureUserByTelegramID(telegramID, username, firstName, lastName); err != nil {
			logger.L().Warnw("failed to ensure telegram user", "error", err, "telegram_id", telegramID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ensure user"})
			return
		}

		if err := services.UpdateUserTelegramBotEnabled(telegramID, true); err != nil {
			logger.L().Warnw("failed to enable telegram bot for user", "error", err, "telegram_id", telegramID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user settings"})
			return
		}
	}

	if update.Message.Text == "" {
		c.Status(http.StatusOK)
		return
	}

	replyText := update.Message.Text
	if update.Message.Chat.Type != Private {
		replyText = "Nah... Don't feel like answering here rn"
	}

	response, err := sendTelegramMessage(update.Message.Chat.ID, replyText)

	if err != nil {
		if err.Error() == "blocked" {
			if update.Message.Chat.Type == Private {
				telegramID, _, _, _ := extractMessageUser(update.Message)

				if updateErr := services.UpdateUserTelegramBotEnabled(telegramID, false); updateErr != nil {
					logger.L().Warnw("failed to disable telegram bot for user", "error", updateErr, "telegram_id", telegramID)
				}
			}

			logger.L().Infow("telegram bot blocked", "error", err)
			c.Status(http.StatusOK)
			return
		}

		logger.L().Warnw("failed to send telegram message", "error", err, "text", replyText)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send telegram message"})
		return
	}

	if !response.OK {
		logger.L().Warnw("telegram response not ok", "response", utils.MustMarshalToString(response))
	}

	c.Status(http.StatusOK)
}

func handleMyChatMemberUpdate(chatMemberUpdate *chatMemberUpdated) {
	if chatMemberUpdate.Chat == nil || chatMemberUpdate.Chat.Type != Private {
		return
	}

	oldChatMember := chatMemberUpdate.OldChatMember
	newChatMember := chatMemberUpdate.NewChatMember

	if oldChatMember == nil || newChatMember == nil {
		return
	}

	chatID := chatMemberUpdate.Chat.ID

	if oldChatMember.Status == Member && newChatMember.Status == Kicked {
		if err := services.UpdateUserTelegramBotEnabled(chatID, false); err != nil {
			logger.L().Warnw("failed to disable telegram bot for user", "error", err, "chat_id", chatID)
		}

		logger.L().Infow("telegram bot blocked", "chat_id", chatID)
		return
	}

	if oldChatMember.Status == Kicked && newChatMember.Status == Member {
		if err := services.UpdateUserTelegramBotEnabled(chatID, true); err != nil {
			logger.L().Warnw("failed to enable telegram bot for user", "error", err, "chat_id", chatID)
		}

		logger.L().Infow("telegram bot unblocked", "chat_id", chatID)
	}
}

func extractMessageUser(message *message) (int64, string, string, string) {
	telegramID := message.Chat.ID
	username := message.Chat.Username
	firstName := message.Chat.FirstName
	lastName := ""

	if message.From != nil {
		telegramID = message.From.ID
		username = message.From.Username
		firstName = message.From.FirstName
		lastName = message.From.LastName
	}

	return telegramID, username, firstName, lastName
}
