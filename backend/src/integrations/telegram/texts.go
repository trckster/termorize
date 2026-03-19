package telegram

import (
	"fmt"
	"math/rand"
	"termorize/src/enums"
)

const telegramMiniAppURL = "https://t.me/termorize_bot/app"

type BotTexts struct {
	Help           string
	Menu           string
	Pong           string
	UnknownCommand string

	CancelNothing string
	CancelDone    string

	NonPrivateChat string

	ExerciseOutdated                       string
	ExerciseCompleted                      string
	ExerciseFailed                         string
	ExerciseSuccess                        string
	ExerciseInvalid                        string
	ExerciseIDK                            string
	ExerciseAnswerPairFormat               string
	ExerciseTranslationKnowledgeUpFormat   string
	ExerciseTranslationKnowledgeDownFormat string

	QuestionTranslateFormat string

	MenuDeleteWord   string
	MenuVocabulary   string
	MenuStatistics   string
	MenuSettings     string
	MenuWhatsGoingOn string

	ChooseLanguage string

	AddVocabularyFirstFormat   string
	AddVocabularyDone          string
	AddVocabularyExists        string
	AddVocabularyInvalid       string
	AddVocabularyTooManyColons string

	DeleteCompleted string
	DeleteNotFound  string

	VocabularyAutoAddedSuffix   string
	VocabularyManualAddedSuffix string

	ButtonOpenApp          string
	ButtonAddTranslation   string
	ButtonDeleteWord       string
	ButtonVocabulary       string
	ButtonStatistics       string
	ButtonSettings         string
	ButtonWhatsGoingOn     string
	ButtonBack             string
	ButtonCancel           string
	ButtonExerciseIDK      string
	ButtonVocabularyAdd    string
	ButtonVocabularyDelete string

	ButtonChangeLanguagePrefix string

	ExerciseReminderPhrases []string
}

var botTextsEn = BotTexts{
	Help:           "This bot will help you memorize a whole bunch of words.\nSend /menu to see options!",
	Menu:           "📌 *Menu* 📌",
	Pong:           "pong",
	UnknownCommand: "Unknown command! /help",

	CancelNothing: "Nothing to cancel!",
	CancelDone:    "Current action cancelled 👌",

	NonPrivateChat: "Nah... Don't feel like answering here rn",

	ExerciseOutdated:                       "This exercise is outdated 🕰️",
	ExerciseCompleted:                      "This exercise is already successfully completed 🗸",
	ExerciseFailed:                         "This exercise was already attempted and failed 😔",
	ExerciseSuccess:                        "That's right! ✅",
	ExerciseInvalid:                        "Not quite... ❌",
	ExerciseIDK:                            "Alright, answer is:",
	ExerciseAnswerPairFormat:               "%s %s — %s %s",
	ExerciseTranslationKnowledgeUpFormat:   "Translation knowledge: *%d%%* 📈",
	ExerciseTranslationKnowledgeDownFormat: "Translation knowledge: *%d%%* 📉",

	QuestionTranslateFormat: "Translate word *%s* to %s\n\n(answer with reply)",

	MenuDeleteWord: "Send the word you want to delete from vocabulary 🗑️",
	MenuVocabulary: "⚒️ Work in progress here! ⚒️",
	MenuStatistics: "⚒️ Work in progress here! ⚒️",
	MenuSettings:   "⚒️ Work in progress here! ⚒️",
	MenuWhatsGoingOn: "Hello! 👋\n\n" +
		"This is the new version of the bot. 🤖\n\n" +
		"Bad news: 😕\n" +
		"- question answer history was lost\n" +
		"- some previously available functionality is currently not working\n\n" +
		"Good news: ✨\n" +
		"- the bot now has a website. It's a bit empty for now, but in the future the site will allow doing things that can't be done in the bot\n" +
		"- all the placeholder sections will be restored 🚀\n\n" +
		"For any questions write: @trckster",

	ChooseLanguage: "Choose language:",

	AddVocabularyFirstFormat:   "Send translation separated by colon (from *%s* to *%s*).\n\nExample — *river:река*",
	AddVocabularyDone:          "Translation added ✅",
	AddVocabularyExists:        "Current translation already exists in vocabulary",
	AddVocabularyInvalid:       "Invalid format. Send translation as word1:word2",
	AddVocabularyTooManyColons: "Invalid format. Use only one colon to separate word and translation",

	DeleteCompleted: "Done ✅",
	DeleteNotFound:  "Word not found ❌",

	VocabularyAutoAddedSuffix:   "\n\nIt was added to your vocabulary",
	VocabularyManualAddedSuffix: "\n\nSuccessfully added to your vocabulary",

	ButtonOpenApp:          "Open App 🌐",
	ButtonAddTranslation:   "Add Translation",
	ButtonDeleteWord:       "Delete Translation",
	ButtonVocabulary:       "Your Vocabulary",
	ButtonStatistics:       "Statistics",
	ButtonSettings:         "Settings",
	ButtonWhatsGoingOn:     "What's happening?",
	ButtonBack:             "Back",
	ButtonCancel:           "Cancel",
	ButtonExerciseIDK:      "Don't know",
	ButtonVocabularyAdd:    "Add to vocabulary",
	ButtonVocabularyDelete: "Delete from vocabulary",

	ButtonChangeLanguagePrefix: "Change ",

	ExerciseReminderPhrases: []string{
		"You are one answer away from finishing this one.",
		"Give this exercise a quick finish",
		"Come back and close this one thx",
		"Friendly reminder",
		"This one is almost entirely lost, save it",
		"If you don't know translation, there's button",
		"A little bit of your attention is required here. Thanks.",
		"You are going to finish this one, right?..\n\nRight?",
		"You forgot something!",
		"Don't ruin your stats, answer this!",
		"⌛",
	},
}

