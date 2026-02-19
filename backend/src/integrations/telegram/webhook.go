package telegram

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"termorize/src/config"
	"termorize/src/logger"
)

type setWebhookRequest struct {
	URL         string `json:"url"`
	SecretToken string `json:"secret_token"`
}

type setWebhookResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

func BuildWebhookSecret() string {
	sum := sha256.Sum256([]byte(config.GetSecret() + ":telegram"))
	return hex.EncodeToString(sum[:])
}

func SetupWebhook() error {
	if config.GetTelegramWebhookURL() == "" {
		logger.L().Warnw("telegram webhook url is missing in env")
		return nil
	}

	body := setWebhookRequest{
		URL:         config.GetTelegramWebhookURL(),
		SecretToken: BuildWebhookSecret(),
	}

	response, err := CallAPI[setWebhookResponse]("setWebhook", body)
	if err != nil {
		return err
	}

	if !response.OK {
		return fmt.Errorf("telegram setWebhook failed: %s", response.Description)
	}

	logger.L().Infow("telegram webhook set", "url", config.GetTelegramWebhookURL())

	return nil
}
