package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port   string
	Secret string

	DBHost           string
	DBPort           string
	DBName           string
	DBUser           string
	DBPassword       string
	TelegramBotToken string

	JWTExpirationTime time.Duration
}

var config *Config

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getRequiredEnv(key string) string {
	value := os.Getenv(key)

	if value == "" {
		panic("Required environment variable is missing: " + key)
	}

	return value
}

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config = &Config{
		Port:   getEnv("PORT", "8080"),
		Secret: getRequiredEnv("SECRET"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "termorize"),
		DBUser:     getRequiredEnv("DB_USER"),
		DBPassword: getRequiredEnv("DB_PASSWORD"),

		TelegramBotToken: getRequiredEnv("TELEGRAM_BOT_TOKEN"),

		JWTExpirationTime: 12 * time.Hour,
	}
}

func GetPort() string {
	return config.Port
}

func GetSecret() string {
	return config.Secret
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

func GetJWTExpirationTime() time.Duration {
	return config.JWTExpirationTime
}
