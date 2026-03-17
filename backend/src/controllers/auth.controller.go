package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"termorize/src/auth"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/http/validators"
	"termorize/src/models"
	"termorize/src/services"
	"time"

	"github.com/gin-gonic/gin"
)

func StartTelegramLogin(c *gin.Context) {
	if !auth.IsTelegramLoginConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "telegram login is not configured"})
		return
	}

	redirectURI := getTelegramLoginRedirectURL(c)
	if redirectURI == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "telegram login redirect is invalid"})
		return
	}

	session, err := auth.NewTelegramLoginSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start telegram login"})
		return
	}
	session.RedirectURI = redirectURI

	sessionToken, err := auth.IssueTelegramLoginSessionToken(*session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start telegram login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"auth_url": auth.BuildTelegramLoginURL(*session, sessionToken)})
}

func CompleteTelegramLogin(c *gin.Context) {
	var request auth.TelegramLoginCallbackRequest
	if !validators.BindJSONWithErrors(c, &request) {
		return
	}

	var (
		profile *auth.TelegramUserProfile
		err     error
	)

	if strings.TrimSpace(request.InitData) != "" {
		profile, err = auth.ValidateTelegramInitData(request.InitData)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "telegram login failed",
				"details": err.Error(),
			})
			return
		}
	} else {
		if strings.TrimSpace(request.Code) == "" || strings.TrimSpace(request.State) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "telegram login payload is invalid"})
			return
		}

		session, err := auth.DecodeTelegramLoginSessionToken(request.State)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "telegram login session is invalid"})
			return
		}

		profile, err = auth.ExchangeTelegramLoginCode(request.Code, session.CodeVerifier, session.RedirectURI, session.Nonce)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "telegram login failed",
				"details": err.Error(),
			})
			return
		}
	}

	user, err := services.CreateOrUpdateUserByTelegramProfile(*profile, getRequestTimeZone(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save user"})
		return
	}

	auth.SetAuthCookie(c, auth.IssueJWT(user.ID))
	c.JSON(http.StatusOK, user)
}

func Me(c *gin.Context) {
	userID := c.MustGet("userID")

	var user models.User
	db.DB.Where("id = ?", userID).First(&user)

	c.JSON(http.StatusOK, user)
}

func Logout(c *gin.Context) {
	auth.DeleteAuthCookie(c)
}

func getRequestTimeZone(c *gin.Context) string {
	timezone := strings.TrimSpace(c.GetHeader("X-Timezone"))

	if timezone == "" {
		return "UTC"
	}

	if _, err := time.LoadLocation(timezone); err != nil {
		return "UTC"
	}

	return timezone
}

func getTelegramLoginRedirectURL(c *gin.Context) string {
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if origin == "" {
		return config.GetTelegramLoginRedirectURL()
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil || parsedOrigin.Scheme == "" || parsedOrigin.Host == "" {
		return config.GetTelegramLoginRedirectURL()
	}

	return fmt.Sprintf("%s://%s/login/telegram/callback", parsedOrigin.Scheme, parsedOrigin.Host)
}
