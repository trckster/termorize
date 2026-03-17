package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"termorize/src/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const telegramIssuer = "https://oauth.telegram.org"
const telegramAuthorizationEndpoint = telegramIssuer + "/auth"
const telegramTokenEndpoint = telegramIssuer + "/token"
const telegramJWKSURL = telegramIssuer + "/.well-known/jwks.json"
const telegramLoginScope = "openid profile telegram:bot_access"
const telegramLoginSessionTTL = 60 * time.Minute
const telegramInitDataTTL = 24 * time.Hour
const telegramWebAppDataKey = "WebAppData"

type TelegramLoginSession struct {
	CodeVerifier string
	Nonce        string
	RedirectURI  string
	ExpiresAt    time.Time
}

type TelegramLoginSessionClaims struct {
	CodeVerifier string `json:"code_verifier"`
	Nonce        string `json:"nonce"`
	RedirectURI  string `json:"redirect_uri"`
	jwt.RegisteredClaims
}

type TelegramLoginCallbackRequest struct {
	Code     string `json:"code"`
	State    string `json:"state"`
	InitData string `json:"init_data"`
}

type TelegramUserProfile struct {
	ID       int64
	Username string
	Name     string
}

type telegramTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	IDToken     string `json:"id_token"`
}

type telegramJWKSet struct {
	Keys []telegramJWK `json:"keys"`
}

type telegramJWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
}

type telegramIDTokenClaims struct {
	ID                telegramNumericString `json:"id"`
	Name              string                `json:"name"`
	PreferredUsername string                `json:"preferred_username"`
	Picture           string                `json:"picture"`
	PhoneNumber       string                `json:"phone_number"`
	Nonce             string                `json:"nonce"`
	jwt.RegisteredClaims
}

type telegramWebAppUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type telegramNumericString int64

func (v *telegramNumericString) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*v = 0
		return nil
	}

	var number int64
	if err := json.Unmarshal(data, &number); err == nil {
		*v = telegramNumericString(number)
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}

	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return err
	}

	*v = telegramNumericString(parsed)
	return nil
}

func IsTelegramLoginConfigured() bool {
	return config.GetTelegramLoginClientID() != "" && config.GetTelegramLoginClientSecret() != ""
}

func NewTelegramLoginSession() (*TelegramLoginSession, error) {
	codeVerifier, err := randomBase64URL(64)
	if err != nil {
		return nil, err
	}

	nonce, err := randomBase64URL(32)
	if err != nil {
		return nil, err
	}

	return &TelegramLoginSession{
		CodeVerifier: codeVerifier,
		Nonce:        nonce,
		ExpiresAt:    time.Now().Add(telegramLoginSessionTTL),
	}, nil
}

func IssueTelegramLoginSessionToken(session TelegramLoginSession) (string, error) {
	claims := TelegramLoginSessionClaims{
		CodeVerifier: session.CodeVerifier,
		Nonce:        session.Nonce,
		RedirectURI:  session.RedirectURI,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(session.ExpiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(config.GetSecret()))
}

func DecodeTelegramLoginSessionToken(tokenString string) (*TelegramLoginSession, error) {
	claims := &TelegramLoginSessionClaims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.GetSecret()), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	return &TelegramLoginSession{
		CodeVerifier: claims.CodeVerifier,
		Nonce:        claims.Nonce,
		RedirectURI:  claims.RedirectURI,
		ExpiresAt:    claims.ExpiresAt.Time,
	}, nil
}

func BuildTelegramLoginURL(session TelegramLoginSession, state string) string {
	values := url.Values{}
	values.Set("client_id", config.GetTelegramLoginClientID())
	values.Set("redirect_uri", session.RedirectURI)
	values.Set("response_type", "code")
	values.Set("scope", telegramLoginScope)
	values.Set("state", state)
	values.Set("nonce", session.Nonce)
	values.Set("code_challenge", buildCodeChallenge(session.CodeVerifier))
	values.Set("code_challenge_method", "S256")

	return telegramAuthorizationEndpoint + "?" + values.Encode()
}

func ExchangeTelegramLoginCode(code string, codeVerifier string, redirectURI string, expectedNonce string) (*TelegramUserProfile, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", config.GetTelegramLoginClientID())
	form.Set("code_verifier", codeVerifier)

	request, err := http.NewRequest(http.MethodPost, telegramTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(config.GetTelegramLoginClientID(), config.GetTelegramLoginClientSecret())

	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("telegram token exchange failed: %s", strings.TrimSpace(string(body)))
	}

	var tokenResponse telegramTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, err
	}

	if tokenResponse.IDToken == "" {
		return nil, errors.New("telegram token response missing id_token")
	}

	return ValidateTelegramIDToken(tokenResponse.IDToken, expectedNonce)
}

