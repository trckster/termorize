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
