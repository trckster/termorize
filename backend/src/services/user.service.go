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
	if _, err := time.LoadLocation(timezone); err != nil || timezone == "" {
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

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("telegram_id = ?", data.ID).First(&user)

		if result.Error == nil {
			user.Name = strings.TrimSpace(data.FirstName + " " + data.LastName)
			user.Username = data.Username
			return tx.Save(&user).Error
		}

		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}

		user = models.User{
			TelegramID: data.ID,
			Username:   data.Username,
			Name:       strings.TrimSpace(data.FirstName + " " + data.LastName),
			Settings:   defaultUserSettings(timezone),
		}

		return tx.Create(&user).Error
	})
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func UpdateUserSettings(userID uint, settings models.UserSettings) (*models.User, error) {
	var user models.User

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}

		settings.Telegram.BotEnabled = user.Settings.Telegram.BotEnabled

		user.Settings = settings

		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func UpdateUserTelegramBotEnabled(telegramID int64, botEnabled bool) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User

		if err := tx.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
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

		return tx.Model(&user).Update("settings", settings).Error
	})
}

func UpdateUserTelegramState(telegramID int64, state enums.TelegramState) (bool, error) {
	updated := false

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User

		if err := tx.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}

			return err
		}

		if user.TelegramState == state {
			return nil
		}

		if err := tx.Model(&user).Update("telegram_state", state).Error; err != nil {
			return err
		}

		updated = true
		return nil
	})
	if err != nil {
		return false, err
	}

	return updated, nil
}

func GetUserByTelegramID(telegramID int64) (*models.User, error) {
	var user models.User

	if err := db.DB.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func EnsureUserByTelegramID(telegramID int64, username string, firstName string, lastName string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User

		err := tx.Where("telegram_id = ?", telegramID).First(&user).Error
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

		return tx.Create(&user).Error
	})
}

func GetUsersWithEnabledDailyQuestions() ([]models.User, error) {
	var users []models.User
	err := db.DB.
		Where("settings->'telegram'->'bot_enabled' = ?", true).
		Where("settings->'telegram'->'daily_questions_enabled' = ?", true).
		Find(&users).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return users, nil
		}
		return nil, err
	}

	return users, nil
}
