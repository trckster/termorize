package config

import (
	"net/url"
	"os"
	"termorize/src/logger"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env       string
	PublicURL string
	Port      string
	Secret    string

	DBHost             string
	DBPort             string
	DBName             string
	DBUser             string
	DBPassword         string
	TelegramBotToken   string
	TelegramWebhookURL string
	GoogleApiKey       string

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
		logger.L().Infow("no .env file found, using environment variables")
	}

	config = &Config{
		Env:       getEnv("ENV", "prod"),
		PublicURL: getEnv("PUBLIC_URL", "http://localhost:3000"),
		Port:      getEnv("PORT", "8080"),
		Secret:    getRequiredEnv("SECRET"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "termorize"),
		DBUser:     getRequiredEnv("DB_USER"),
		DBPassword: getRequiredEnv("DB_PASSWORD"),

		TelegramBotToken:   getRequiredEnv("TELEGRAM_BOT_TOKEN"),
		TelegramWebhookURL: getEnv("TELEGRAM_WEBHOOK_URL", ""),
		GoogleApiKey:       getRequiredEnv("GOOGLE_API_KEY"),

		JWTExpirationTime: 12 * time.Hour,
	}
}

func GetDomain() string {
	if IsLocal() {
		return "localhost"
	}

	parseUrl, err := url.Parse(config.PublicURL)
	if err != nil {
		panic("invalid public url")
	}

	return parseUrl.Hostname()
}

// GetPublicURL returns frontend URL
func GetPublicURL() string {
	return config.PublicURL
}

func IsLocal() bool {
	return config.Env == "local"
}

func IsProduction() bool { return config.Env == "prod" }

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

func GetTelegramWebhookURL() string {
	return config.TelegramWebhookURL
}

func GetGoogleApiKey() string {
	return config.GoogleApiKey
}

func GetJWTExpirationTime() time.Duration {
	return config.JWTExpirationTime
}
