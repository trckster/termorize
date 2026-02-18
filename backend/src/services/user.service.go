package services

import (
	"errors"
	"strings"
	"termorize/src/auth"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"gorm.io/gorm"
)

func defaultUserSettings(timezone string) models.UserSettings {
	if _, err := time.LoadLocation(timezone); err != nil {
		timezone = "UTC"
	}

	return models.UserSettings{
		NativeLanguage:       enums.LanguageRu,
		MainLearningLanguage: enums.LanguageEn,
		TimeZone:             timezone,
		Telegram: models.UserTelegramSettings{
			BotEnabled:             false,
			DailyQuestionsEnabled:  false,
			DailyQuestionsCount:    0,
			DailyQuestionsSchedule: []models.UserTelegramQuestionsScheduleItem{},
		},
	}
}

func CreateOrUpdateUserByTelegramAuthData(data auth.TelegramAuthData, timezone string) (*models.User, error) {
	var user models.User
	result := db.DB.Where("telegram_id = ?", data.ID).First(&user)

	if result.Error == nil {
		return updateUserByTelegramAuthData(&user, data)
	}

	return createUserByTelegramAuthData(data, timezone)
}

func createUserByTelegramAuthData(data auth.TelegramAuthData, timezone string) (*models.User, error) {
	user := models.User{
		TelegramID: data.ID,
		Username:   data.Username,
		Name:       strings.TrimSpace(data.FirstName + " " + data.LastName),
		Settings:   defaultUserSettings(timezone),
	}

	err := db.DB.Create(&user).Error

	return &user, err
}

func updateUserByTelegramAuthData(user *models.User, data auth.TelegramAuthData) (*models.User, error) {
	user.Name = strings.TrimSpace(data.FirstName + " " + data.LastName)
	user.Username = data.Username

	return user, db.DB.Save(&user).Error
}

func UpdateUserSettings(userID uint, settings models.UserSettings) (*models.User, error) {
	var user models.User

	if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	// Edit of this flag is acceptable only via interaction with bot.
	// - Enable bot by sending any message
	// - Disable bot by blocking it
	settings.Telegram.BotEnabled = user.Settings.Telegram.BotEnabled

	user.Settings = settings

	if err := db.DB.Save(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func UpdateUserTelegramBotEnabled(telegramID int64, botEnabled bool) error {
	var user models.User

	if err := db.DB.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}

		return err
	}

	if user.Settings.Telegram.BotEnabled == botEnabled {
		return nil
	}

	settings := user.Settings
	settings.Telegram.BotEnabled = botEnabled

	return db.DB.Model(&user).Update("settings", settings).Error
}

func EnsureUserByTelegramID(telegramID int64, username string, firstName string, lastName string) error {
	var user models.User

	err := db.DB.Where("telegram_id = ?", telegramID).First(&user).Error
	if err == nil {
		return nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	settings := defaultUserSettings("")
	settings.Telegram.BotEnabled = true

	user = models.User{
		TelegramID: telegramID,
		Username:   username,
		Name:       strings.TrimSpace(firstName + " " + lastName),
		Settings:   settings,
	}

	return db.DB.Create(&user).Error
}
