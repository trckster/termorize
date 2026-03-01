package telegram

import (
	"fmt"
	"termorize/src/config"
	"termorize/src/enums"
)

const (
	telegramTextHelp = "This bot will help you memorize a whole bunch of words.\n" +
		"Send /menu to see options!"
	telegramTextMenu = "📌 *Menu* 📌"

	telegramTextPong           = "pong"
	telegramTextUnknownCommand = "Unknown command! /help"

	telegramTextCancelNothing = "Nothing to cancel!"
	telegramTextCancelDone    = "Current action cancelled 👌"

	telegramTextNonPrivateChat = "Nah... Don't feel like answering here rn"

	telegramTextExerciseOutdated                       = "This exercise is outdated 🕰️"
	telegramTextExerciseCompleted                      = "This exercise is already successfully completed 🗸"
	telegramTextExerciseFailed                         = "This exercise was already attempted and failed 😔"
	telegramTextExerciseSuccess                        = "That's right! ✅"
	telegramTextExerciseInvalid                        = "Not quite... ❌"
	telegramTextExerciseIDK                            = "Alright, answer is:"
	telegramTextExerciseAnswerPairFormat               = "%s %s — %s %s"
	telegramTextExerciseTranslationKnowledgeUpFormat   = "Translation knowledge: *%d%%* 📈"
	telegramTextExerciseTranslationKnowledgeDownFormat = "Translation knowledge: *%d%%* 📉"

	telegramTextQuestionTranslateFormat = "Translate word *%s* to %s"

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

	telegramTextVocabularyAutoAddedSuffix   = "\n\nIt was added to vocabulary"
	telegramTextVocabularyManualAddedSuffix = "\n\nSuccessfully added to vocabulary"
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
	telegramButtonVocabularyAdd      = "Add to vocabulary"
	telegramButtonVocabularyDelete   = "Delete from vocabulary"
)

func BuildBasicExerciseQuestion(
	originalWord string,
	translationWord string,
	originalLanguage enums.Language,
	translationLanguage enums.Language,
	exerciseType enums.ExerciseType,
) string {
	if exerciseType == enums.ExerciseTypeBasicReversed {
		return buildTranslateQuestionText(translationWord, originalLanguage.DisplayNameWithFlag())
	}

	return buildTranslateQuestionText(originalWord, translationLanguage.DisplayNameWithFlag())
}

func buildTranslateQuestionText(word string, language string) string {
	return fmt.Sprintf(telegramTextQuestionTranslateFormat, word, language)
}

func buildAddVocabularyFirstText(nativeLanguage string, mainLearningLanguage string) string {
	return fmt.Sprintf(telegramTextAddVocabularyFirstFormat, nativeLanguage, mainLearningLanguage, config.GetPublicURL())
}
