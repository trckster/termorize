package telegram

import (
	"fmt"
	"math/rand"
	"termorize/src/enums"
)

const telegramMiniAppURL = "https://t.me/termorize_bot/app"

type BotTexts struct {
	Start          string
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
	ExerciseVocabularyDeleted              string
	ExerciseSuccess                        string
	ExerciseAlmost                         string
	ExerciseInvalid                        string
	ExerciseIDK                            string
	ExerciseAnswerPairFormat               string
	ExerciseTranslationKnowledgeUpFormat   string
	ExerciseTranslationKnowledgeDownFormat string

	QuestionTranslateFormat string

	MenuDeleteWord                   string
	MenuVocabularyEmpty              string
	MenuVocabularyLatestFormat       string
	MenuVocabularyMoreFormat         string
	MenuStatistics                   string
	MenuStatisticsTitle              string
	MenuStatisticsVocabulary         string
	MenuStatisticsExercises          string
	MenuStatisticsTotalFormat        string
	MenuStatisticsMasteredFormat     string
	MenuStatisticsInProgressFormat   string
	MenuStatisticsPendingFormat      string
	MenuStatisticsSuccessfulFormat   string
	MenuStatisticsUnsuccessfulFormat string
	MenuSettingsTitle                string
	MenuSettingsSystemLanguage       string
	MenuSettingsDailyExercises       string
	MenuSettingsEnabled              string
	MenuSettingsDisabled             string
	MenuSettingsFullVersionNote      string
	MenuWhatsGoingOn                 string

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

	ButtonOpenApp               string
	ButtonAddTranslation        string
	ButtonDeleteWord            string
	ButtonVocabulary            string
	ButtonStatistics            string
	ButtonSettings              string
	ButtonChangeSystemLanguage  string
	ButtonEnableDailyExercises  string
	ButtonDisableDailyExercises string
	ButtonWhatsGoingOn          string
	ButtonBack                  string
	ButtonCancel                string
	ButtonExerciseIDK           string
	ButtonVocabularyAdd         string
	ButtonVocabularyDelete      string

	ButtonChangeLanguagePrefix string

	ExerciseReminderPhrases []string
}

