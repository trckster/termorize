package telegram

func handleMessage(message *message) error {
	if message.Chat.Type == Private {
		active, err := ensurePrivateMessageUser(message)
		if err != nil {
			return err
		}
		if !active {
			return nil
		}
	}

	if message.Text == "" {
		return nil
	}

	if message.Chat.Type != Private {
		telegramID, _, _, _ := extractMessageUser(message)
		t := getBotTextsForTelegramID(telegramID)
		return SendMessage(message.Chat.ID, t.NonPrivateChat)
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

	if command, ok := parseMessageCommand(message.Text); ok {
		return routeMessageCommand(message, command)
	}

	handledTranslationMessage, err := handlePlainTranslationMessage(message)
	if err != nil {
		return err
	}

	if handledTranslationMessage {
		return nil
	}

	return SendMessage(message.Chat.ID, message.Text)
}
