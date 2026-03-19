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

	telegramID, _, _, _ := extractMessageUser(message)
	t := getBotTextsForTelegramID(telegramID)

	if strings.EqualFold(strings.TrimSpace(message.Text), t.ButtonCancel) {
		if _, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone); err != nil {
			return true, err
		}

		return true, SendMessageWithInlineKeyboardMarkdown(message.Chat.ID, t.Menu, getMenuKeyboard(t))
	}

	user, err := services.GetUserByTelegramID(telegramID)
	if err != nil {
		return false, err
	}

	if user == nil {
		return false, nil
	}

	switch user.TelegramState {
	case enums.TelegramStateAddingVocabulary:
		return true, handleAddingVocabularyMessage(message, user, telegramID, t)
	case enums.TelegramStateDeletingVocabulary:
		return true, handleDeletingVocabularyMessage(message, user.ID, telegramID, t)
	default:
		return false, nil
	}
}

func handleAddingVocabularyMessage(message *message, user *models.User, telegramID int64, t BotTexts) error {
	if strings.Count(message.Text, ":") > 1 {
		return SendMessage(message.Chat.ID, t.AddVocabularyTooManyColons)
	}

	sourceWord, targetWord, ok := parseVocabularyPair(message.Text)
	if !ok {
		return SendMessage(message.Chat.ID, t.AddVocabularyInvalid)
	}

	_, err := services.CreateVocabulary(user.ID, services.CreateVocabularyRequest{
		Original:            sourceWord,
		Translation:         targetWord,
		OriginalLanguage:    user.Settings.TranslationSourceLanguage,
		TranslationLanguage: user.Settings.TranslationTargetLanguage,
	})
	if err != nil {
		if services.VocabularyAlreadyExistsError(err) {
			return SendMessage(message.Chat.ID, t.AddVocabularyExists)
		}

		return err
	}

	if _, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone); err != nil {
		return err
	}

	return SendMessage(message.Chat.ID, t.AddVocabularyDone)
}

func handleDeletingVocabularyMessage(message *message, userID uint, telegramID int64, t BotTexts) error {
	deleted, err := services.DeleteVocabularyByWord(userID, message.Text)
	if err != nil {
		return err
	}

	if !deleted {
		return SendMessage(message.Chat.ID, t.DeleteNotFound)
	}

	if _, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone); err != nil {
		return err
	}

	return SendMessage(message.Chat.ID, t.DeleteCompleted)
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