var botTextsRu = BotTexts{
	Help:           "Этот бот поможет тебе запомнить множество слов.\nОтправь /menu чтобы увидеть опции!",
	Menu:           "📌 *Меню* 📌",
	Pong:           "pong",
	UnknownCommand: "Неизвестная команда! /help",

	CancelNothing: "Нечего отменять!",
	CancelDone:    "Текущее действие отменено 👌",

	NonPrivateChat: "Нет... Не хочу отвечать здесь",

	ExerciseOutdated:                       "Это упражнение устарело 🕰️",
	ExerciseCompleted:                      "Это упражнение уже успешно выполнено 🗸",
	ExerciseFailed:                         "Это упражнение уже было выполнено с ошибкой 😔",
	ExerciseSuccess:                        "Правильно! ✅",
	ExerciseInvalid:                        "Не совсем... ❌",
	ExerciseIDK:                            "Хорошо, ответ:",
	ExerciseAnswerPairFormat:               "%s %s — %s %s",
	ExerciseTranslationKnowledgeUpFormat:   "Знание перевода: *%d%%* 📈",
	ExerciseTranslationKnowledgeDownFormat: "Знание перевода: *%d%%* 📉",

	QuestionTranslateFormat: "Переведи слово *%s* на %s\n\n(ответь реплаем)",

	MenuDeleteWord: "Отправь слово, которое хочешь удалить из словаря 🗑️",
	MenuVocabulary: "⚒️ В процессе разработки! ⚒️",
	MenuStatistics: "⚒️ В процессе разработки! ⚒️",
	MenuSettings:   "⚒️ В процессе разработки! ⚒️",
	MenuWhatsGoingOn: "Привет! 👋\n\n" +
		"Это новая версия бота. 🤖\n\n" +
		"Плохие новости: 😕\n" +
		"- история ответов на вопросы утеряна\n" +
		"- часть ранее доступного функционала сейчас не работает\n\n" +
		"Хорошие новости: ✨\n" +
		"- теперь у бота есть сайт. Пока что там пустовато, но в перспективе сайт позволит сделать то, что нельзя делать в боте\n" +
		"- появилась поддержка русского и английского в интерфейсе (поменять можно в настройках на сайте)\n" +
		"- все места, где сейчас заглушки, будут восстановлены 🚀\n\n" +
		"По любым вопросам пишите: @trckster",

	ChooseLanguage: "Выбери язык:",

	AddVocabularyFirstFormat:   "Отправь перевод через двоеточие (с *%s* на *%s*).\n\nПример — *river:река*",
	AddVocabularyDone:          "Перевод добавлен ✅",
	AddVocabularyExists:        "Такой перевод уже есть в словаре",
	AddVocabularyInvalid:       "Неверный формат. Отправь перевод как слово1:слово2",
	AddVocabularyTooManyColons: "Неверный формат. Используй только одно двоеточие для разделения слова и перевода",

	DeleteCompleted: "Готово ✅",
	DeleteNotFound:  "Слово не найдено ❌",

	VocabularyAutoAddedSuffix:   "\n\nДобавлено в твой словарь",
	VocabularyManualAddedSuffix: "\n\nУспешно добавлено в словарь",

	ButtonOpenApp:          "Открыть приложение 🌐",
	ButtonAddTranslation:   "Добавить перевод",
	ButtonDeleteWord:       "Удалить перевод",
	ButtonVocabulary:       "Мой словарь",
	ButtonStatistics:       "Статистика",
	ButtonSettings:         "Настройки",
	ButtonWhatsGoingOn:     "Что происходит?",
	ButtonBack:             "Назад",
	ButtonCancel:           "Отмена",
	ButtonExerciseIDK:      "Не знаю",
	ButtonVocabularyAdd:    "Добавить в словарь",
	ButtonVocabularyDelete: "Удалить из словаря",

	ButtonChangeLanguagePrefix: "Изменить ",

	ExerciseReminderPhrases: []string{
		"Давай давай ДАВАЙ давай давай",
		"Одно упражнение потерялось",
		"У тебя получится — заверши, пожалуйста",
		"Тут ещё один вопросик остался",
		"Ты же помнишь перевод?",
		"Вот тут надо доработать. Спасибо.",
		"Если не знаешь перевод, там есть кнопка.",
		"Если на это не ответить, оно истечёт :с",
		"Дружеское напоминание",
		"Я понимаю, у тебя были дела. Теперь надо ответить",
		"Не порти статистику, доделай упражнение",
		"⌛",
	},
}

