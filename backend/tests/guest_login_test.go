package tests

import (
	"database/sql"
	"net/http"
	"regexp"
	"testing"
	"time"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGuestLoginCreatesReadyToPracticeAccount(t *testing.T) {
	testkit.Truncate(t)

	createdAfter := time.Now().UTC()
	rec := testkit.RequestWithHeaders(
		t,
		http.MethodPost,
		"/api/guest/login",
		nil,
		http.Header{
			"Accept-Language": []string{"en-US,en;q=0.9"},
			"X-Timezone":      []string{"Europe/Rome"},
		},
	)
	createdBefore := time.Now().UTC()

	testkit.RequireStatus(t, rec, http.StatusCreated)

	authCookie := telegramLoginAuthCookie(rec.Result())
	require.NotNil(t, authCookie)
	assert.NotEmpty(t, authCookie.Value)
	assert.True(t, authCookie.HttpOnly)
	assert.Equal(t, 7*24*60*60, authCookie.MaxAge)

	var user models.User
	testkit.DecodeJSON(t, rec, &user)
	assert.Regexp(t, regexp.MustCompile(`^absolutely_random_user_\d{6}$`), user.Username)
	assert.Regexp(t, regexp.MustCompile(`^[A-Z][a-z]+ [A-Z][a-z]+$`), user.Name)
	require.NotNil(t, user.GuestExpiresAt)
	assert.False(t, user.GuestExpiresAt.Before(createdAfter.Add(services.GuestAccountLifetime)))
	assert.False(t, user.GuestExpiresAt.After(createdBefore.Add(services.GuestAccountLifetime)))
	assert.Equal(t, "Europe/Rome", user.Settings.TimeZone)
	assert.Equal(t, enums.LanguageEn, user.Settings.SystemLanguage)
	assert.False(t, user.Settings.Telegram.BotEnabled)
	assert.False(t, user.Settings.Telegram.DailyQuestionsEnabled)

	var telegramID sql.NullInt64
	require.NoError(t, db.DB.Raw("SELECT telegram_id FROM users WHERE id = ?", user.ID).Scan(&telegramID).Error)
	assert.False(t, telegramID.Valid)

	var vocabulary []models.Vocabulary
	require.NoError(t, db.DB.
		Where("user_id = ?", user.ID).
		Preload("Translation").
		Find(&vocabulary).Error)
	require.Len(t, vocabulary, 50)
	knowledgeCounts := map[string]int{}
	for _, item := range vocabulary {
		knowledgeCounts[guestKnowledgeBand(t, item.Progress)]++
		require.NotNil(t, item.Translation)
		assert.Equal(t, enums.TranslationSourceDictionary, item.Translation.Source)
		assert.Nil(t, item.Translation.UserID)
	}
	assert.Equal(t, 15, knowledgeCounts["easy"])
	assert.Equal(t, 20, knowledgeCounts["medium"])
	assert.Equal(t, 15, knowledgeCounts["hard"])

	firstPage, err := services.GetVocabulary(user.ID, 1, 20, "")
	require.NoError(t, err)
	require.Len(t, firstPage.Data, 20)
	firstPageKnowledgeCounts := map[string]int{}
	for _, item := range firstPage.Data {
		firstPageKnowledgeCounts[guestKnowledgeBand(t, item.Progress)]++
	}
	assert.Positive(t, firstPageKnowledgeCounts["easy"])
	assert.Positive(t, firstPageKnowledgeCounts["medium"])
	assert.Positive(t, firstPageKnowledgeCounts["hard"])

	meRec := testkit.Request(t, http.MethodGet, "/api/me", nil, authCookie)
	testkit.RequireStatus(t, meRec, http.StatusOK)
	var me models.User
	testkit.DecodeJSON(t, meRec, &me)
	assert.Equal(t, user.ID, me.ID)
	require.NotNil(t, me.GuestExpiresAt)
	assert.WithinDuration(t, *user.GuestExpiresAt, *me.GuestExpiresAt, time.Microsecond)

	exerciseRec := testkit.Request(
		t,
		http.MethodPost,
		"/api/exercises/random",
		map[string]any{"exclude_audio": true},
		authCookie,
	)
	testkit.RequireStatus(t, exerciseRec, http.StatusOK)
	var exercise struct {
		ExerciseID uuid.UUID `json:"exercise_id"`
	}
	testkit.DecodeJSON(t, exerciseRec, &exercise)
	assert.NotEqual(t, uuid.Nil, exercise.ExerciseID)
}

func TestGuestLoginUsesRussianBrowserLanguage(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.RequestWithHeaders(
		t,
		http.MethodPost,
		"/api/guest/login",
		nil,
		http.Header{"Accept-Language": []string{"ru-RU,ru;q=0.9,en;q=0.8"}},
	)
	testkit.RequireStatus(t, rec, http.StatusCreated)

	var user models.User
	testkit.DecodeJSON(t, rec, &user)
	assert.Equal(t, enums.LanguageRu, user.Settings.SystemLanguage)
	assert.Equal(t, "UTC", user.Settings.TimeZone)
}

func TestExpiredGuestSessionCannotAccessProtectedData(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/guest/login", nil)
	testkit.RequireStatus(t, rec, http.StatusCreated)
	authCookie := telegramLoginAuthCookie(rec.Result())
	require.NotNil(t, authCookie)

	var guest models.User
	testkit.DecodeJSON(t, rec, &guest)
	now := time.Now().UTC()
	require.NoError(t, db.DB.Model(&guest).Update("guest_expires_at", now.Add(-time.Minute)).Error)

	vocabularyRec := testkit.Request(
		t,
		http.MethodGet,
		"/api/vocabulary?page=1&page_size=20",
		nil,
		authCookie,
	)
	testkit.RequireStatus(t, vocabularyRec, http.StatusUnauthorized)

	deleted, err := services.DeleteExpiredGuestUsers(now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	vocabularyRec = testkit.Request(
		t,
		http.MethodGet,
		"/api/vocabulary?page=1&page_size=20",
		nil,
		authCookie,
	)
	testkit.RequireStatus(t, vocabularyRec, http.StatusUnauthorized)

	var preservedVocabularyCount int64
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).
		Where("user_id = ?", guest.ID).
		Count(&preservedVocabularyCount).Error)
	assert.Equal(t, int64(50), preservedVocabularyCount)
}

