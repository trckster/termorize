package tests

import (
	"errors"
	"net/http"
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

func TestAudioExerciseDirectionsAndBasicScoring(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)

	direct, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeAudioDirect)
	require.NoError(t, err)
	require.NotNil(t, direct.AudioWordID)
	assert.Equal(t, vocabulary.Translation.Original.ID, *direct.AudioWordID)
	assert.Equal(t, "paper", direct.QuestionWord)
	assert.Equal(t, enums.LanguageEn, direct.Language)
	assert.Equal(t, enums.LanguageIt, direct.AnswerLanguage)

	directResult, err := services.VerifyExerciseAnswer(direct.ExerciseID, user.ID, "carta")
	require.NoError(t, err)
	assert.Equal(t, "correct", directResult.Result)
	assert.Equal(t, services.ExerciseBasicCorrectProgressDelta, directResult.ProgressDelta)

	reversed, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeAudioReversed)
	require.NoError(t, err)
	require.NotNil(t, reversed.AudioWordID)
	assert.Equal(t, vocabulary.Translation.Translation.ID, *reversed.AudioWordID)
	assert.Equal(t, "carta", reversed.QuestionWord)
	assert.Equal(t, enums.LanguageIt, reversed.Language)
	assert.Equal(t, enums.LanguageEn, reversed.AnswerLanguage)

	reversedResult, err := services.VerifyExerciseAnswer(reversed.ExerciseID, user.ID, "papre")
	require.NoError(t, err)
	assert.Equal(t, "almost", reversedResult.Result)
	assert.Equal(t, services.ExerciseBasicAlmostProgressDelta, reversedResult.ProgressDelta)
}

func TestIgnoredAudioLanguageFiltersOnlyItsSpokenDirection(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
		IgnoredAudioLanguages: []enums.Language{enums.LanguageEn},
	}))
	exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)

	_, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeAudioDirect)
	assert.ErrorIs(t, err, services.ErrNoVocabularyForExercise)

	reversed, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeAudioReversed)
	require.NoError(t, err)
	assert.Equal(t, enums.ExerciseTypeAudioReversed, reversed.Type)
}

func TestRandomExerciseExclusionAlwaysCreatesNonAudioExercise(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)

	for range 20 {
		rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/random", map[string]any{
			"exclude_audio": true,
		})
		testkit.RequireStatus(t, rec, http.StatusOK)
		var exercise struct {
			Type        enums.ExerciseType `json:"type"`
			AudioWordID *uuid.UUID         `json:"audio_word_id"`
		}
		testkit.DecodeJSON(t, rec, &exercise)
		assert.NotEqual(t, enums.ExerciseTypeAudioDirect, exercise.Type)
		assert.NotEqual(t, enums.ExerciseTypeAudioReversed, exercise.Type)
		assert.Nil(t, exercise.AudioWordID)
	}
}

func TestRandomExerciseRejectsMalformedOptions(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)

	rec := testkit.AuthedRawRequest(t, user, http.MethodPost, "/api/exercises/random", "{", nil)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)
}

func TestIgnoreAudioLanguageSoftDeletesWithoutScoring(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeAudioDirect, enums.ExerciseStatusInProgress, vocabulary.ID)
	beforeProgress := exerciseReloadVocabulary(t, vocabulary.ID).Progress

	updatedUser, err := services.IgnoreAudioLanguageForExercise(exercise.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, []enums.Language{enums.LanguageEn}, updatedUser.Settings.IgnoredAudioLanguages)

	var scoped models.Exercise
	err = db.DB.Where("id = ?", exercise.ID).Take(&scoped).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var deleted models.Exercise
	require.NoError(t, db.DB.Unscoped().Where("id = ?", exercise.ID).Take(&deleted).Error)
	assert.True(t, deleted.DeletedAt.Valid)
	assert.Equal(t, enums.ExerciseStatusInProgress, deleted.Status)
	assert.Nil(t, deleted.FinishedAt)
	require.NotNil(t, exercise.StartedAt)
	require.NotNil(t, deleted.StartedAt)
	assert.WithinDuration(t, *exercise.StartedAt, *deleted.StartedAt, time.Microsecond)

	link := exerciseLink(t, exercise.ID, vocabulary.ID)
	assert.Nil(t, link.Result)
	assert.Nil(t, link.ResultReason)
	assert.Nil(t, link.ProgressDelta)
	assert.Nil(t, link.KnowledgeAfter)
	assert.Nil(t, link.AnsweredAt)
	assert.Equal(t, beforeProgress, exerciseReloadVocabulary(t, vocabulary.ID).Progress)

	byID, err := services.GetExercisesByIDs(user.ID, []uuid.UUID{exercise.ID})
	require.NoError(t, err)
	assert.Empty(t, byID)
	statistics, err := services.GetExerciseStatistics(user.ID)
	require.NoError(t, err)
	assert.Zero(t, statistics.InProgress)
}