var botTextsEn = BotTexts{
	Start: "Welcome to *Termorize*.\n\n" +
		"Here you can:\n" +
		"- translate words and short phrases\n" +
		"- save custom translations to your vocabulary\n" +
		"- practice on the website\n" +
		"- receive automatic exercises in Telegram\n" +
		"- review statistics and adjust learning settings",
	Help:           "Use /menu to open the main menu.",
	Menu:           "📌 *Menu* 📌",
	Pong:           "pong",
	UnknownCommand: "Unknown command! Use /menu",

	CancelNothing: "Nothing to cancel!",
	CancelDone:    "Current action cancelled 👌",

	NonPrivateChat: "Nah... Don't feel like answering here rn",

	ExerciseOutdated:                       "This exercise is outdated 🕰️",
	ExerciseCompleted:                      "This exercise is already successfully completed 🗸",
	ExerciseFailed:                         "This exercise was already attempted and failed 😔",
	ExerciseVocabularyDeleted:              "This vocabulary was previously deleted 🗑️",
	ExerciseSuccess:                        "That's right! ✅",
	ExerciseAlmost:                         "Almost! The correct answer is:",
	ExerciseInvalid:                        "Not quite... ❌",
	ExerciseIDK:                            "Alright, answer is:",
	ExerciseAnswerPairFormat:               "%s %s — %s %s",
	ExerciseTranslationKnowledgeUpFormat:   "Translation knowledge: *%d%%* 📈",
	ExerciseTranslationKnowledgeDownFormat: "Translation knowledge: *%d%%* 📉",

	QuestionTranslateFormat: "Translate word *%s* to %s\n\n(answer with reply)",

	MenuDeleteWord:                   "Send the word you want to delete from vocabulary 🗑️",
	MenuVocabularyEmpty:              "Your vocabulary is empty for now. Add some translations!",
	MenuVocabularyLatestFormat:       "Latest translations (%d):",
	MenuVocabularyMoreFormat:         "And %d more translations are in your vocabulary.\nFull vocabulary is available on the website.",
	MenuStatistics:                   "📊 *Statistics*",
	MenuStatisticsTitle:              "📊 *Statistics*",
	MenuStatisticsVocabulary:         "📚 *Vocabulary*",
	MenuStatisticsExercises:          "📝 *Exercises*",
	MenuStatisticsTotalFormat:        "🧠 Total: *%d*",
	MenuStatisticsMasteredFormat:     "🏆 Mastered: *%d*",
	MenuStatisticsInProgressFormat:   "🌱 In Progress: *%d*",
	MenuStatisticsPendingFormat:      "⏳ Pending: *%d*",
	MenuStatisticsSuccessfulFormat:   "✅ Successful: *%d*",
	MenuStatisticsUnsuccessfulFormat: "❌ Unsuccessful: *%d*",
	MenuSettingsTitle:                "⚙️ *Settings*",
	MenuSettingsSystemLanguage:       "System Language",
	MenuSettingsDailyExercises:       "Daily Exercises",
	MenuSettingsEnabled:              "Enabled",
	MenuSettingsDisabled:             "Disabled",
	MenuSettingsFullVersionNote:      "All settings are available on the website!",
	MenuWhatsGoingOn: "*Termorize* is a vocabulary trainer that works in both Telegram and the web app.\n\n" +
		"You can use it to:\n" +
		"- translate words and short phrases\n" +
		"- add your own word pairs to vocabulary\n" +
		"- review saved vocabulary and learning progress\n" +
		"- practice with exercises on the website\n" +
		"- receive automatic Telegram exercises on your schedule\n" +
		"- change interface language and learning settings\n\n" +
		"Use *Open App* for the full experience, or stay in the bot for quick actions.",

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

	ButtonOpenApp:               "Open App 🌐",
	ButtonAddTranslation:        "Add Translation",
	ButtonDeleteWord:            "Delete Translation",
	ButtonVocabulary:            "Your Vocabulary",
	ButtonStatistics:            "Statistics",
	ButtonSettings:              "Settings",
	ButtonChangeSystemLanguage:  "Change System Language",
	ButtonEnableDailyExercises:  "Enable Daily Exercises",
	ButtonDisableDailyExercises: "Disable Daily Exercises",
	ButtonWhatsGoingOn:          "About",
	ButtonBack:                  "Back",
	ButtonCancel:                "Cancel",
	ButtonExerciseIDK:           "Don't know",
	ButtonVocabularyAdd:         "Add to vocabulary",
	ButtonVocabularyDelete:      "Delete from vocabulary",

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
		"I've had a call from heaven, they're missing their most diligent angel... Wanna finish the task?",
	},
}

