package telegram

import (
	"fmt"
	"math/rand"
	"termorize/src/enums"
)

const telegramMiniAppURL = "https://t.me/termorize_bot/app"

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

	telegramTextQuestionTranslateFormat = "Translate word *%s* to %s\n\n" +
		"(answer with reply)"

	telegramTextMenuDeleteWord   = "Send the word you want to delete from vocabulary 🗑️"
	telegramTextMenuVocabulary   = "⚒️ Work in progress here! ⚒️"
	telegramTextMenuStatistics   = "⚒️ Work in progress here! ⚒️"
	telegramTextMenuSettings     = "⚒️ Work in progress here! ⚒️"
	telegramTextMenuWhatsGoingOn = "Привет! 👋\n\n" +
		"Это новая версия бота. 🤖\n\n" +
		"Плохие новости: 😕\n" +
		"- история ответов на вопросы утеряна\n" +
		"- часть ранее доступного функционала сейчас не работает\n" +
		"- интерфейс на английском языке\n\n" +
		"Хорошие новости: ✨\n" +
		"- я планирую добавить возможность изменить язык интерфейса. Хотя бы на русский\n" +
		"- теперь у бота есть сайт. Пока что там пустовато, но в перспективе сайт позволит сделать то, что нельзя делать в боте\n" +
		"- все места, где сейчас заглушки, будут восстановлены 🚀\n\n" +
		"По любым вопросам пишите: @trckster"

	telegramTextAddVocabularyFirstFormat = "Send translation separated by colon (from *%s* to *%s*).\n\n" +
		"Example — *river:река*\n\n" +
		"To add translation in different languages, use the *Open App* button in Menu"
	telegramTextAddVocabularyDone          = "Translation added ✅"
	telegramTextAddVocabularyExists        = "Current translation already exists in vocabulary"
	telegramTextAddVocabularyInvalid       = "Invalid format. Send translation as word1:word2"
	telegramTextAddVocabularyTooManyColons = "Invalid format. Use only one colon to separate word and translation"

	telegramTextDeleteCompleted = "Done ✅"
	telegramTextDeleteNotFound  = "Word not found ❌"

	telegramTextVocabularyAutoAddedSuffix   = "\n\nIt was added to your vocabulary"
	telegramTextVocabularyManualAddedSuffix = "\n\nSuccessfully added to your vocabulary"
)

const (
	telegramButtonMenuOpenApp        = "Open App 🌐"
	telegramButtonMenuAddTranslation = "Add Translation"
	telegramButtonMenuDeleteWord     = "Delete Translation"
	telegramButtonMenuVocabulary     = "Your Vocabulary"
	telegramButtonMenuStatistics     = "Statistics"
	telegramButtonMenuSettings       = "Settings"
	telegramButtonMenuWhatsGoingOn   = "Что происходит?"
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

func buildAddVocabularyFirstText(systemLanguage string, mainLearningLanguage string) string {
	return fmt.Sprintf(telegramTextAddVocabularyFirstFormat, systemLanguage, mainLearningLanguage)
}

var telegramTextExerciseReminderPhrases = []string{
	"You are one answer away from finishing this one.",
	"Give this exercise a quick finish.",
	"Come back and close this exercise out.",
	"You have got this - finish the exercise.",
	"Take one more minute and finish this exercise.",
}

func BuildExerciseReminderText() string {
	if len(telegramTextExerciseReminderPhrases) == 0 {
		return "Finish this exercise."
	}

	return telegramTextExerciseReminderPhrases[rand.Intn(len(telegramTextExerciseReminderPhrases))]
}
