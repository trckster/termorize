package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string

	DBHost           string
	DBPort           string
	DBName           string
	DBUser           string
	DBPassword       string
	TelegramBotToken string
}

var config *Config

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config = &Config{
		Port: getEnv("PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "termorize"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),

		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
	}
}

func GetPort() string {
	return config.Port
}

func GetDBHost() string {
	return config.DBHost
}

func GetDBPort() string {
	return config.DBPort
}

func GetDBName() string {
	return config.DBName
}

func GetDBUser() string {
	return config.DBUser
}

func GetDBPassword() string {
	return config.DBPassword
}

func GetTelegramBotToken() string {
	return config.TelegramBotToken
}
