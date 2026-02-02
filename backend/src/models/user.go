package models

import "time"

type User struct {
	ID         uint      `json:"id"`
	Username   string    `json:"username"`
	TelegramID int64     `gorm:"uniqueIndex" json:"-"`
	Name       string    `json:"name"`
	PhotoUrl   string    `json:"photo_url"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"-"`
}