func TestGuestLoginReusesSeedTranslations(t *testing.T) {
	testkit.Truncate(t)

	for range 2 {
		rec := testkit.Request(t, http.MethodPost, "/api/guest/login", nil)
		testkit.RequireStatus(t, rec, http.StatusCreated)
	}

	var userCount, vocabularyCount, translationCount, wordCount int64
	require.NoError(t, db.DB.Model(&models.User{}).Count(&userCount).Error)
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).Count(&vocabularyCount).Error)
	require.NoError(t, db.DB.Model(&models.Translation{}).Count(&translationCount).Error)
	require.NoError(t, db.DB.Model(&models.Word{}).Count(&wordCount).Error)
	assert.Equal(t, int64(2), userCount)
	assert.Equal(t, int64(100), vocabularyCount)
	assert.Equal(t, int64(50), translationCount)
	assert.Equal(t, int64(100), wordCount)
}

func TestDeleteExpiredGuestUsersSoftDeletesOnlyExpiredGuests(t *testing.T) {
	testkit.Truncate(t)

	now := time.Date(2026, time.August, 24, 10, 0, 0, 0, time.UTC)
	expiredGuest, err := services.CreateGuestUser("UTC", enums.LanguageEn)
	require.NoError(t, err)
	activeGuest, err := services.CreateGuestUser("UTC", enums.LanguageEn)
	require.NoError(t, err)
	telegramUser := testkit.CreateUser(t, testkit.WithAdmin())

	exercise := models.Exercise{
		Type:   enums.ExerciseTypeBasicDirect,
		Status: enums.ExerciseStatusInProgress,
		UserID: expiredGuest.ID,
	}
	require.NoError(t, db.DB.Create(&exercise).Error)
	expiredGuestID := expiredGuest.ID
	collection := models.Collection{
		Title:       "Guest collection",
		OwnerID:     &expiredGuestID,
		IsPublished: true,
	}
	require.NoError(t, db.DB.Create(&collection).Error)

	require.NoError(t, db.DB.Model(expiredGuest).Update("guest_expires_at", now.Add(-time.Minute)).Error)
	require.NoError(t, db.DB.Model(activeGuest).Update("guest_expires_at", now.Add(time.Minute)).Error)

	deleted, err := services.DeleteExpiredGuestUsers(now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	assert.ErrorIs(t, db.DB.First(&models.User{}, expiredGuest.ID).Error, gorm.ErrRecordNotFound)
	require.NoError(t, db.DB.First(&models.User{}, activeGuest.ID).Error)
	require.NoError(t, db.DB.First(&models.User{}, telegramUser.ID).Error)

	var preservedGuest models.User
	require.NoError(t, db.DB.Unscoped().First(&preservedGuest, expiredGuest.ID).Error)
	assert.True(t, preservedGuest.DeletedAt.Valid)
	assert.False(t, activeGuest.DeletedAt.Valid)

	var expiredVocabularyCount, activeVocabularyCount int64
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).
		Where("user_id = ?", expiredGuest.ID).
		Count(&expiredVocabularyCount).Error)
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).
		Where("user_id = ?", activeGuest.ID).
		Count(&activeVocabularyCount).Error)
	assert.Equal(t, int64(50), expiredVocabularyCount)
	assert.Equal(t, int64(50), activeVocabularyCount)
	require.NoError(t, db.DB.First(&models.Exercise{}, "id = ?", exercise.ID).Error)
	require.NoError(t, db.DB.First(&models.Collection{}, "id = ?", collection.ID).Error)
	collections, err := services.ListCollections(telegramUser.ID, 1, 20, "", nil)
	require.NoError(t, err)
	require.Len(t, collections.Data, 1)
	assert.Equal(t, collection.ID, collections.Data[0].ID)
	assert.Empty(t, collections.Data[0].OwnerUsername)

	recentUsers, err := services.GetRecentUsersForAdmin(telegramUser.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), recentUsers.Total)
	for _, recentUser := range recentUsers.Data {
		assert.NotEqual(t, expiredGuest.ID, recentUser.ID)
	}
}

func guestKnowledgeBand(t *testing.T, progress models.ProgressEntries) string {
	t.Helper()
	require.Len(t, progress, 1)
	require.Equal(t, enums.KnowledgeTypeTranslation, progress[0].Type)

	knowledge := progress[0].Knowledge
	switch {
	case knowledge >= 70 && knowledge <= 90:
		return "easy"
	case knowledge >= 30 && knowledge <= 60:
		return "medium"
	case knowledge >= 0 && knowledge <= 10:
		return "hard"
	default:
		t.Fatalf("unexpected guest vocabulary knowledge: %d", knowledge)
		return ""
	}
}
