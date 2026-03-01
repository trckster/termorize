package telegram

import (
	"strings"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/services"
)

var menuKeyboard = [][]inlineKeyboardButton{
	{{Text: telegramButtonMenuAddTranslation, CallbackData: "menu:add_translation"}, {Text: telegramButtonMenuDeleteWord, CallbackData: "menu:delete_translation"}},
	{{Text: telegramButtonMenuVocabulary, CallbackData: "menu:your_vocabulary"}, {Text: telegramButtonMenuStatistics, CallbackData: "menu:statistics"}},
	{{Text: telegramButtonMenuSettings, CallbackData: "menu:settings"}},
}

type messageCommandHandler func(message *message, args string) error

var messageCommandHandlers = map[string]messageCommandHandler{
	"ping": func(message *message, args string) error {
		return SendMessage(message.Chat.ID, telegramTextPong)
	},
	"help": func(message *message, args string) error {
		return SendMessage(message.Chat.ID, telegramTextHelp)
	},
	"start": func(message *message, args string) error {
		return SendMessage(message.Chat.ID, telegramTextHelp)
	},
	"menu": func(message *message, args string) error {
		return SendMessageWithInlineKeyboardMarkdown(message.Chat.ID, telegramTextMenu, menuKeyboard)
	},
	"cancel": func(message *message, args string) error {
		telegramID, _, _, _ := extractMessageUser(message)
		updated, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone)
		if err != nil {
			return err
		}

		if !updated {
			return SendMessage(message.Chat.ID, telegramTextCancelNothing)
		}

		return SendMessage(message.Chat.ID, telegramTextCancelDone)
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
		return SendMessage(message.Chat.ID, telegramTextNonPrivateChat)
	}

	handledExerciseAnswer, err := handleExerciseAnswer(message)
	if err != nil {
		return err
	}

	if handledExerciseAnswer {
		return nil
	}

	handledStateMessage, err := handleStateMessage(message)
	if err != nil {
		return err
	}

	if handledStateMessage {
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

func handleStateMessage(message *message) (bool, error) {
	if strings.HasPrefix(strings.TrimSpace(message.Text), "/") {
		return false, nil
	}

	if strings.EqualFold(strings.TrimSpace(message.Text), telegramButtonMenuCancel) {
		telegramID, _, _, _ := extractMessageUser(message)
		if _, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone); err != nil {
			return true, err
		}

		return true, SendMessageWithInlineKeyboardMarkdown(message.Chat.ID, telegramTextMenu, menuKeyboard)
	}

	telegramID, _, _, _ := extractMessageUser(message)
	user, err := services.GetUserByTelegramID(telegramID)
	if err != nil {
		return false, err
	}

	if user == nil {
		return false, nil
	}

	if user.TelegramState == enums.TelegramStateAddingVocabulary {
		parts := strings.SplitN(message.Text, ":", 2)
		if len(parts) != 2 {
			return true, SendMessage(message.Chat.ID, telegramTextAddVocabularyInvalid)
		}

		nativeWord := strings.TrimSpace(parts[0])
		learningWord := strings.TrimSpace(parts[1])
		if nativeWord == "" || learningWord == "" {
			return true, SendMessage(message.Chat.ID, telegramTextAddVocabularyInvalid)
		}

		_, err := services.CreateVocabulary(user.ID, services.CreateVocabularyRequest{
			Original:            learningWord,
			Translation:         nativeWord,
			OriginalLanguage:    user.Settings.MainLearningLanguage,
			TranslationLanguage: user.Settings.NativeLanguage,
		})
		if err != nil {
			if services.IsTranslationAlreadyExistsError(err) {
				return true, SendMessage(message.Chat.ID, telegramTextAddVocabularyExists)
			}

			return true, err
		}

		if _, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone); err != nil {
			return true, err
		}

		return true, SendMessage(message.Chat.ID, telegramTextAddVocabularyDone)
	}

	if user.TelegramState != enums.TelegramStateDeletingVocabulary {
		return false, nil
	}

	deleted, err := services.DeleteVocabularyByWord(user.ID, message.Text)
	if err != nil {
		return true, err
	}

	if !deleted {
		return true, SendMessage(message.Chat.ID, telegramTextDeleteNotFound)
	}

	if _, err := services.UpdateUserTelegramState(telegramID, enums.TelegramStateNone); err != nil {
		return true, err
	}

	return true, SendMessage(message.Chat.ID, telegramTextDeleteCompleted)
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
		return true, SendMessage(message.Chat.ID, telegramTextExerciseOutdated)
	case enums.ExerciseStatusCompleted:
		return true, SendMessage(message.Chat.ID, telegramTextExerciseCompleted)
	case enums.ExerciseStatusFailed:
		return true, SendMessage(message.Chat.ID, telegramTextExerciseFailed)
	case enums.ExerciseStatusPending:
		return true, nil
	case enums.ExerciseStatusInProgress:
	default:
		return true, nil
	}

	if err := removeMessageInlineKeyboard(message.Chat.ID, message.ReplyToMessage.MessageID); err != nil {
		logger.L().Warnw("failed to remove inline keyboard", "error", err, "chat_id", message.Chat.ID, "message_id", message.ReplyToMessage.MessageID)
	}

	isCorrect := isCorrectExerciseAnswer(message.Text, exercise.ExerciseType, exercise.OriginalWord, exercise.TranslationWord)
	if isCorrect {
		if err := services.CompleteExercise(exercise.ExerciseID); err != nil {
			return false, err
		}
		return true, SendMessage(message.Chat.ID, telegramTextExerciseSuccess)
	}

	updated, err := services.FailExercise(exercise.ExerciseID)
	if err != nil {
		return false, err
	}

	if !updated {
		return true, nil
	}

	answerText := buildIDKAnswer(exercise.OriginalWord, exercise.TranslationWord, exercise.ExerciseType)
	return true, SendMessageMarkdown(message.Chat.ID, answerText)
}

func isCorrectExerciseAnswer(answer string, exerciseType enums.ExerciseType, originalWord string, translationWord string) bool {
	normalizedAnswer := strings.TrimSpace(answer)

	if exerciseType == enums.ExerciseTypeBasicDirect {
		return strings.EqualFold(normalizedAnswer, strings.TrimSpace(translationWord))
	}

	if exerciseType == enums.ExerciseTypeBasicReversed {
		return strings.EqualFold(normalizedAnswer, strings.TrimSpace(originalWord))
	}

	return false
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
		return SendMessage(message.Chat.ID, "Unknown command! /help")
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
