package telegram

import (
	"fmt"
	"math/rand"
)

const (
	questionTypeOriginalToTranslation = "o2t"
	questionTypeTranslationToOriginal = "t2o"

	telegramTextHelp = "This bot will help you memorize whole bunch of words.\n" +
		"Send /menu to see options!"
	telegramTextMenu = "Choose an option:"

	telegramTextPong = "pong"

	telegramTextCancelNothing = "Nothing to cancel!"
	telegramTextCancelDone    = "Current action cancelled."

	telegramTextNonPrivateChat = "Nah... Don't feel like answering here rn"

	telegramTextExerciseOutdated  = "This exercise is outdated."
	telegramTextExerciseCompleted = "This exercise is already successfully completed!"
	telegramTextExerciseFailed    = "This exercise was already attempted and failed!"
	telegramTextExerciseSuccess   = "Success"

	telegramTextQuestionTranslatePrefix = "Translate this word:"
	telegramTextQuestionOriginalPrefix  = "What is the original word for:"

	telegramTextQuestionTranslateFormat = "Translate this word: %s"
	telegramTextQuestionOriginalFormat  = "What is the original word for: %s"

	telegramTextMenuAddTranslation = "Work in progress here!"
	telegramTextMenuDeleteWord     = "Which word do you want to delete from vocabulary? 🗑️"
	telegramTextMenuVocabulary     = "Work in progress here!"
	telegramTextMenuStatistics     = "Work in progress here!"
	telegramTextMenuSettings       = "Work in progress here!"

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
	telegramButtonExerciseIDK        = "IDK"
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
