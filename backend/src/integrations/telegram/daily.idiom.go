package telegram

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/models"
	"termorize/src/services"
)

func SendDailyIdiom(user models.User, daily services.DailyIdiomResponse) error {
	title := "Idiom of the day"
	if user.Settings.SystemLanguage == enums.LanguageRu {
		title = "Идиома дня"
	}
	return SendMessageWithInlineKeyboard(user.TelegramID, title+"\n\n"+daily.Language.Flag()+" "+daily.Idiom.Word,
		buildIdiomKeyboard(daily.Idiom.ID, GetBotTexts(user.Settings.SystemLanguage), ""))
}

func handleIdiomCallback(callback *callbackQuery, payload []string) error {
	if callback == nil || callback.From == nil || callback.Message == nil || callback.Message.Chat.ID != callback.From.ID ||
		(len(payload) != 2 && len(payload) != 3) {
		return nil
	}
	action := payload[0]
	if (action != "add" && action != "translate" && action != "dismiss") || (len(payload) == 3 && action != "add") {
		return nil
	}
	id, err := uuid.Parse(payload[1])
	if err != nil {
		return nil
	}
	user, err := services.GetUserByTelegramID(callback.From.ID)
	if err != nil || user == nil {
		return err
	}
	t := GetBotTexts(user.Settings.SystemLanguage)
	if action == "dismiss" {
		return editMessageInlineKeyboardTolerant(callback.Message.Chat.ID, callback.Message.MessageID, [][]inlineKeyboardButton{})
	}
	// Resolve the actual selection's language (an older message may predate settings changes).
	daily, err := services.GetDailyIdiomSelection(id)
	if err != nil {
		return idiomFailure(callback, user, err, false)
	}
	target := services.IdiomTargetLanguage(user.Settings, daily.Language)
	// A preview pins the target so saving preserves the translation shown, even
	// if the user's language preferences change before they press Add.
	if len(payload) == 3 {
		target = enums.Language(payload[2])
		if !enums.IsSupportedLanguage(target) || target == daily.Language {
			return nil
		}
	}
	translated, err := services.TranslateDailyIdiom(context.Background(), id, target)
	if err != nil {
		return idiomFailure(callback, user, err, false)
	}
	if action == "translate" {
		text := buildVocabularyTranslationText(daily.Language, translated.SourceWord, translated.TranslatedWord, target)
		return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, text, buildIdiomKeyboard(id, t, target))
	}
	_, err = services.CreateVocabularyByTranslation(user.ID, translated.TranslationID)
	if err != nil && !errors.Is(err, services.ErrVocabularyAlreadyExists) {
		return idiomFailure(callback, user, err, true)
	}
	suffix := t.VocabularyManualAddedSuffix
	if errors.Is(err, services.ErrVocabularyAlreadyExists) {
		suffix = "\n\n" + t.AddVocabularyExists
	}
	text := buildVocabularyTranslationText(daily.Language, translated.SourceWord, translated.TranslatedWord, target) + suffix
	return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, text, [][]inlineKeyboardButton{})
}

func idiomFailure(callback *callbackQuery, user *models.User, err error, saving bool) error {
	logger.L().Warnw("failed to add daily idiom", "user_id", user.ID, "error", err)
	text := "Could not translate the idiom. Please try the button again."
	if saving {
		text = "Could not save the idiom. Please try the button again."
	}
	if user.Settings.SystemLanguage == enums.LanguageRu {
		text = "Не удалось перевести идиому. Нажмите кнопку ещё раз."
		if saving {
			text = "Не удалось сохранить идиому. Нажмите кнопку ещё раз."
		}
	}
	return SendMessage(callback.Message.Chat.ID, text)
}

// Keep each action on its own row so localized labels remain readable in Telegram.
func buildIdiomKeyboard(id uuid.UUID, t BotTexts, translatedTarget enums.Language) [][]inlineKeyboardButton {
	rows := [][]inlineKeyboardButton{}
	addData := "idiom:add:" + id.String()
	if translatedTarget == "" {
		rows = append(rows, []inlineKeyboardButton{{Text: t.ButtonIdiomTranslate, CallbackData: "idiom:translate:" + id.String()}})
	} else {
		addData += ":" + string(translatedTarget)
	}
	return append(rows,
		[]inlineKeyboardButton{{Text: t.ButtonVocabularyAdd, CallbackData: addData}},
		[]inlineKeyboardButton{{Text: t.ButtonIdiomDismiss, CallbackData: "idiom:dismiss:" + id.String()}},
	)
}