func TestPendingExerciseDeleteUsesSoftDeleteAndKeepsVocabularyLink(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	scheduledFor := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	exercise := models.Exercise{
		Type:         enums.ExerciseTypeBasicDirect,
		Status:       enums.ExerciseStatusPending,
		UserID:       user.ID,
		ScheduledFor: &scheduledFor,
	}
	require.NoError(t, db.DB.Create(&exercise).Error)
	require.NoError(t, db.DB.Create(&models.ExerciseVocabulary{
		ExerciseID: exercise.ID, VocabularyID: vocabulary.ID, IsCorrect: true,
	}).Error)

	require.NoError(t, db.DB.Transaction(func(tx *gorm.DB) error {
		return services.DeletePendingExercisesByUserID(tx, user.ID)
	}))

	var deleted models.Exercise
	require.NoError(t, db.DB.Unscoped().Where("id = ?", exercise.ID).Take(&deleted).Error)
	assert.True(t, deleted.DeletedAt.Valid)
	assert.Equal(t, enums.ExerciseStatusPending, deleted.Status)
	require.NotNil(t, deleted.ScheduledFor)
	assert.WithinDuration(t, scheduledFor, *deleted.ScheduledFor, time.Microsecond)
	assert.NotZero(t, exerciseLink(t, exercise.ID, vocabulary.ID).ID)
}

func TestDisablingDailyExercisesSoftDeletesPendingExercises(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
		Telegram: models.UserTelegramSettings{
			DailyQuestionsEnabled:  true,
			DailyQuestionsCount:    2,
			DailyQuestionsSchedule: []models.UserTelegramQuestionsScheduleItem{{From: "10:00", To: "12:00"}},
		},
	}))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	scheduledFor := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	exercise := models.Exercise{
		Type:         enums.ExerciseTypeAudioDirect,
		Status:       enums.ExerciseStatusPending,
		UserID:       user.ID,
		ScheduledFor: &scheduledFor,
	}
	require.NoError(t, db.DB.Create(&exercise).Error)
	require.NoError(t, db.DB.Create(&models.ExerciseVocabulary{
		ExerciseID: exercise.ID, VocabularyID: vocabulary.ID, IsCorrect: true,
	}).Error)

	payload := authSettingsValidPayload()
	payload["telegram"].(map[string]any)["daily_questions_enabled"] = false
	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var deleted models.Exercise
	require.NoError(t, db.DB.Unscoped().Where("id = ?", exercise.ID).Take(&deleted).Error)
	assert.True(t, deleted.DeletedAt.Valid)
	assert.Equal(t, enums.ExerciseStatusPending, deleted.Status)
	require.NotNil(t, deleted.ScheduledFor)
	assert.WithinDuration(t, scheduledFor, *deleted.ScheduledFor, time.Microsecond)
	assert.NotZero(t, exerciseLink(t, exercise.ID, vocabulary.ID).ID)
}

func TestIgnoringAudioLanguageReplacesQueuedAudioAtSameTime(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	currentVocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	replacementVocabulary := exerciseSeedVocabulary(t, user.ID, "chair", "sedia", enums.LanguageEn, enums.LanguageIt)
	current := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeAudioDirect, enums.ExerciseStatusInProgress, currentVocabulary.ID)

	scheduledFor := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	queued := models.Exercise{
		Type:         enums.ExerciseTypeAudioDirect,
		Status:       enums.ExerciseStatusPending,
		UserID:       user.ID,
		ScheduledFor: &scheduledFor,
	}
	require.NoError(t, db.DB.Create(&queued).Error)
	require.NoError(t, db.DB.Create(&models.ExerciseVocabulary{
		ExerciseID: queued.ID, VocabularyID: replacementVocabulary.ID, IsCorrect: true,
	}).Error)

	_, err := services.IgnoreAudioLanguageForExercise(current.ID, user.ID)
	require.NoError(t, err)

	var deletedQueued models.Exercise
	require.NoError(t, db.DB.Unscoped().Where("id = ?", queued.ID).Take(&deletedQueued).Error)
	assert.True(t, deletedQueued.DeletedAt.Valid)
	assert.Equal(t, enums.ExerciseStatusPending, deletedQueued.Status)

	var activePending []models.Exercise
	require.NoError(t, db.DB.
		Where("user_id = ? AND status = ?", user.ID, enums.ExerciseStatusPending).
		Find(&activePending).Error)
	require.Len(t, activePending, 1)
	require.NotNil(t, activePending[0].ScheduledFor)
	assert.WithinDuration(t, scheduledFor, *activePending[0].ScheduledFor, time.Microsecond)
	assert.NotEqual(t, enums.ExerciseTypeAudioDirect, activePending[0].Type)
}