var botTextsRu = BotTexts{
	Start: "Добро пожаловать в *Termorize*.\n\n" +
		"Здесь можно:\n" +
		"- переводить слова и короткие фразы\n" +
		"- сохранять свои переводы в словарь\n" +
		"- практиковаться на сайте\n" +
		"- получать автоматические упражнения в Telegram\n" +
		"- смотреть статистику и настраивать обучение",
	Help:           "Используй /menu, чтобы открыть главное меню.",
	Menu:           "📌 *Меню* 📌",
	Pong:           "pong",
	UnknownCommand: "Неизвестная команда! Используй /menu",

	CancelNothing: "Нечего отменять!",
	CancelDone:    "Текущее действие отменено 👌",

	NonPrivateChat: "Нет... Не хочу отвечать здесь",

	ExerciseOutdated:                       "Это упражнение устарело 🕰️",
	ExerciseCompleted:                      "Это упражнение уже успешно выполнено ✅",
	ExerciseFailed:                         "Это упражнение уже было выполнено с ошибкой 😔",
	ExerciseVocabularyDeleted:              "Это слово было когда-то удалено из словаря 🗑️",
	ExerciseSuccess:                        "Правильно! ✅",
	ExerciseAlmost:                         "Почти! Правильный ответ:",
	ExerciseInvalid:                        "Не совсем... ❌",
	ExerciseIDK:                            "Хорошо, ответ:",
	ExerciseAnswerPairFormat:               "%s %s — %s %s",
	ExerciseTranslationKnowledgeUpFormat:   "Знание перевода: *%d%%* 📈",
	ExerciseTranslationKnowledgeDownFormat: "Знание перевода: *%d%%* 📉",

	QuestionTranslateFormat: "Переведи слово *%s* на %s\n\n(ответь реплаем)",

	MenuDeleteWord:                   "Отправь слово, которое хочешь удалить из словаря 🗑️",
	MenuVocabularyEmpty:              "Твой словарь пока пуст. Добавь несколько переводов!",
	MenuVocabularyLatestFormat:       "Последние переводы (%d):",
	MenuVocabularyMoreFormat:         "И еще %d переводов есть в словаре.\nПолный список доступен на сайте.",
	MenuStatistics:                   "📊 *Статистика*",
	MenuStatisticsTitle:              "📊 *Статистика*",
	MenuStatisticsVocabulary:         "📚 *Словарь*",
	MenuStatisticsExercises:          "📝 *Упражнения*",
	MenuStatisticsTotalFormat:        "🧠 Всего: *%d*",
	MenuStatisticsMasteredFormat:     "🏆 Освоено: *%d*",
	MenuStatisticsInProgressFormat:   "🌱 В процессе: *%d*",
	MenuStatisticsPendingFormat:      "⏳ В ожидании: *%d*",
	MenuStatisticsSuccessfulFormat:   "✅ Успешно: *%d*",
	MenuStatisticsUnsuccessfulFormat: "❌ Неуспешно: *%d*",
	MenuSettingsTitle:                "⚙️ *Настройки*",
	MenuSettingsSystemLanguage:       "Язык Системы",
	MenuSettingsDailyExercises:       "Ежедневные Упражнения",
	MenuSettingsEnabled:              "Включены",
	MenuSettingsDisabled:             "Выключены",
	MenuSettingsFullVersionNote:      "Полная версия настроек доступна на сайте.",
	MenuWhatsGoingOn: "*Termorize* - это сервис для изучения слов, который работает и в Telegram, и в веб-приложении.\n\n" +
		"Здесь можно:\n" +
		"- переводить слова и короткие фразы\n" +
		"- добавлять свои пары слов в словарь\n" +
		"- смотреть сохраненный словарь и прогресс обучения\n" +
		"- практиковаться в упражнениях на сайте\n" +
		"- получать автоматические упражнения в Telegram по расписанию\n" +
		"- менять язык интерфейса и настройки обучения\n\n" +
		"Для полного функционала нажми *Открыть приложение*, а для быстрых действий можно остаться в боте.",

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

	ButtonOpenApp:               "Открыть приложение 🌐",
	ButtonAddTranslation:        "Добавить перевод",
	ButtonDeleteWord:            "Удалить перевод",
	ButtonVocabulary:            "Мой словарь",
	ButtonStatistics:            "Статистика",
	ButtonSettings:              "Настройки",
	ButtonChangeSystemLanguage:  "Изменить Язык Системы",
	ButtonEnableDailyExercises:  "Включить Ежедневные Упражнения",
	ButtonDisableDailyExercises: "Выключить Ежедневные Упражнения",
	ButtonWhatsGoingOn:          "О проекте",
	ButtonBack:                  "Назад",
	ButtonCancel:                "Отмена",
	ButtonExerciseIDK:           "Не знаю",
	ButtonVocabularyAdd:         "Добавить в словарь",
	ButtonVocabularyDelete:      "Удалить из словаря",

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
		"Звонили из рая, передали, что от них сбежал самый прилежный ангел... Закончишь задачку?",
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

func BuildSettingsText(systemLanguage enums.Language, dailyExercisesEnabled bool, texts BotTexts) string {
	dailyExercisesStatus := texts.MenuSettingsDisabled
	if dailyExercisesEnabled {
		dailyExercisesStatus = texts.MenuSettingsEnabled
	}

	return fmt.Sprintf(
		"%s\n\n%s: %s\n\n%s: %s\n\n%s",
		texts.MenuSettingsTitle,
		texts.MenuSettingsSystemLanguage,
		systemLanguage.DisplayNameWithFlag(),
		texts.MenuSettingsDailyExercises,
		dailyExercisesStatus,
		texts.MenuSettingsFullVersionNote,
	)
}

func BuildExerciseReminderText(texts BotTexts) string {
	if len(texts.ExerciseReminderPhrases) == 0 {
		return "Finish this exercise."
	}

	return texts.ExerciseReminderPhrases[rand.Intn(len(texts.ExerciseReminderPhrases))]
}
