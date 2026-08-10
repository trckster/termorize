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

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string

	TelegramBotToken          string
	TelegramLoginClientID     string
	TelegramLoginClientSecret string
	TelegramWebhookURL        string

	GoogleApiKey string

	OpenRouterApiKey string

	SentryDSN string

	JWTExpirationTime time.Duration
}

var config *Config

const (
	openRouterModel            = "google/gemini-2.5-flash"
	openRouterTTSModel         = "google/gemini-3.1-flash-tts-preview"
	openRouterTTSVoice         = "Kore"
	openRouterFallbackTTSModel = "microsoft/mai-voice-2"
)

type OpenRouterTTSConfig struct {
	Model          string
	Voice          string
	ResponseFormat string
	LanguagePrompt bool
}

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

	publicURL := getEnv("PUBLIC_URL", "http://localhost:3000")

	config = &Config{
		Env:       getEnv("ENV", "prod"),
		PublicURL: publicURL,
		Port:      getEnv("PORT", "8080"),
		Secret:    getRequiredEnv("SECRET"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "termorize"),
		DBUser:     getRequiredEnv("DB_USER"),
		DBPassword: getRequiredEnv("DB_PASSWORD"),

		TelegramBotToken:          getRequiredEnv("TELEGRAM_BOT_TOKEN"),
		TelegramWebhookURL:        getEnv("TELEGRAM_WEBHOOK_URL", ""),
		TelegramLoginClientID:     getRequiredEnv("TELEGRAM_LOGIN_CLIENT_ID"),
		TelegramLoginClientSecret: getRequiredEnv("TELEGRAM_LOGIN_CLIENT_SECRET"),

		GoogleApiKey: getRequiredEnv("GOOGLE_API_KEY"),

		OpenRouterApiKey: getEnv("OPENROUTER_API_KEY", ""),

		SentryDSN: getEnv("SENTRY_DSN", ""),

		JWTExpirationTime: 7 * 24 * time.Hour,
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

func GetTelegramLoginClientID() string {
	return config.TelegramLoginClientID
}

func GetTelegramLoginClientSecret() string {
	return config.TelegramLoginClientSecret
}

func GetTelegramLoginRedirectURL() string {
	return config.PublicURL + "/login/telegram/callback"
}

func GetGoogleApiKey() string {
	return config.GoogleApiKey
}

func GetOpenRouterApiKey() string {
	return config.OpenRouterApiKey
}

func GetOpenRouterModel() string {
	return openRouterModel
}

func GetOpenRouterTTSModel() string {
	return openRouterTTSModel
}

func GetOpenRouterTTSVoice() string {
	return openRouterTTSVoice
}

func GetOpenRouterTTSConfigs(language string) []OpenRouterTTSConfig {
	return []OpenRouterTTSConfig{
		{Model: openRouterTTSModel, Voice: openRouterTTSVoice, ResponseFormat: "pcm", LanguagePrompt: true},
		{Model: openRouterFallbackTTSModel, Voice: getOpenRouterFallbackTTSVoice(language), ResponseFormat: "mp3"},
	}
}

func getOpenRouterFallbackTTSVoice(language string) string {
	voices := map[string]string{
		"en": "en-US-Harper:MAI-Voice-2",
		"ru": "ru-RU-SvetlanaNeural",
		"it": "it-IT-Rosa:MAI-Voice-2",
		"de": "de-DE-Mia:MAI-Voice-2",
		"es": "es-ES-Marta:MAI-Voice-2",
		"fr": "fr-FR-Soleil:MAI-Voice-2",
		"pl": "pl-PL-ZofiaNeural",
		"tr": "tr-TR-EmelNeural",
		"pt": "pt-BR-FranciscaNeural",
		"uk": "uk-UA-PolinaNeural",
	}
	if voice, ok := voices[language]; ok {
		return voice
	}
	return voices["en"]
}

func GetJWTExpirationTime() time.Duration {
	return config.JWTExpirationTime
}

func GetSentryDSN() string {
	return config.SentryDSN
}

func GetEnv() string {
	return config.Env
}