func TestFailedAudioPreparationCreatesNonAudioReplacementAtSameTime(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	scheduledFor := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	exercise := models.Exercise{
		Type:         enums.ExerciseTypeAudioDirect,
		Status:       enums.ExerciseStatusPending,
		UserID:       user.ID,
		ScheduledFor: &scheduledFor,
	}
	require.NoError(t, db.DB.Create(&exercise).Error)
	require.NoError(t, db.DB.Create(&models.ExerciseVocabulary{
		ExerciseID: exercise.ID, VocabularyID: vocabulary.ID, IsCorrect: true,
	}).Error)

	replaced, err := services.ReplacePendingAudioExercise(exercise.ID, true)
	require.NoError(t, err)
	assert.True(t, replaced)

	var deleted models.Exercise
	require.NoError(t, db.DB.Unscoped().Where("id = ?", exercise.ID).Take(&deleted).Error)
	assert.True(t, deleted.DeletedAt.Valid)
	assert.Equal(t, enums.ExerciseStatusPending, deleted.Status)

	var replacement models.Exercise
	require.NoError(t, db.DB.Where("user_id = ? AND status = ?", user.ID, enums.ExerciseStatusPending).Take(&replacement).Error)
	assert.NotEqual(t, enums.ExerciseTypeAudioDirect, replacement.Type)
	assert.NotEqual(t, enums.ExerciseTypeAudioReversed, replacement.Type)
	require.NotNil(t, replacement.ScheduledFor)
	assert.WithinDuration(t, scheduledFor, *replacement.ScheduledFor, time.Microsecond)
}

func TestIgnoredAudioSettingsAreValidatedCanonicalAndRemovable(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	payload := authSettingsValidPayload()
	payload["ignored_audio_languages"] = []string{"tr", "en", "tr"}

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	testkit.RequireStatus(t, rec, http.StatusOK)
	var updated models.User
	testkit.DecodeJSON(t, rec, &updated)
	assert.Equal(t, []enums.Language{enums.LanguageEn, enums.LanguageTr}, updated.Settings.IgnoredAudioLanguages)

	for range 2 {
		rec = testkit.AuthedRequest(t, user, http.MethodDelete, "/api/settings/ignored-audio-languages/en", nil)
		testkit.RequireStatus(t, rec, http.StatusOK)
		testkit.DecodeJSON(t, rec, &updated)
		assert.Equal(t, []enums.Language{enums.LanguageTr}, updated.Settings.IgnoredAudioLanguages)
	}

	payload = authSettingsValidPayload()
	payload["ignored_audio_languages"] = []string{"xx"}
	rec = testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)
}

func TestExerciseSoftDeleteMigrationCreatesActiveScheduleIndex(t *testing.T) {
	var definition string
	err := db.DB.Raw(`
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = 'public' AND indexname = 'index_exercises_active_status_scheduled_for'
	`).Scan(&definition).Error
	require.NoError(t, err)
	require.NotEmpty(t, definition)
	assert.Contains(t, definition, "status")
	assert.Contains(t, definition, "scheduled_for")
	assert.Contains(t, definition, "deleted_at IS NULL")
}

func TestAudioCancellationEndpointRejectsNonAudioExercise(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocabulary.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+exercise.ID.String()+"/ignore-audio-language", nil)
	testkit.RequireStatus(t, rec, http.StatusConflict)

	var active models.Exercise
	require.NoError(t, db.DB.Where("id = ?", exercise.ID).Take(&active).Error)
	assert.False(t, active.DeletedAt.Valid)
}

func TestAudioCancellationEndpointRejectsScoredExercise(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeAudioDirect, enums.ExerciseStatusCompleted, vocabulary.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+exercise.ID.String()+"/ignore-audio-language", nil)
	testkit.RequireStatus(t, rec, http.StatusConflict)

	var active models.Exercise
	require.NoError(t, db.DB.Where("id = ?", exercise.ID).Take(&active).Error)
	assert.False(t, active.DeletedAt.Valid)
	var storedUser models.User
	require.NoError(t, db.DB.Where("id = ?", user.ID).Take(&storedUser).Error)
	assert.Empty(t, storedUser.Settings.IgnoredAudioLanguages)
}

func TestStartingConcurrentlyCancelledAudioKeepsTelegramMessageReference(t *testing.T) {
	testkit.Truncate(t)
	const messageID int64 = 901
	user := testkit.CreateUser(t, testkit.WithTelegramID(700901))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeAudioDirect, enums.ExerciseStatusPending, vocabulary.ID)
	require.NoError(t, db.DB.Delete(&exercise).Error)

	err := services.StartTelegramExercise(exercise.ID, messageID)
	assert.ErrorIs(t, err, services.ErrExerciseNotInProgress)

	var cancelled models.Exercise
	require.NoError(t, db.DB.Unscoped().Where("id = ?", exercise.ID).Take(&cancelled).Error)
	require.NotNil(t, cancelled.TelegramMessageID)
	assert.Equal(t, messageID, *cancelled.TelegramMessageID)
	loaded, err := services.GetExerciseByTelegramMessage(messageID, user.TelegramID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.True(t, loaded.Deleted)
}

func TestSoftDeletedExerciseCanOnlyBeLoadedUnscoped(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeAudioDirect, enums.ExerciseStatusInProgress, vocabulary.ID)
	require.NoError(t, db.DB.Delete(&exercise).Error)

	var scoped models.Exercise
	assert.True(t, errors.Is(db.DB.Take(&scoped, "id = ?", exercise.ID).Error, gorm.ErrRecordNotFound))
	var unscoped models.Exercise
	require.NoError(t, db.DB.Unscoped().Take(&unscoped, "id = ?", exercise.ID).Error)
	assert.True(t, unscoped.DeletedAt.Valid)
}
