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
	"gorm.io/gorm/clause"
)

func defaultUserSettings(timezone string, botEnabled bool) models.UserSettings {
	if _, err := time.LoadLocation(timezone); err != nil || timezone == "" {
		timezone = "UTC"
	}

	return models.UserSettings{
		SystemLanguage:              enums.LanguageRu,
		MainLearningLanguage:        enums.LanguageEn,
		TranslationSourceLanguage:   enums.LanguageEn,
		TranslationTargetLanguage:   enums.LanguageRu,
		IgnoredAudioLanguages:       []enums.Language{},
		IgnoredDescriptionLanguages: []enums.Language{},
		TimeZone:                    timezone,
		Telegram: models.UserTelegramSettings{
			BotEnabled:             botEnabled,
			DailyQuestionsEnabled:  true,
			DailyQuestionsCount:    2,
			DailyQuestionsSchedule: []models.UserTelegramQuestionsScheduleItem{{From: "10:00", To: "22:00"}},
		},
	}
}

func CreateOrUpdateUserByTelegramProfile(profile auth.TelegramUserProfile, timezone string) (*models.User, error) {
	var user models.User

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("telegram_id = ?", profile.ID).First(&user)

		if result.Error == nil {
			if user.DeletedAt.Valid {
				return ErrUserDeleted
			}

			user.Name = strings.TrimSpace(profile.Name)
			user.Username = profile.Username
			return tx.Save(&user).Error
		}

		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}

		user = models.User{
			TelegramID: profile.ID,
			Username:   profile.Username,
			Name:       strings.TrimSpace(profile.Name),
			Settings:   defaultUserSettings(timezone, true),
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
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}

		wasDailyQuestionsEnabled := user.Settings.Telegram.DailyQuestionsEnabled
		previousIgnoredAudioLanguages := append([]enums.Language(nil), user.Settings.IgnoredAudioLanguages...)
		previousIgnoredDescriptionLanguages := append([]enums.Language(nil), user.Settings.IgnoredDescriptionLanguages...)
		previousMainLearningLanguage := user.Settings.MainLearningLanguage
		settings.Telegram.BotEnabled = user.Settings.Telegram.BotEnabled
		settings = settings.WithDefaults()
		newlyIgnoredLanguages := newlyIgnoredAudioLanguages(previousIgnoredAudioLanguages, settings.IgnoredAudioLanguages)
		newlyIgnoredDescriptionLanguages := newlyIgnoredAudioLanguages(previousIgnoredDescriptionLanguages, settings.IgnoredDescriptionLanguages)
		descriptionEligibilityChanged := previousMainLearningLanguage != settings.MainLearningLanguage ||
			containsLanguage(newlyIgnoredDescriptionLanguages, settings.MainLearningLanguage)

		user.Settings = settings

		if err := tx.Model(&user).Update("settings", settings).Error; err != nil {
			return err
		}
		if descriptionEligibilityChanged {
			if err := cancelInProgressDescriptionExercises(tx, user.ID); err != nil {
				return err
			}
		}

		if wasDailyQuestionsEnabled && !settings.Telegram.DailyQuestionsEnabled {
			if err := DeletePendingExercisesByUserID(tx, user.ID); err != nil {
				return err
			}
		} else {
			if err := replacePendingAudioExercisesForLanguages(tx, user.ID, newlyIgnoredLanguages); err != nil {
				return err
			}
			if descriptionEligibilityChanged {
				if err := replacePendingDescriptionExercises(tx, user.ID); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func UpdateUserTranslationTargetLanguageByID(userID uint, language enums.Language) (*models.User, error) {
	var user models.User

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}

		settings := user.Settings
		if settings.TranslationTargetLanguage == language {
			return nil
		}

		if settings.TranslationSourceLanguage == language {
			settings.TranslationSourceLanguage = settings.TranslationTargetLanguage
		}
		settings.TranslationTargetLanguage = language
		user.Settings = settings

		return tx.Model(&user).Update("settings", settings).Error
	})
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func UpdateUserTelegramBotEnabled(telegramID int64, botEnabled bool) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
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

func UpdateUserTelegramDailyQuestionsEnabled(telegramID int64, toggle bool) (*models.User, error) {
	var user models.User

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}

			return err
		}

		settings := user.Settings
		nextValue := settings.Telegram.DailyQuestionsEnabled
		if toggle {
			nextValue = !nextValue
		}

		if settings.Telegram.DailyQuestionsEnabled == nextValue {
			return nil
		}

		settings.Telegram.DailyQuestionsEnabled = nextValue
		user.Settings = settings

		if err := tx.Model(&user).Update("settings", settings).Error; err != nil {
			return err
		}

		if !nextValue {
			if err := DeletePendingExercisesByUserID(tx, user.ID); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if user.ID == 0 {
		return nil, nil
	}

	return &user, nil
}

func UpdateUserTelegramState(telegramID int64, state enums.TelegramState) (bool, error) {
	updated := false

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
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

func IsUserDeletedByTelegramID(telegramID int64) (bool, error) {
	var user models.User

	err := db.DB.Unscoped().Select("id", "deleted_at").Where("telegram_id = ?", telegramID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}

		return false, err
	}

	return user.DeletedAt.Valid, nil
}

func EnsureUserByTelegramID(telegramID int64, username string, firstName string, lastName string) (bool, error) {
	active := false
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User

		err := tx.Unscoped().Where("telegram_id = ?", telegramID).First(&user).Error
		if err == nil {
			active = !user.DeletedAt.Valid
			return nil
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		user = models.User{
			TelegramID: telegramID,
			Username:   username,
			Name:       strings.TrimSpace(firstName + " " + lastName),
			Settings:   defaultUserSettings("", true),
		}

		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		active = true
		return nil
	})

	return active, err
}

func UpdateUserTranslationLanguage(telegramID int64, isSource bool, lang enums.Language) (*models.User, error) {
	var user models.User

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}

			return err
		}

		settings := user.Settings
		if (isSource && settings.TranslationTargetLanguage == lang) ||
			(!isSource && settings.TranslationSourceLanguage == lang) {
			return nil
		}

		if isSource {
			settings.TranslationSourceLanguage = lang
		} else {
			settings.TranslationTargetLanguage = lang
		}

		user.Settings = settings
		return tx.Model(&user).Update("settings", settings).Error
	})
	if err != nil {
		return nil, err
	}

	if user.ID == 0 {
		return nil, nil
	}

	return &user, nil
}

func SwapUserTranslationLanguages(telegramID int64) (*models.User, error) {
	var user models.User

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}

			return err
		}

		settings := user.Settings
		settings.TranslationSourceLanguage, settings.TranslationTargetLanguage =
			settings.TranslationTargetLanguage, settings.TranslationSourceLanguage

		user.Settings = settings
		return tx.Model(&user).Update("settings", settings).Error
	})
	if err != nil {
		return nil, err
	}

	if user.ID == 0 {
		return nil, nil
	}

	return &user, nil
}

func UpdateUserSystemLanguage(telegramID int64, lang enums.Language) (*models.User, error) {
	var user models.User

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}

			return err
		}

		settings := user.Settings
		settings.SystemLanguage = lang

		user.Settings = settings
		return tx.Model(&user).Update("settings", settings).Error
	})
	if err != nil {
		return nil, err
	}

	if user.ID == 0 {
		return nil, nil
	}

	return &user, nil
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
