package services

import (
	"strings"
	"termorize/src/auth"
	"termorize/src/data/db"
	"termorize/src/models"
)

func CreateOrUpdateUserByTelegramAuthData(data auth.TelegramAuthData) (*models.User, error) {
	var user models.User
	result := db.DB.Where("telegram_id = ?", data.ID).First(&user)

	if result.Error == nil {
		return updateUserByTelegramAuthData(&user, data)
	}

	return createUserByTelegramAuthData(data)
}

func createUserByTelegramAuthData(data auth.TelegramAuthData) (*models.User, error) {
	user := models.User{
		TelegramID: data.ID,
		Username:   data.Username,
		Name:       strings.TrimSpace(data.FirstName + " " + data.LastName),
		PhotoUrl:   data.PhotoUrl,
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
