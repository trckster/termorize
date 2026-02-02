package models

import "time"

type User struct {
	ID         uint `gorm:"primaryKey"`
	Username   string
	TelegramID int64 `gorm:"uniqueIndex"`
	Name       string
	PhotoUrl   string
	CreatedAt  time.Time `gorm:"autoCreateTime:milli"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime:milli"`
}
