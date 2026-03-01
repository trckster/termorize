package telegram

import (
	"fmt"
	"math/rand"
	"termorize/src/config"
)

const (
	questionTypeOriginalToTranslation = "o2t"
	questionTypeTranslationToOriginal = "t2o"

	telegramTextHelp = "This bot will help you memorize a whole bunch of words.\n" +
		"Send /menu to see options!"
	telegramTextMenu = "📌 *Menu* 📌"

	telegramTextPong = "pong"

	telegramTextCancelNothing = "Nothing to cancel!"
	telegramTextCancelDone    = "Current action cancelled 👌"

	telegramTextNonPrivateChat = "Nah... Don't feel like answering here rn"

	telegramTextExerciseOutdated  = "This exercise is outdated 🕰️"
	telegramTextExerciseCompleted = "This exercise is already successfully completed 🗸"
	telegramTextExerciseFailed    = "This exercise was already attempted and failed 😔"
	telegramTextExerciseSuccess   = "That's right! ✅"

	// TODO Ought to be reworked
	telegramTextQuestionTranslatePrefix = "Translate this word:"
	telegramTextQuestionOriginalPrefix  = "What is the original word for:"

	// TODO Ought to be reworked
	telegramTextQuestionTranslateFormat = "Translate this word: %s\n\n" +
		"(answer via reply to this message)"
	telegramTextQuestionOriginalFormat = "What is the original word for: %s\n\n" +
		"(answer via reply to this message)"

	telegramTextMenuDeleteWord = "Send the word you want to delete from vocabulary 🗑️"
	telegramTextMenuVocabulary = "⚒️ Work in progress here! ⚒️"
	telegramTextMenuStatistics = "⚒️ Work in progress here! ⚒️"
	telegramTextMenuSettings   = "⚒️ Work in progress here! ⚒️"

	telegramTextAddVocabularyFirstFormat = "Send translation separated by colon (from *%s* to *%s*).\n\n" +
		"Example — *река:river*\n\n" +
		"To add translation in different languages, proceed to the website: %s"
	telegramTextAddVocabularyDone    = "Translation added ✅"
	telegramTextAddVocabularyExists  = "Current translation already exists in vocabulary"
	telegramTextAddVocabularyInvalid = "Invalid format. Send translation as word1:word2"

	telegramTextDeleteCompleted = "Done ✅"
	telegramTextDeleteNotFound  = "Word not found ❌"

	telegramTextIDKOriginalPrefix    = "Correct original word: "
	telegramTextIDKTranslationPrefix = "Correct translation: "
)

const (
	telegramButtonMenuAddTranslation = "Add Translation"
	telegramButtonMenuDeleteWord     = "Delete Translation"
	telegramButtonMenuVocabulary     = "Your Vocabulary"
	telegramButtonMenuStatistics     = "Statistics"
	telegramButtonMenuSettings       = "Settings"
	telegramButtonMenuBack           = "Back"
	telegramButtonMenuCancel         = "Cancel"
	telegramButtonExerciseIDK        = "Don't know"
)

func BuildBasicExerciseQuestion(originalWord string, translationWord string) (string, string) {
	if rand.Intn(2) == 0 {
		return buildTranslateQuestionText(originalWord), questionTypeOriginalToTranslation
	}

	return buildOriginalQuestionText(translationWord), questionTypeTranslationToOriginal
}

func buildTranslateQuestionText(word string) string {
	return fmt.Sprintf(telegramTextQuestionTranslateFormat, word)
}

func buildOriginalQuestionText(word string) string {
	return fmt.Sprintf(telegramTextQuestionOriginalFormat, word)
}

func buildAddVocabularyFirstText(nativeLanguage string, mainLearningLanguage string) string {
	return fmt.Sprintf(telegramTextAddVocabularyFirstFormat, nativeLanguage, mainLearningLanguage, config.GetPublicURL())
}
