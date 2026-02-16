package controllers

import (
	"errors"
	nethttp "net/http"
	"strings"
	"termorize/src/enums"
	"termorize/src/http/validators"
	"termorize/src/models"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UpdateSettingsRequest struct {
	NativeLanguage       enums.Language                `json:"native_language" binding:"required,enum=Language"`
	MainLearningLanguage enums.Language                `json:"main_learning_language" binding:"required,enum=Language"`
	TimeZone             string                        `json:"time_zone" binding:"required,timezone"`
	Telegram             UpdateSettingsTelegramRequest `json:"telegram" binding:"required"`
}

type UpdateSettingsTelegramRequest struct {
	DailyQuestionsEnabled  bool                                        `json:"daily_questions_enabled"`
	DailyQuestionsCount    uint                                        `json:"daily_questions_count" binding:"max=100"`
	DailyQuestionsSchedule []UpdateSettingsTelegramScheduleItemRequest `json:"daily_questions_schedule" binding:"required,dive"`
}

type UpdateSettingsTelegramScheduleItemRequest struct {
	From string `json:"from" binding:"required,hhmm"`
	To   string `json:"to" binding:"required,hhmm"`
}

func GetSettings(c *gin.Context) {
	c.JSON(nethttp.StatusOK, gin.H{
		"languages": enums.AllLanguages(),
	})
}

func UpdateSettings(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req UpdateSettingsRequest
	if !validators.BindJSONWithErrors(c, &req) {
		return
	}

	settings := models.UserSettings{
		NativeLanguage:       req.NativeLanguage,
		MainLearningLanguage: req.MainLearningLanguage,
		TimeZone:             strings.TrimSpace(req.TimeZone),
		Telegram: models.UserTelegramSettings{
			DailyQuestionsEnabled: req.Telegram.DailyQuestionsEnabled,
			DailyQuestionsCount:   req.Telegram.DailyQuestionsCount,
			DailyQuestionsSchedule: func() []models.UserTelegramQuestionsScheduleItem {
				schedule := make([]models.UserTelegramQuestionsScheduleItem, len(req.Telegram.DailyQuestionsSchedule))
				for i, item := range req.Telegram.DailyQuestionsSchedule {
					schedule[i] = models.UserTelegramQuestionsScheduleItem{From: item.From, To: item.To}
				}
				return schedule
			}(),
		},
	}

	user, err := services.UpdateUserSettings(userID, settings)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(nethttp.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, user)
}
