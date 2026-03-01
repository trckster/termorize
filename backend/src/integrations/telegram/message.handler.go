package telegram

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

	if command, ok := parseMessageCommand(message.Text); ok {
		return routeMessageCommand(message, command)
	}

	return SendMessage(message.Chat.ID, message.Text)
}
