package telegram

import (
	"encoding/json"
	"net/http"
	"strings"
	"termorize/src/logger"
	"termorize/src/services"
	"termorize/src/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	if update.CallbackQuery != nil {
		if err := handleCallbackQuery(update.CallbackQuery); err != nil {
			logger.L().Warnw("failed to handle callback query", "error", err, "callback_query", utils.MustMarshalToString(update.CallbackQuery))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process callback"})
			return
		}

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

	if err := SendMessage(update.Message.Chat.ID, replyText); err != nil {
		logger.L().Warnw("failed to send telegram message", "error", err, "text", replyText)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send telegram message"})
		return
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

func handleCallbackQuery(callback *callbackQuery) error {
	if callback == nil {
		return nil
	}

	if callback.ID != "" {
		if err := answerTelegramCallbackQuery(callback.ID); err != nil {
			logger.L().Warnw("failed to answer callback query", "error", err, "callback_id", callback.ID)
		}
	}

	if callback.From == nil {
		return nil
	}

	action, exerciseID, questionType, ok := parseExerciseCallbackData(callback.Data)
	if !ok || action != "idk" {
		return nil
	}

	if callback.Message != nil {
		if err := removeMessageInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID); err != nil {
			logger.L().Warnw("failed to remove inline keyboard", "error", err, "chat_id", callback.Message.Chat.ID, "message_id", callback.Message.MessageID)
		}
	}

	updated, err := services.FailExercise(exerciseID)
	if err != nil || !updated {
		return err
	}

	words, err := services.GetExerciseWordsByTelegram(exerciseID, callback.From.ID)
	if err != nil {
		return err
	}

	if words == nil {
		return nil
	}

	answerText := buildIDKAnswer(words.OriginalWord, words.TranslationWord, questionType)
	return SendMessage(callback.From.ID, answerText)
}

func parseExerciseCallbackData(data string) (string, uuid.UUID, string, bool) {
	parts := strings.Split(data, ":")
	if len(parts) != 4 || parts[0] != "exercise" {
		return "", uuid.Nil, "", false
	}

	exerciseID, err := uuid.Parse(parts[2])
	if err != nil {
		return "", uuid.Nil, "", false
	}

	if parts[3] != "o2t" && parts[3] != "t2o" {
		return "", uuid.Nil, "", false
	}

	return parts[1], exerciseID, parts[3], true
}

func buildIDKAnswer(originalWord string, translationWord string, questionType string) string {
	if questionType == "t2o" {
		return "Correct original word: " + originalWord
	}

	return "Correct translation: " + translationWord
}