func ValidateTelegramIDToken(idToken string, expectedNonce string) (*TelegramUserProfile, error) {
	claims := &telegramIDTokenClaims{}

	_, err := jwt.ParseWithClaims(idToken, claims, func(token *jwt.Token) (interface{}, error) {
		kid, _ := token.Header["kid"].(string)
		return fetchTelegramPublicKey(kid)
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithIssuer(telegramIssuer), jwt.WithAudience(config.GetTelegramLoginClientID()))
	if err != nil {
		return nil, err
	}

	if expectedNonce != "" && claims.Nonce != "" && claims.Nonce != expectedNonce {
		return nil, errors.New("telegram id_token nonce mismatch")
	}

	telegramID := int64(claims.ID)
	if telegramID == 0 {
		parsedSub, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			return nil, errors.New("telegram id_token missing user id")
		}

		telegramID = parsedSub
	}

	name := strings.TrimSpace(claims.Name)
	if name == "" {
		name = claims.PreferredUsername
	}

	return &TelegramUserProfile{
		ID:       telegramID,
		Username: claims.PreferredUsername,
		Name:     name,
	}, nil
}

func ValidateTelegramInitData(initData string) (*TelegramUserProfile, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, errors.New("telegram init data is malformed")
	}

	receivedHash := strings.TrimSpace(values.Get("hash"))
	if receivedHash == "" {
		return nil, errors.New("telegram init data missing hash")
	}

	authDateRaw := strings.TrimSpace(values.Get("auth_date"))
	if authDateRaw == "" {
		return nil, errors.New("telegram init data missing auth_date")
	}

	authDateUnix, err := strconv.ParseInt(authDateRaw, 10, 64)
	if err != nil {
		return nil, errors.New("telegram init data auth_date is invalid")
	}

	authDate := time.Unix(authDateUnix, 0)
	if time.Since(authDate) > telegramInitDataTTL || authDate.After(time.Now().Add(1*time.Minute)) {
		return nil, errors.New("telegram init data has expired")
	}

	dataCheckString := buildTelegramInitDataCheckString(values)
	secret := telegramInitDataSecret(config.GetTelegramBotToken())
	expectedHash := telegramInitDataHash(secret, dataCheckString)
	if !hmac.Equal([]byte(strings.ToLower(receivedHash)), []byte(expectedHash)) {
		return nil, errors.New("telegram init data hash is invalid")
	}

	rawUser := strings.TrimSpace(values.Get("user"))
	if rawUser == "" {
		return nil, errors.New("telegram init data missing user")
	}

	var user telegramWebAppUser
	if err := json.Unmarshal([]byte(rawUser), &user); err != nil {
		return nil, errors.New("telegram init data user is invalid")
	}

	if user.ID == 0 {
		return nil, errors.New("telegram init data missing user id")
	}

	name := strings.TrimSpace(strings.Join([]string{user.FirstName, user.LastName}, " "))
	if name == "" {
		name = user.Username
	}

	return &TelegramUserProfile{
		ID:       user.ID,
		Username: user.Username,
		Name:     name,
	}, nil
}

func fetchTelegramPublicKey(expectedKid string) (*rsa.PublicKey, error) {
	response, err := (&http.Client{Timeout: 10 * time.Second}).Get(telegramJWKSURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("telegram jwks fetch failed: %s", strings.TrimSpace(string(body)))
	}

	var jwks telegramJWKSet
	if err := json.NewDecoder(response.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	for _, key := range jwks.Keys {
		if key.Kid != expectedKid || key.Kty != "RSA" {
			continue
		}

		return jwkToRSAPublicKey(key)
	}

	return nil, errors.New("telegram jwks key not found")
}

func buildTelegramInitDataCheckString(values url.Values) string {
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

	return strings.Join(parts, "\n")
}

func telegramInitDataSecret(botToken string) []byte {
	mac := hmac.New(sha256.New, []byte(telegramWebAppDataKey))
	mac.Write([]byte(botToken))
	return mac.Sum(nil)
}

func telegramInitDataHash(secret []byte, dataCheckString string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(dataCheckString))
	return hex.EncodeToString(mac.Sum(nil))
}

func jwkToRSAPublicKey(key telegramJWK) (*rsa.PublicKey, error) {
	modulusBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, err
	}

	exponentBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, err
	}

	modulus := new(big.Int).SetBytes(modulusBytes)
	exponent := new(big.Int).SetBytes(exponentBytes)

	return &rsa.PublicKey{
		N: modulus,
		E: int(exponent.Int64()),
	}, nil
}

func buildCodeChallenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func randomBase64URL(length int) (string, error) {
	bytesBuffer := make([]byte, length)
	if _, err := rand.Read(bytesBuffer); err != nil {
		return "", err
	}

	encoded := base64.RawURLEncoding.EncodeToString(bytesBuffer)
	if len(encoded) > 128 {
		encoded = encoded[:128]
	}

	return encoded, nil
}
