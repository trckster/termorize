package telegram

import (
	"strings"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
)

func handleStateMessage(message *message) (bool, error) {
	if strings.HasPrefix(strings.TrimSpace(message.Text), "/") {
		return false, nil
	}

	if strings.EqualFold(strings.TrimSpace(message.Text), telegramButtonMenuCancel) {
		telegramID, _, _, _ := extractMessageUser(message)
		if _, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone); err != nil {
			return true, err
		}

		return true, SendMessageWithInlineKeyboardMarkdown(message.Chat.ID, telegramTextMenu, menuKeyboard)
	}

	telegramID, _, _, _ := extractMessageUser(message)
	user, err := services.GetUserByTelegramID(telegramID)
	if err != nil {
		return false, err
	}

	if user == nil {
		return false, nil
	}

	switch user.TelegramState {
	case enums.TelegramStateAddingVocabulary:
		return true, handleAddingVocabularyMessage(message, user, telegramID)
	case enums.TelegramStateDeletingVocabulary:
		return true, handleDeletingVocabularyMessage(message, user.ID, telegramID)
	default:
		return false, nil
	}
}

func handleAddingVocabularyMessage(message *message, user *models.User, telegramID int64) error {
	nativeWord, learningWord, ok := parseVocabularyPair(message.Text)
	if !ok {
		return SendMessage(message.Chat.ID, telegramTextAddVocabularyInvalid)
	}

	_, err := services.CreateVocabulary(user.ID, services.CreateVocabularyRequest{
		Original:            learningWord,
		Translation:         nativeWord,
		OriginalLanguage:    user.Settings.MainLearningLanguage,
		TranslationLanguage: user.Settings.NativeLanguage,
	})
	if err != nil {
		if services.IsTranslationAlreadyExistsError(err) {
			return SendMessage(message.Chat.ID, telegramTextAddVocabularyExists)
		}

		return err
	}

	if _, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone); err != nil {
		return err
	}

	return SendMessage(message.Chat.ID, telegramTextAddVocabularyDone)
}

func handleDeletingVocabularyMessage(message *message, userID uint, telegramID int64) error {
	deleted, err := services.DeleteVocabularyByWord(userID, message.Text)
	if err != nil {
		return err
	}

	if !deleted {
		return SendMessage(message.Chat.ID, telegramTextDeleteNotFound)
	}

	if _, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone); err != nil {
		return err
	}

	return SendMessage(message.Chat.ID, telegramTextDeleteCompleted)
}

func parseVocabularyPair(text string) (string, string, bool) {
	parts := strings.SplitN(text, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	nativeWord := strings.TrimSpace(parts[0])
	learningWord := strings.TrimSpace(parts[1])
	if nativeWord == "" || learningWord == "" {
		return "", "", false
	}

	return nativeWord, learningWord, true
}
