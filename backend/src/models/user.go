package models

import "time"

type User struct {
	ID         uint   `gorm:"primaryKey"`
	Username   string `gorm:"uniqueIndex"`
	TelegramID int64  `gorm:"uniqueIndex"`
	Name       string
	PhotoUrl   string
	CreatedAt  time.Time `gorm:"autoCreateTime:milli"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime:milli"`
}
