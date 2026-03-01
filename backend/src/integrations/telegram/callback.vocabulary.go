package telegram

import (
	"strings"
	"termorize/src/services"

	"github.com/google/uuid"
)

func handleVocabularyCallback(callback *callbackQuery, payload []string) error {
	if callback.Message == nil {
		return nil
	}

	if len(payload) < 1 {
		return nil
	}

	action := payload[0]
	switch action {
	case vocabularyActionAdd:
		return handleVocabularyAddCallback(callback, payload[1:])
	case vocabularyActionDelete:
		return handleVocabularyDeleteCallback(callback, payload[1:])
	default:
		return nil
	}
}

func handleVocabularyAddCallback(callback *callbackQuery, payload []string) error {
	if len(payload) != 1 {
		return nil
	}

	translationID, err := uuid.Parse(payload[0])
	if err != nil {
		return nil
	}

	user, err := services.GetUserByTelegramID(callback.From.ID)
	if err != nil {
		return err
	}

	if user == nil {
		return nil
	}

	_, err = services.CreateVocabularyByTranslation(user.ID, translationID)
	if err != nil && !services.IsTranslationAlreadyExistsError(err) {
		return err
	}

	updatedText := callback.Message.Text + telegramTextVocabularyManualAddedSuffix
	return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, updatedText, [][]inlineKeyboardButton{})
}

func handleVocabularyDeleteCallback(callback *callbackQuery, payload []string) error {
	if len(payload) != 1 {
		return nil
	}

	vocabularyID, err := uuid.Parse(payload[0])
	if err != nil {
		return nil
	}

	user, err := services.GetUserByTelegramID(callback.From.ID)
	if err != nil {
		return err
	}

	if user == nil {
		return nil
	}

	err = services.DeleteVocabulary(user.ID, vocabularyID)
	if err != nil && err.Error() != "vocabulary item not found" {
		return err
	}

	updatedText := strings.TrimSuffix(callback.Message.Text, telegramTextVocabularyAutoAddedSuffix)
	return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, updatedText, [][]inlineKeyboardButton{})
}
