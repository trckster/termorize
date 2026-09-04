package tests

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescriptionExerciseUsesTranslatedWordAndCachesClue(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
		MainLearningLanguage: enums.LanguageEn,
	}))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)

	calls := 0
	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
		GenerateDescriptionFunc: func(word, wordLanguage, descriptionLanguage string) (*openrouter.GeneratedDescription, error) {
			calls++
			assert.Equal(t, "carta", word)
			assert.Equal(t, "Italian", wordLanguage)
			assert.Equal(t, "English", descriptionLanguage)
			return &openrouter.GeneratedDescription{Description: "A thin material used for writing or printing."}, nil
		},
	})

	first, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeDescriptionReversed)
	require.NoError(t, err)
	assert.Equal(t, enums.ExerciseTypeDescriptionReversed, first.Type)
	assert.Empty(t, first.QuestionWord)
	assert.Equal(t, "A thin material used for writing or printing.", first.Description)
	assert.Equal(t, enums.LanguageEn, first.Language)
	assert.Equal(t, enums.LanguageEn, first.AnswerLanguage)
	assert.Nil(t, first.AudioWordID)

	second, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeDescriptionReversed)
	require.NoError(t, err)
	assert.Equal(t, first.Description, second.Description)
	assert.Equal(t, 1, calls)

	var descriptions []models.TranslationDescription
	require.NoError(t, db.DB.Where("translation_id = ?", vocabulary.Translation.ID).Find(&descriptions).Error)
	require.Len(t, descriptions, 1)
	assert.Equal(t, config.GetOpenRouterModel(), descriptions[0].Model)
	assert.Equal(t, first.Description, descriptions[0].Description)

	result, err := services.VerifyExerciseAnswer(first.ExerciseID, user.ID, "papre")
	require.NoError(t, err)
	assert.Equal(t, services.ExerciseVocabularyResultAlmost, result.Result)
	assert.Equal(t, "paper", result.CorrectAnswer)
	assert.Equal(t, services.ExerciseBasicAlmostProgressDelta, result.ProgressDelta)
}

func TestIgnoringDescriptionLanguageReplacesQueuedExerciseAtSameTime(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
		MainLearningLanguage: enums.LanguageEn,
		Telegram: models.UserTelegramSettings{
			DailyQuestionsEnabled:  true,
			DailyQuestionsCount:    2,
			DailyQuestionsSchedule: []models.UserTelegramQuestionsScheduleItem{{From: "10:00", To: "12:00"}},
		},
	}))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	exerciseSeedVocabulary(t, user.ID, "chair", "sedia", enums.LanguageEn, enums.LanguageIt)
	scheduledFor := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	queued := models.Exercise{
		Type:         enums.ExerciseTypeDescriptionReversed,
		Status:       enums.ExerciseStatusPending,
		UserID:       user.ID,
		ScheduledFor: &scheduledFor,
	}
	require.NoError(t, db.DB.Create(&queued).Error)
	require.NoError(t, db.DB.Create(&models.ExerciseVocabulary{
		ExerciseID: queued.ID, VocabularyID: vocabulary.ID, IsCorrect: true,
	}).Error)

	payload := authSettingsValidPayload()
	payload["main_learning_language"] = "en"
	payload["ignored_description_languages"] = []string{"en", "en"}
	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	testkit.RequireStatus(t, rec, http.StatusOK)
	var updated models.User
	testkit.DecodeJSON(t, rec, &updated)
	assert.Equal(t, []enums.Language{enums.LanguageEn}, updated.Settings.IgnoredDescriptionLanguages)

	var deleted models.Exercise
	require.NoError(t, db.DB.Unscoped().Where("id = ?", queued.ID).Take(&deleted).Error)
	assert.True(t, deleted.DeletedAt.Valid)

	var replacement models.Exercise
	require.NoError(t, db.DB.Where("user_id = ? AND status = ?", user.ID, enums.ExerciseStatusPending).Take(&replacement).Error)
	assert.NotEqual(t, enums.ExerciseTypeDescriptionReversed, replacement.Type)
	require.NotNil(t, replacement.ScheduledFor)
	assert.WithinDuration(t, scheduledFor, *replacement.ScheduledFor, time.Microsecond)
}