func GetBotTexts(lang enums.Language) BotTexts {
	if lang == enums.LanguageRu {
		return botTextsRu
	}
	return botTextsEn
}

func BuildBasicExerciseQuestion(
	originalWord string,
	translationWord string,
	originalLanguage enums.Language,
	translationLanguage enums.Language,
	exerciseType enums.ExerciseType,
	texts BotTexts,
) string {
	if exerciseType == enums.ExerciseTypeBasicReversed {
		return buildTranslateQuestionText(translationWord, originalLanguage.DisplayNameWithFlag(), texts)
	}

	return buildTranslateQuestionText(originalWord, translationLanguage.DisplayNameWithFlag(), texts)
}

func buildTranslateQuestionText(word string, language string, texts BotTexts) string {
	return fmt.Sprintf(texts.QuestionTranslateFormat, word, language)
}

func buildAddVocabularyFirstText(systemLanguage string, mainLearningLanguage string, texts BotTexts) string {
	return fmt.Sprintf(texts.AddVocabularyFirstFormat, systemLanguage, mainLearningLanguage)
}

func BuildExerciseReminderText(texts BotTexts) string {
	if len(texts.ExerciseReminderPhrases) == 0 {
		return "Finish this exercise."
	}

	return texts.ExerciseReminderPhrases[rand.Intn(len(texts.ExerciseReminderPhrases))]
}
