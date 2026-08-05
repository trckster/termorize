package controllers

import (
	"errors"
	"net/http"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const recentUsersLimit = 50

type recentUser struct {
	ID             uint      `json:"id"`
	Name           string    `json:"name"`
	Username       string    `json:"username"`
	VocabularySize int64     `json:"vocabulary_size"`
	OpenRouterCost float64   `json:"openrouter_cost"`
	LatestUsage    time.Time `json:"latest_usage"`
}

type recentUsersResponse struct {
	Data  []recentUser `json:"data"`
	Total int64        `json:"total"`
}

func GetAdminUsers(c *gin.Context) {
	userID := c.MustGet("userID")

	var viewer models.User
	if err := db.DB.Select("is_admin").First(&viewer, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		ServerError(c, err)
		return
	}

	if !viewer.IsAdmin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	response := recentUsersResponse{Data: make([]recentUser, 0)}
	if err := db.DB.Model(&models.User{}).Count(&response.Total).Error; err != nil {
		ServerError(c, err)
		return
	}

	err := db.DB.Raw(`
		SELECT
			users.id,
			users.name,
			users.username,
			COALESCE(vocabulary_stats.size, 0) AS vocabulary_size,
			COALESCE(openrouter_stats.cost, 0) AS open_router_cost,
			MAX(activity.used_at) AS latest_usage
		FROM users
		JOIN (
			SELECT user_id, created_at AS used_at
			FROM vocabulary
			UNION ALL
			SELECT user_id, finished_at AS used_at
			FROM exercises
			WHERE status IN (?, ?) AND finished_at IS NOT NULL
			UNION ALL
			SELECT user_id, created_at AS used_at
			FROM openrouter_usages
		) AS activity ON activity.user_id = users.id
		LEFT JOIN (
			SELECT user_id, COUNT(*) AS size
			FROM vocabulary
			WHERE deleted_at IS NULL
			GROUP BY user_id
		) AS vocabulary_stats ON vocabulary_stats.user_id = users.id
		LEFT JOIN (
			SELECT user_id, SUM(cost) AS cost
			FROM openrouter_usages
			GROUP BY user_id
		) AS openrouter_stats ON openrouter_stats.user_id = users.id
		GROUP BY users.id, users.name, users.username, vocabulary_stats.size, openrouter_stats.cost
		ORDER BY latest_usage DESC, users.id DESC
		LIMIT ?
	`, enums.ExerciseStatusCompleted, enums.ExerciseStatusFailed, recentUsersLimit).Scan(&response.Data).Error
	if err != nil {
		ServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}
