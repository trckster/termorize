package tests

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type adminUsersBody struct {
	Data []struct {
		ID             uint       `json:"id"`
		Name           string     `json:"name"`
		Username       string     `json:"username"`
		VocabularySize int64      `json:"vocabulary_size"`
		LatestUsage    *time.Time `json:"latest_usage"`
		DeletedAt      *time.Time `json:"deleted_at"`
	} `json:"data"`
	Total int64 `json:"total"`
}

func TestAdminUsersRequiresAuthentication(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/admin/users", nil)
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestAdminUsersRejectsNonAdmins(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/admin/users", nil)
	testkit.RequireStatus(t, rec, http.StatusForbidden)
}

func TestAdminUsersReturnsRecentActivityForAdmins(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	vocabularyUser := testkit.CreateUser(t,
		testkit.WithTelegramID(101), testkit.WithName("Vocabulary User"), testkit.WithUsername("vocabulary_user"))
	exerciseUser := testkit.CreateUser(t,
		testkit.WithTelegramID(202), testkit.WithName("Exercise User"), testkit.WithUsername("exercise_user"))
	inactiveUser := testkit.CreateUser(t)

	older := time.Date(2026, time.August, 1, 10, 0, 0, 0, time.UTC)
	newer := older.Add(24 * time.Hour)

	vocabulary := exerciseSeedVocabulary(t, vocabularyUser.ID, "recent", "недавний", enums.LanguageEn, enums.LanguageRu)
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).Where("id = ?", vocabulary.ID).Update("created_at", older).Error)
	deletedVocabulary := exerciseSeedVocabulary(t, vocabularyUser.ID, "deleted", "удалённый", enums.LanguageEn, enums.LanguageRu)
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).Where("id = ?", deletedVocabulary.ID).Updates(map[string]any{
		"created_at": older.Add(-time.Hour),
		"deleted_at": newer.Add(time.Hour),
	}).Error)

	exercise := models.Exercise{
		Type:       enums.ExerciseTypeBasicDirect,
		Status:     enums.ExerciseStatusFailed,
		UserID:     exerciseUser.ID,
		FinishedAt: &newer,
	}
	require.NoError(t, db.DB.Create(&exercise).Error)

	// Pending and ignored exercises must not count as usage for this page.
	for _, status := range []enums.ExerciseStatus{enums.ExerciseStatusPending, enums.ExerciseStatusIgnored} {
		finishedAt := newer.Add(48 * time.Hour)
		require.NoError(t, db.DB.Create(&models.Exercise{
			Type: enums.ExerciseTypeBasicDirect, Status: status, UserID: inactiveUser.ID, FinishedAt: &finishedAt,
		}).Error)
	}

	rec := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/users", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body adminUsersBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, int64(4), body.Total)
	require.Len(t, body.Data, 2)
	assert.Equal(t, exerciseUser.ID, body.Data[0].ID)
	assert.Equal(t, "Exercise User", body.Data[0].Name)
	assert.Equal(t, "exercise_user", body.Data[0].Username)
	assert.Zero(t, body.Data[0].VocabularySize)
	require.NotNil(t, body.Data[0].LatestUsage)
	assert.WithinDuration(t, newer, *body.Data[0].LatestUsage, time.Second)
	assert.Nil(t, body.Data[0].DeletedAt)
	assert.Equal(t, vocabularyUser.ID, body.Data[1].ID)
	assert.Equal(t, int64(1), body.Data[1].VocabularySize)
	require.NotNil(t, body.Data[1].LatestUsage)
	assert.WithinDuration(t, older, *body.Data[1].LatestUsage, time.Second)
	assert.Nil(t, body.Data[1].DeletedAt)
}

func TestAdminUsersIncludesDeletedUsersWithoutActivity(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	deletedUser := testkit.CreateUser(t,
		testkit.WithTelegramID(303), testkit.WithName("Deleted User"), testkit.WithUsername("deleted_user"))
	require.NoError(t, db.DB.Delete(&deletedUser).Error)

	rec := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/users", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body adminUsersBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, int64(2), body.Total)
	require.Len(t, body.Data, 1)
	assert.Equal(t, deletedUser.ID, body.Data[0].ID)
	assert.Equal(t, "Deleted User", body.Data[0].Name)
	assert.Equal(t, "deleted_user", body.Data[0].Username)
	assert.Zero(t, body.Data[0].VocabularySize)
	assert.Nil(t, body.Data[0].LatestUsage)
	assert.NotNil(t, body.Data[0].DeletedAt)
}

func TestAdminUsersLimitsResultsToFifty(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	baseTime := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 51; i++ {
		user := testkit.CreateUser(t)
		finishedAt := baseTime.Add(time.Duration(i) * time.Hour)
		require.NoError(t, db.DB.Create(&models.Exercise{
			Type: enums.ExerciseTypeBasicDirect, Status: enums.ExerciseStatusCompleted,
			UserID: user.ID, FinishedAt: &finishedAt,
		}).Error)
	}

	rec := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/users", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body adminUsersBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, int64(52), body.Total)
	require.Len(t, body.Data, 50)
	require.NotNil(t, body.Data[0].LatestUsage)
	require.NotNil(t, body.Data[49].LatestUsage)
	assert.True(t, body.Data[0].LatestUsage.After(*body.Data[49].LatestUsage))
}
