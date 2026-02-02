package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"termorize/src/config"
	"termorize/src/utils"
	"time"
)

type TelegramAuthData struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	PhotoUrl  string `json:"photo_url"`
	AuthDate  int64  `json:"auth_date"` // In seconds
	Hash      string `json:"hash"`
}

func ValidateTelegramAuth(data TelegramAuthData) bool {
	// If auth happened more than hour ago, we assume
	if !utils.WasWithin(data.AuthDate*1000, time.Hour) {
		return false
	}

	checkString := buildDataCheckString(data)
	secretKey := sha256.Sum256([]byte(config.GetTelegramBotToken()))
	expectedHash := hmac.New(sha256.New, secretKey[:])
	expectedHash.Write([]byte(checkString))
	computedHash := hex.EncodeToString(expectedHash.Sum(nil))

	return computedHash == data.Hash
}

func buildDataCheckString(data TelegramAuthData) string {
	fields := map[string]string{
		"auth_date":  fmt.Sprintf("%d", data.AuthDate),
		"first_name": data.FirstName,
		"id":         fmt.Sprintf("%d", data.ID),
	}

	if data.LastName != "" {
		fields["last_name"] = data.LastName
	}
	if data.Username != "" {
		fields["username"] = data.Username
	}
	if data.PhotoUrl != "" {
		fields["photo_url"] = data.PhotoUrl
	}

	var keys []string
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, fields[k]))
	}

	return strings.Join(pairs, "\n")
}
