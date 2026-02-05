package database

import (
	"fmt"
	"termorize/src/config"
	"termorize/src/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.GetDBHost(),
		config.GetDBPort(),
		config.GetDBUser(),
		config.GetDBPassword(),
		config.GetDBName(),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db

	return nil
}

func Migrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Word{},
		&models.Translation{},
		&models.Vocabulary{},
	)
}
