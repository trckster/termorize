package telegram

import (
	"fmt"
	"strings"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"unicode"
)

func handlePlainTranslationMessage(message *message) (bool, error) {
	telegramID, _, _, _ := extractMessageUser(message)
	user, err := services.GetUserByTelegramID(telegramID)
	if err != nil {
		return false, err
	}

	if user == nil {
		return false, nil
	}

	word := strings.TrimSpace(message.Text)
	if word == "" {
		return true, nil
	}

	sourceLanguage, targetLanguage, err := detectMessageTranslationLanguages(user, word)
	if err != nil {
		return true, err
	}

	translationResult, err := services.TranslateWithTranslation(word, sourceLanguage, targetLanguage)
	if err != nil {
		return true, err
	}

	baseText := buildVocabularyTranslationText(sourceLanguage, translationResult.SourceWord, translationResult.TranslatedWord, targetLanguage)
	if len(strings.Fields(word)) < 5 {
		vocabulary, createErr := services.CreateVocabularyByTranslation(user.ID, translationResult.TranslationID)
		if createErr == nil {
			return true, SendMessageWithInlineKeyboard(message.Chat.ID, baseText+telegramTextVocabularyAutoAddedSuffix, buildVocabularyDeleteKeyboard(vocabulary.ID.String()))
		}

		if services.IsTranslationAlreadyExistsError(createErr) {
			return true, SendMessage(message.Chat.ID, baseText)
		}

		return true, createErr
	}

	return true, SendMessageWithInlineKeyboard(message.Chat.ID, baseText, buildVocabularyAddKeyboard(translationResult.TranslationID.String()))
}

func detectMessageTranslationLanguages(user *models.User, text string) (enums.Language, enums.Language, error) {
	nativeLanguage := user.Settings.NativeLanguage
	learningLanguage := user.Settings.MainLearningLanguage

	detectedLanguage, matchedSupportedLanguage, err := services.DetectLanguage(text)
	if err == nil && matchedSupportedLanguage {
		if detectedLanguage == nativeLanguage {
			return nativeLanguage, learningLanguage, nil
		}

		if detectedLanguage == learningLanguage {
			return learningLanguage, nativeLanguage, nil
		}
	}

	if containsCyrillic(text) {
		if nativeLanguage == enums.LanguageRu {
			return nativeLanguage, learningLanguage, nil
		}

		if learningLanguage == enums.LanguageRu {
			return learningLanguage, nativeLanguage, nil
		}
	}

	return nativeLanguage, learningLanguage, nil
}

func buildVocabularyTranslationText(sourceLanguage enums.Language, sourceWord string, translatedWord string, targetLanguage enums.Language) string {
	return fmt.Sprintf("%s %s — %s %s", sourceLanguage.Flag(), sourceWord, translatedWord, targetLanguage.Flag())
}

func containsCyrillic(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}

	return false
}
