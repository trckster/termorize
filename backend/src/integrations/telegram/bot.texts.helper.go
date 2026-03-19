package telegram

import (
	"termorize/src/enums"
	"termorize/src/services"
)

func getBotTextsForTelegramID(telegramID int64) BotTexts {
	user, err := services.GetUserByTelegramID(telegramID)
	if err != nil || user == nil {
		return GetBotTexts(enums.LanguageEn)
	}
	return GetBotTexts(user.Settings.SystemLanguage)
}
