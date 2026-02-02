package auth

import (
	"strconv"
	"termorize/src/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func IssueJWT(userID uint) string {
	claims := jwt.RegisteredClaims{
		Subject:   strconv.Itoa(int(userID)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.GetJWTExpirationTime())),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, _ := token.SignedString([]byte(config.GetSecret()))

	return signedToken
}

func DecodeJWT(token string) (uint, error) {
	claims := &jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.GetSecret()), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		return 0, err
	}

	sub, _ := claims.GetSubject()
	userID, err := strconv.Atoi(sub)
	if err != nil {
		return 0, err
	}

	return uint(userID), nil
}
