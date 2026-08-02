package telegram

import (
	"termorize/src/config"
	"termorize/src/integrations/openrouter"
	"termorize/src/logger"
	"termorize/src/services"
)

func handlePronunciationCallback(callback *callbackQuery, payload []string) {
	if callback.Message == nil || len(payload) != 1 {
		return
	}

	translationID, err := parseCallbackUUID(payload[0])
	if err != nil {
		return
	}

	targetWord, err := services.GetTranslationTargetWord(translationID)
	if err != nil {
		logPronunciationFailure("failed to resolve pronunciation translation", err, translationID.String())
		return
	}

	model := config.GetOpenRouterTTSModel()
	voice := config.GetOpenRouterTTSVoice()
	pronunciation, err := services.FindWordPronunciationMetadata(targetWord.ID, model, voice)
	if err != nil {
		logPronunciationFailure("failed to load pronunciation cache", err, translationID.String())
		return
	}

	if pronunciation != nil && pronunciation.TelegramFileID != nil {
		if _, err := SendAudioByFileID(callback.Message.Chat.ID, *pronunciation.TelegramFileID, targetWord.Word); err == nil {
			removePronunciationButtonAfterSuccess(callback)
			return
		} else {
			logger.L().Warnw("cached telegram pronunciation audio file id was rejected", "error", err, "pronunciation_id", pronunciation.ID)
		}
	}

	if pronunciation == nil {
		audio, err := openrouter.NewSpeechClient().GenerateSpeech(targetWord.Word)
		if err != nil {
			logPronunciationFailure("failed to generate pronunciation", err, translationID.String())
			return
		}

		pronunciation, err = services.StoreWordPronunciation(targetWord.ID, model, voice, audio)
		if err != nil {
			logPronunciationFailure("failed to store pronunciation", err, translationID.String())
			return
		}
	} else {
		pronunciation.Audio, pronunciation.MIMEType, err = services.GetWordPronunciationAudio(pronunciation.ID)
		if err != nil {
			logPronunciationFailure("failed to load pronunciation audio", err, translationID.String())
			return
		}
	}

	telegramFileID, err := SendAudioMP3(callback.Message.Chat.ID, pronunciation.Audio, pronunciation.MIMEType, targetWord.Word)
	if err != nil {
		logPronunciationFailure("failed to upload pronunciation to telegram", err, translationID.String())
		return
	}

	if err := services.SetWordPronunciationTelegramFileID(pronunciation.ID, telegramFileID); err != nil {
		logPronunciationFailure("failed to cache telegram pronunciation file id", err, translationID.String())
	}

	removePronunciationButtonAfterSuccess(callback)
}

func logPronunciationFailure(message string, err error, translationID string) {
	logger.L().Warnw(message, "error", err, "translation_id", translationID)
}

func removePronunciationButtonAfterSuccess(callback *callbackQuery) {
	keyboard, removed := withoutPronunciationButtons(callback.Message.ReplyMarkup)
	if !removed {
		return
	}

	if err := editMessageInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, keyboard); err != nil {
		logger.L().Warnw(
			"failed to remove delivered pronunciation button",
			"error", err,
			"chat_id", callback.Message.Chat.ID,
			"message_id", callback.Message.MessageID,
		)
	}
}

func withoutPronunciationButtons(markup *inlineKeyboardMarkup) ([][]inlineKeyboardButton, bool) {
	if markup == nil {
		return nil, false
	}

	keyboard := make([][]inlineKeyboardButton, 0, len(markup.InlineKeyboard))
	removed := false
	for _, row := range markup.InlineKeyboard {
		filteredRow := make([]inlineKeyboardButton, 0, len(row))
		for _, button := range row {
			handlerType, _, ok := parseCallbackData(button.CallbackData)
			if ok && handlerType == callbackTypePronunciation {
				removed = true
				continue
			}
			filteredRow = append(filteredRow, button)
		}
		if len(filteredRow) > 0 {
			keyboard = append(keyboard, filteredRow)
		}
	}

	return keyboard, removed
}
