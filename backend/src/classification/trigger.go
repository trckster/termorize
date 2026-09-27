package classification

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

const TriggerPath = "/api/internal/classification/sweep"

func TriggerToken(secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("termorize:classification-sweep"))
	return hex.EncodeToString(mac.Sum(nil))
}

func Trigger(ctx context.Context, client *http.Client, backendURL, secret string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(backendURL, "/")+TriggerPath, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+TriggerToken(secret))
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("classification sweep trigger failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("classification sweep trigger returned HTTP %d", response.StatusCode)
	}
	return nil
}
