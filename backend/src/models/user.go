package models

import (
	"database/sql/driver"
	"encoding/json"
	"termorize/src/enums"
	"time"
)

type UserSettings struct {
	NativeLanguage       enums.Language       `json:"native_language"`
	MainLearningLanguage enums.Language       `json:"main_learning_language"`
	TimeZone             string               `json:"time_zone"`
	Telegram             UserTelegramSettings `json:"telegram"`
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
	return json.Marshal(s)
}

func (p *UserSettings) Scan(value any) error {
	return json.Unmarshal(value.([]byte), p)
}

type User struct {
	ID         uint         `json:"id"`
	Username   string       `json:"username"`
	TelegramID int64        `json:"-"`
	Name       string       `json:"name"`
	PhotoUrl   string       `json:"photo_url"`
	Settings   UserSettings `json:"settings"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"-"`
}
