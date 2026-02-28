package telegram

import (
	"strings"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/services"
)

const (
	translateQuestionPrefix = "Translate this word:"
	originalQuestionPrefix  = "What is the original word for:"

	helpText = "This bot will help you memorize whole bunch of words.\n" +
		"Send /menu to see options!"

	menuMessageText = "Choose an option:"
)

var menuKeyboard = [][]inlineKeyboardButton{
	{{Text: "Add Translation", CallbackData: "menu:add_translation"}, {Text: "Delete Translation", CallbackData: "menu:delete_translation"}},
	{{Text: "Your Vocabulary", CallbackData: "menu:your_vocabulary"}, {Text: "Statistics", CallbackData: "menu:statistics"}},
	{{Text: "Settings", CallbackData: "menu:settings"}},
}

type messageCommandHandler func(message *message, args string) error

var messageCommandHandlers = map[string]messageCommandHandler{
	"ping": func(message *message, args string) error {
		return SendMessage(message.Chat.ID, "pong")
	},
	"help": func(message *message, args string) error {
		return SendMessage(message.Chat.ID, helpText)
	},
	"menu": func(message *message, args string) error {
		return SendMessageWithInlineKeyboard(message.Chat.ID, menuMessageText, menuKeyboard)
	},
}

func handleMessage(message *message) error {
	if message.Chat.Type == Private {
		if err := ensurePrivateMessageUser(message); err != nil {
			return err
		}
	}

	if message.Text == "" {
		return nil
	}

	if message.Chat.Type != Private {
		return SendMessage(message.Chat.ID, "Nah... Don't feel like answering here rn")
	}

	handledExerciseAnswer, err := handleExerciseAnswer(message)
	if err != nil {
		return err
	}

	if handledExerciseAnswer {
		return nil
	}

	if command, args, ok := parseMessageCommand(message.Text); ok {
		if err := routeMessageCommand(message, command, args); err != nil {
			return err
		}
		return nil
	}

	return SendMessage(message.Chat.ID, message.Text)
}

func handleExerciseAnswer(message *message) (bool, error) {
	if message.ReplyToMessage == nil {
		return false, nil
	}

	telegramID, _, _, _ := extractMessageUser(message)
	exercise, err := services.GetExerciseByTelegramMessage(message.ReplyToMessage.MessageID, telegramID)
	if err != nil {
		return false, err
	}

	if exercise == nil {
		return false, nil
	}

	switch exercise.Status {
	case enums.ExerciseStatusIgnored:
		return true, SendMessage(message.Chat.ID, "This exercise is outdated.")
	case enums.ExerciseStatusCompleted:
		return true, SendMessage(message.Chat.ID, "This exercise is already successfully completed!")
	case enums.ExerciseStatusFailed:
		return true, SendMessage(message.Chat.ID, "This exercise was already attempted and failed!")
	case enums.ExerciseStatusPending:
		return true, nil
	case enums.ExerciseStatusInProgress:
	default:
		return true, nil
	}

	if err := removeMessageInlineKeyboard(message.Chat.ID, message.ReplyToMessage.MessageID); err != nil {
		logger.L().Warnw("failed to remove inline keyboard", "error", err, "chat_id", message.Chat.ID, "message_id", message.ReplyToMessage.MessageID)
	}

	questionType, ok := detectQuestionType(message.ReplyToMessage.Text)
	if !ok {
		return true, nil
	}

	isCorrect := isCorrectExerciseAnswer(message.Text, questionType, exercise.OriginalWord, exercise.TranslationWord)
	if isCorrect {
		if err := services.CompleteExercise(exercise.ExerciseID); err != nil {
			return false, err
		}
		return true, SendMessage(message.Chat.ID, "Success")
	}

	updated, err := services.FailExercise(exercise.ExerciseID)
	if err != nil {
		return false, err
	}

	if !updated {
		return true, nil
	}

	answerText := buildIDKAnswer(exercise.OriginalWord, exercise.TranslationWord, questionType)
	return true, SendMessage(message.Chat.ID, answerText)
}

func isCorrectExerciseAnswer(answer string, questionType string, originalWord string, translationWord string) bool {
	normalizedAnswer := strings.TrimSpace(answer)

	if questionType == "o2t" {
		return strings.EqualFold(normalizedAnswer, strings.TrimSpace(translationWord))
	}

	if questionType == "t2o" {
		return strings.EqualFold(normalizedAnswer, strings.TrimSpace(originalWord))
	}

	return false
}

func detectQuestionType(questionText string) (string, bool) {
	normalizedQuestion := strings.TrimSpace(questionText)

	if strings.HasPrefix(normalizedQuestion, translateQuestionPrefix) {
		return "o2t", true
	}

	if strings.HasPrefix(normalizedQuestion, originalQuestionPrefix) {
		return "t2o", true
	}

	return "", false
}

func ensurePrivateMessageUser(message *message) error {
	telegramID, username, firstName, lastName := extractMessageUser(message)

	if err := services.EnsureUserByTelegramID(telegramID, username, firstName, lastName); err != nil {
		logger.L().Warnw("failed to ensure telegram user", "error", err, "telegram_id", telegramID)
		return err
	}

	if err := services.UpdateUserTelegramBotEnabled(telegramID, true); err != nil {
		logger.L().Warnw("failed to enable telegram bot for user", "error", err, "telegram_id", telegramID)
		return err
	}

	return nil
}

func parseMessageCommand(text string) (string, string, bool) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") {
		return "", "", false
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", "", false
	}

	command := strings.TrimPrefix(parts[0], "/")
	if command == "" {
		return "", "", false
	}

	if index := strings.Index(command, "@"); index >= 0 {
		command = command[:index]
	}

	if command == "" {
		return "", "", false
	}

	arguments := ""
	if len(parts) > 1 {
		arguments = strings.Join(parts[1:], " ")
	}

	return strings.ToLower(command), arguments, true
}

func routeMessageCommand(message *message, command string, args string) error {
	handler, exists := messageCommandHandlers[command]
	if !exists {
		return nil
	}

	return handler(message, args)
}

func extractMessageUser(message *message) (int64, string, string, string) {
	telegramID := message.Chat.ID
	username := message.Chat.Username
	firstName := message.Chat.FirstName
	lastName := ""

	if message.From != nil {
		telegramID = message.From.ID
		username = message.From.Username
		firstName = message.From.FirstName
		lastName = message.From.LastName
	}

	return telegramID, username, firstName, lastName
}
