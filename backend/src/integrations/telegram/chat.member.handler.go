package telegram

import (
	"termorize/src/logger"
	"termorize/src/services"
)

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
