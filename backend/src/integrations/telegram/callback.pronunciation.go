package telegram

import (
	"termorize/src/logger"
	"termorize/src/services"
)

func handlePronunciationCallback(callback *callbackQuery, payload []string) {
	if callback.Message == nil || (len(payload) != 1 && len(payload) != 2) {
		return
	}

	// Callbacks from older messages have no side and pronounce the translation.
	side := pronunciationSideTarget
	if len(payload) == 2 {
		side = payload[1]
		if side != pronunciationSideSource && side != pronunciationSideTarget {
			return
		}
	}

	translationID, err := parseCallbackUUID(payload[0])
	if err != nil {
		return
	}

	sourceWord, targetWord, err := services.GetTranslationWords(translationID)
	if err != nil {
		logPronunciationFailure("failed to resolve pronunciation translation", err, translationID.String())
		return
	}

	word := targetWord
	if side == pronunciationSideSource {
		word = sourceWord
	}

	pronunciation, err := services.FindConfiguredWordPronunciationMetadata(word.ID, string(word.Language))
	if err != nil {
		logPronunciationFailure("failed to load pronunciation cache", err, translationID.String())
		return
	}

	if pronunciation != nil && pronunciation.TelegramFileID != nil {
		if _, err := SendAudioByFileID(callback.Message.Chat.ID, *pronunciation.TelegramFileID, word.Word); err == nil {
			return
		} else {
			logger.L().Warnw("cached telegram pronunciation audio file id was rejected", "error", err, "pronunciation_id", pronunciation.ID)
		}
	}

	if pronunciation == nil {
		pronunciation, err = services.GetOrCreateWordPronunciation(word.ID)
		if err != nil {
			logPronunciationFailure("failed to generate pronunciation", err, translationID.String())
			return
		}
	} else {
		pronunciation.Audio, pronunciation.MIMEType, err = services.GetWordPronunciationAudio(pronunciation.ID)
		if err != nil {
			logPronunciationFailure("failed to load pronunciation audio", err, translationID.String())
			return
		}
	}

	telegramFileID, err := SendAudioMP3(callback.Message.Chat.ID, pronunciation.Audio, pronunciation.MIMEType, word.Word)
	if err != nil {
		logPronunciationFailure("failed to upload pronunciation to telegram", err, translationID.String())
		return
	}

	if err := services.SetWordPronunciationTelegramFileID(pronunciation.ID, telegramFileID); err != nil {
		logPronunciationFailure("failed to cache telegram pronunciation file id", err, translationID.String())
	}
}

func logPronunciationFailure(message string, err error, translationID string) {
	logger.L().Warnw(message, "error", err, "translation_id", translationID)
}