func TestAllowingDescriptionLanguageKeepsQueuedExercise(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
		MainLearningLanguage:        enums.LanguageEn,
		IgnoredDescriptionLanguages: []enums.Language{enums.LanguageEn},
	}))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	queued := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeDescriptionReversed, enums.ExerciseStatusPending, vocabulary.ID)
	scheduledFor := time.Now().UTC().Add(time.Hour)
	require.NoError(t, db.DB.Model(&queued).Updates(map[string]any{
		"scheduled_for": scheduledFor,
		"started_at":    nil,
	}).Error)

	payload := authSettingsValidPayload()
	payload["main_learning_language"] = "en"
	payload["ignored_description_languages"] = []string{}
	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var active models.Exercise
	require.NoError(t, db.DB.Where("id = ?", queued.ID).Take(&active).Error)
	assert.False(t, active.DeletedAt.Valid)
}

func TestIgnoredDescriptionLanguageValidationRejectsUnsupportedCode(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	payload := authSettingsValidPayload()
	payload["ignored_description_languages"] = []string{"xx"}

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var unchanged models.User
	require.NoError(t, db.DB.Where("id = ?", user.ID).Take(&unchanged).Error)
	assert.Empty(t, unchanged.Settings.IgnoredDescriptionLanguages)
}

func TestDescriptionExerciseOnlyUsesEligibleLearningLanguage(t *testing.T) {
	tests := []struct {
		name     string
		settings models.UserSettings
	}{
		{
			name:     "different main learning language",
			settings: models.UserSettings{MainLearningLanguage: enums.LanguageDe},
		},
		{
			name: "ignored description language",
			settings: models.UserSettings{
				MainLearningLanguage:        enums.LanguageEn,
				IgnoredDescriptionLanguages: []enums.Language{enums.LanguageEn},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testkit.Truncate(t)
			user := testkit.CreateUser(t, testkit.WithSettings(test.settings))
			exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)

			_, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeDescriptionReversed)
			assert.ErrorIs(t, err, services.ErrNoVocabularyForExercise)
		})
	}
}

func TestDescriptionExerciseRejectsClueContainingAnswer(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{MainLearningLanguage: enums.LanguageEn}))
	exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
		GenerateDescriptionFunc: func(string, string, string) (*openrouter.GeneratedDescription, error) {
			return &openrouter.GeneratedDescription{Description: "Paper used for writing."}, nil
		},
	})

	_, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeDescriptionReversed)
	assert.ErrorIs(t, err, services.ErrDescriptionGenerationFailed)

	var count int64
	require.NoError(t, db.DB.Model(&models.TranslationDescription{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestDescriptionExercisePropagatesOpenRouterFailureWhenExplicitlyRequested(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{MainLearningLanguage: enums.LanguageEn}))
	exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
		GenerateDescriptionFunc: func(string, string, string) (*openrouter.GeneratedDescription, error) {
			return nil, errors.New("model unavailable")
		},
	})

	_, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeDescriptionReversed)
	assert.ErrorIs(t, err, services.ErrDescriptionGenerationFailed)

	var count int64
	require.NoError(t, db.DB.Model(&models.Exercise{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestDescriptionCacheUniquePerTranslationAndModel(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	description := models.TranslationDescription{
		TranslationID: vocabulary.Translation.ID,
		Model:         config.GetOpenRouterModel(),
		Description:   "First clue.",
	}
	require.NoError(t, db.DB.Create(&description).Error)

	duplicate := models.TranslationDescription{
		TranslationID: vocabulary.Translation.ID,
		Model:         config.GetOpenRouterModel(),
		Description:   "Second clue.",
	}
	assert.Error(t, db.DB.Create(&duplicate).Error)
}

func TestStartingConcurrentlyCancelledDescriptionKeepsTelegramMessageReference(t *testing.T) {
	testkit.Truncate(t)
	const messageID int64 = 902
	user := testkit.CreateUser(t, testkit.WithTelegramID(700902))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeDescriptionReversed, enums.ExerciseStatusPending, vocabulary.ID)

	pending, err := services.IsPendingDescriptionExercise(exercise.ID)
	require.NoError(t, err)
	assert.True(t, pending)
	require.NoError(t, db.DB.Delete(&exercise).Error)
	pending, err = services.IsPendingDescriptionExercise(exercise.ID)
	require.NoError(t, err)
	assert.False(t, pending)

	err = services.StartTelegramExercise(exercise.ID, messageID)
	assert.ErrorIs(t, err, services.ErrExerciseNotInProgress)

	var cancelled models.Exercise
	require.NoError(t, db.DB.Unscoped().Where("id = ?", exercise.ID).Take(&cancelled).Error)
	require.NotNil(t, cancelled.TelegramMessageID)
	assert.Equal(t, messageID, *cancelled.TelegramMessageID)
	loaded, err := services.GetExerciseByTelegramMessage(messageID, user.TelegramID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.True(t, loaded.Deleted)
	assert.Equal(t, enums.ExerciseTypeDescriptionReversed, loaded.ExerciseType)
}
