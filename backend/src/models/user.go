package models

import (
	"database/sql/driver"
	"encoding/json"
	"termorize/src/enums"
	"time"
)

type UserSettings struct {
	SystemLanguage            enums.Language       `json:"system_language"`
	MainLearningLanguage      enums.Language       `json:"main_learning_language"`
	TranslationSourceLanguage enums.Language       `json:"translation_source_language"`
	TranslationTargetLanguage enums.Language       `json:"translation_target_language"`
	TimeZone                  string               `json:"time_zone"`
	Telegram                  UserTelegramSettings `json:"telegram"`
}

type UserTelegramSettings struct {
	BotEnabled             bool                                `json:"bot_enabled"`
	DailyQuestionsEnabled  bool                                `json:"daily_questions_enabled"`
	DailyQuestionsCount    uint                                `json:"daily_questions_count"`
	DailyQuestionsSchedule []UserTelegramQuestionsScheduleItem `json:"daily_questions_schedule"`
}

type UserTelegramQuestionsScheduleItem struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (s UserSettings) Value() (driver.Value, error) {
	s = s.WithDefaults()
	return json.Marshal(s)
}

func (p *UserSettings) Scan(value any) error {
	if err := json.Unmarshal(value.([]byte), p); err != nil {
		return err
	}

	*p = p.WithDefaults()
	return nil
}

func (s UserSettings) WithDefaults() UserSettings {
	if s.SystemLanguage == "" {
		s.SystemLanguage = enums.LanguageRu
	}

	if s.MainLearningLanguage == "" {
		s.MainLearningLanguage = enums.LanguageEn
	}

	if s.TranslationSourceLanguage == "" {
		s.TranslationSourceLanguage = enums.LanguageEn
	}

	if s.TranslationTargetLanguage == "" {
		s.TranslationTargetLanguage = enums.LanguageRu
	}

	return s
}

type User struct {
	ID            uint                `json:"id"`
	Username      string              `json:"username"`
	TelegramID    int64               `json:"-"`
	Name          string              `json:"name"`
	Settings      UserSettings        `json:"settings"`
	TelegramState enums.TelegramState `json:"-" gorm:"default:''"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"-"`
}
