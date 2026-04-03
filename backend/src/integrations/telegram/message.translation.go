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

	t := GetBotTexts(user.Settings.SystemLanguage)

	word := strings.TrimSpace(message.Text)
	if word == "" {
		return true, nil
	}

	sourceLanguage, targetLanguage, err := detectMessageTranslationLanguages(user, word)
	if err != nil {
		return true, err
	}

	translationResult, err := services.Translate(word, sourceLanguage, targetLanguage)
	if err != nil {
		return true, err
	}

	baseText := buildVocabularyTranslationText(sourceLanguage, translationResult.SourceWord, translationResult.TranslatedWord, targetLanguage)
	translationMatchesSource := strings.EqualFold(
		strings.TrimSpace(translationResult.SourceWord),
		strings.TrimSpace(translationResult.TranslatedWord),
	)

	if len(strings.Fields(word)) < 5 && !translationMatchesSource {
		vocabulary, createErr := services.CreateVocabularyByTranslation(user.ID, translationResult.TranslationID)
		if createErr == nil {
			return true, SendMessageWithInlineKeyboard(message.Chat.ID, baseText+t.VocabularyAutoAddedSuffix, buildVocabularyDeleteKeyboard(vocabulary.ID.String(), t))
		}

		if services.VocabularyAlreadyExistsError(createErr) {
			return true, SendMessage(message.Chat.ID, baseText)
		}

		return true, createErr
	}

	if translationMatchesSource {
		return true, SendMessage(message.Chat.ID, baseText)
	}

	return true, SendMessageWithInlineKeyboard(message.Chat.ID, baseText, buildVocabularyAddKeyboard(translationResult.TranslationID.String(), t))
}

func detectMessageTranslationLanguages(user *models.User, text string) (enums.Language, enums.Language, error) {
	sourceLanguage := user.Settings.TranslationSourceLanguage
	targetLanguage := user.Settings.TranslationTargetLanguage

	detectedLanguage, matchedSupportedLanguage, err := services.DetectLanguage(text)
	if err == nil && matchedSupportedLanguage {
		if detectedLanguage == sourceLanguage {
			return sourceLanguage, targetLanguage, nil
		}

		if detectedLanguage == targetLanguage {
			return targetLanguage, sourceLanguage, nil
		}
	}

	if containsCyrillic(text) {
		if sourceLanguage == enums.LanguageRu {
			return sourceLanguage, targetLanguage, nil
		}

		if targetLanguage == enums.LanguageRu {
			return targetLanguage, sourceLanguage, nil
		}
	}

	return sourceLanguage, targetLanguage, nil
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
