package telegram

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"termorize/src/config"
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

	log.Println("Telegram webhook set successfully at " + config.GetTelegramWebhookURL())

	return nil
}
