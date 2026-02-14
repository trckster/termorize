package services

import (
	"strings"
	"termorize/src/auth"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"
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
		PhotoUrl:   data.PhotoUrl,
		Settings:   defaultUserSettings(timezone),
	}

	err := db.DB.Create(&user).Error

	return &user, err
}

func updateUserByTelegramAuthData(user *models.User, data auth.TelegramAuthData) (*models.User, error) {
	user.Name = strings.TrimSpace(data.FirstName + " " + data.LastName)
	user.Username = data.Username
	user.PhotoUrl = data.PhotoUrl

	return user, db.DB.Save(&user).Error
}

func UpdateUserSettings(userID uint, settings models.UserSettings) (*models.User, error) {
	var user models.User

	if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	user.Settings = settings

	if err := db.DB.Save(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
