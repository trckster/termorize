package services

import (
	"errors"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"gorm.io/gorm"
)

const recentUsersLimit = 50

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrAdminRequired = errors.New("admin access required")
)

type RecentUser struct {
	ID             uint      `json:"id"`
	Name           string    `json:"name"`
	Username       string    `json:"username"`
	VocabularySize int64     `json:"vocabulary_size"`
	LatestUsage    time.Time `json:"latest_usage"`
}

type RecentUsersResponse struct {
	Data  []RecentUser `json:"data"`
	Total int64        `json:"total"`
}

func GetUserByID(userID uint) (*models.User, error) {
	var user models.User

	if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return &user, nil
}

func GetRecentUsersForAdmin(viewerID uint) (*RecentUsersResponse, error) {
	if err := RequireAdmin(viewerID); err != nil {
		return nil, err
	}

	response := RecentUsersResponse{Data: make([]RecentUser, 0)}
	if err := db.DB.Model(&models.User{}).Count(&response.Total).Error; err != nil {
		return nil, err
	}

	err := db.DB.Raw(`
		SELECT
			users.id,
			users.name,
			users.username,
			COALESCE(vocabulary_stats.size, 0) AS vocabulary_size,
			MAX(activity.used_at) AS latest_usage
		FROM users
		JOIN (
			SELECT user_id, created_at AS used_at
			FROM vocabulary
			UNION ALL
			SELECT user_id, finished_at AS used_at
			FROM exercises
			WHERE status IN (?, ?) AND finished_at IS NOT NULL AND deleted_at IS NULL
		) AS activity ON activity.user_id = users.id
		LEFT JOIN (
			SELECT user_id, COUNT(*) AS size
			FROM vocabulary
			WHERE deleted_at IS NULL
			GROUP BY user_id
		) AS vocabulary_stats ON vocabulary_stats.user_id = users.id
		WHERE users.deleted_at IS NULL
		GROUP BY users.id, users.name, users.username, vocabulary_stats.size
		ORDER BY latest_usage DESC, users.id DESC
		LIMIT ?
	`, enums.ExerciseStatusCompleted, enums.ExerciseStatusFailed, recentUsersLimit).Scan(&response.Data).Error
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func RequireAdmin(userID uint) error {
	var viewer models.User
	if err := db.DB.Select("is_admin").First(&viewer, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}

		return err
	}
	if !viewer.IsAdmin {
		return ErrAdminRequired
	}
	return nil
}
