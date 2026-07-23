package testkit

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"termorize/src/auth"
	"termorize/src/config"

	"github.com/golang-jwt/jwt/v5"
)

type TelegramLoginProfile struct {
	ID       int64
	Username string
	Name     string
}

func MockTelegramLogin(t *testing.T, profile TelegramLoginProfile) {
	t.Helper()

	if profile.ID == 0 {
		profile.ID = 777000
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("testkit: failed to generate RSA key: %v", err)
	}
	const kid = "testkit-key"

	idToken := signTelegramIDToken(t, privateKey, kid, profile)
	jwks := buildJWKS(kid, &privateKey.PublicKey)

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fake-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     idToken,
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})

	server := httptest.NewServer(mux)
	restore := auth.SetTelegramOAuthEndpointsForTest(server.URL+"/token", server.URL+"/jwks")

	t.Cleanup(func() {
		restore()
		server.Close()
	})
}

func signTelegramIDToken(t *testing.T, key *rsa.PrivateKey, kid string, profile TelegramLoginProfile) string {
	t.Helper()

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":                "https://oauth.telegram.org",
		"aud":                config.GetTelegramLoginClientID(),
		"sub":                strconv.FormatInt(profile.ID, 10),
		"iat":                now.Unix(),
		"exp":                now.Add(time.Hour).Unix(),
		"id":                 profile.ID,
		"name":               profile.Name,
		"preferred_username": profile.Username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("testkit: failed to sign id_token: %v", err)
	}
	return signed
}

func buildJWKS(kid string, pub *rsa.PublicKey) map[string]any {
	return map[string]any{
		"keys": []map[string]any{
			{
				"kid": kid,
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			},
		},
	}
}

func BuildTelegramInitData(userID int64, username, firstName string) string {
	userJSON := fmt.Sprintf(`{"id":%d,"username":%q,"first_name":%q}`, userID, username, firstName)

	values := url.Values{}
	values.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	values.Set("user", userJSON)

	values.Set("hash", telegramInitDataHash(values))
	return values.Encode()
}

func telegramInitDataHash(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key == "hash" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	dataCheckString := strings.Join(parts, "\n")

	secretMac := hmac.New(sha256.New, []byte("WebAppData"))
	secretMac.Write([]byte(config.GetTelegramBotToken()))
	secret := secretMac.Sum(nil)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(dataCheckString))
	return hex.EncodeToString(mac.Sum(nil))
}
