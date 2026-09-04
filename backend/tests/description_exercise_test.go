package tests

import (
	"errors"
	"net/http"
	"strings"
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

func TestDescriptionExerciseDirectionsUseAndCacheWordDefinitions(t *testing.T) {
	tests := []struct {
		name                string
		exerciseType        enums.ExerciseType
		mainLanguage        enums.Language
		expectedWord        string
		expectedLanguage    enums.Language
		expectedAnswer      string
		expectedAnswerLang  enums.Language
		description         string
		almostCorrectAnswer string
	}{
		{
			name:                "direct",
			exerciseType:        enums.ExerciseTypeDescriptionDirect,
			mainLanguage:        enums.LanguageEn,
			expectedWord:        "paper",
			expectedLanguage:    enums.LanguageEn,
			expectedAnswer:      "paper",
			expectedAnswerLang:  enums.LanguageEn,
			description:         "A thin material used for writing or printing.",
			almostCorrectAnswer: "papre",
		},
		{
			name:                "reversed",
			exerciseType:        enums.ExerciseTypeDescriptionReversed,
			mainLanguage:        enums.LanguageIt,
			expectedWord:        "carta",
			expectedLanguage:    enums.LanguageIt,
			expectedAnswer:      "carta",
			expectedAnswerLang:  enums.LanguageIt,
			description:         "Un materiale sottile usato per scrivere o stampare.",
			almostCorrectAnswer: "crata",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testkit.Truncate(t)
			user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
				MainLearningLanguage: test.mainLanguage,
			}))
			vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
			expectedWordID := vocabulary.Translation.Original.ID
			if test.exerciseType == enums.ExerciseTypeDescriptionReversed {
				expectedWordID = vocabulary.Translation.Translation.ID
			}

			calls := 0
			testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
				GenerateDescriptionFunc: func(word, wordLanguage, descriptionLanguage string) (*openrouter.GeneratedDescription, error) {
					calls++
					assert.Equal(t, test.expectedWord, word)
					assert.Equal(t, test.expectedLanguage.DisplayName(), wordLanguage)
					assert.Equal(t, test.expectedLanguage.DisplayName(), descriptionLanguage)
					return &openrouter.GeneratedDescription{Description: test.description}, nil
				},
			})

			first, err := services.CreateRandomExerciseOfTypes(user.ID, test.exerciseType)
			require.NoError(t, err)
			assert.Equal(t, test.exerciseType, first.Type)
			assert.Empty(t, first.QuestionWord)
			assert.Equal(t, test.description, first.Description)
			assert.Equal(t, test.expectedLanguage, first.Language)
			assert.Equal(t, test.expectedAnswerLang, first.AnswerLanguage)
			assert.Nil(t, first.AudioWordID)

			second, err := services.CreateRandomExerciseOfTypes(user.ID, test.exerciseType)
			require.NoError(t, err)
			assert.Equal(t, first.Description, second.Description)
			assert.Equal(t, 1, calls)

			var descriptions []models.WordDescription
			require.NoError(t, db.DB.Where("word_id = ?", expectedWordID).Find(&descriptions).Error)
			require.Len(t, descriptions, 1)
			assert.Equal(t, config.GetOpenRouterModel(), descriptions[0].Model)
			assert.Equal(t, first.Description, descriptions[0].Description)

			result, err := services.VerifyExerciseAnswer(first.ExerciseID, user.ID, test.almostCorrectAnswer)
			require.NoError(t, err)
			assert.Equal(t, services.ExerciseVocabularyResultAlmost, result.Result)
			assert.Equal(t, test.expectedAnswer, result.CorrectAnswer)
			assert.Equal(t, services.ExerciseBasicAlmostProgressDelta, result.ProgressDelta)
		})
	}
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
		Type:         enums.ExerciseTypeDescriptionDirect,
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
	assert.False(t, replacement.Type == enums.ExerciseTypeDescriptionDirect || replacement.Type == enums.ExerciseTypeDescriptionReversed)
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
	queued := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeDescriptionDirect, enums.ExerciseStatusPending, vocabulary.ID)
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

func TestIgnoringUnrelatedDescriptionLanguageKeepsQueuedExercise(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
		MainLearningLanguage: enums.LanguageEn,
	}))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	queued := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeDescriptionDirect, enums.ExerciseStatusPending, vocabulary.ID)
	scheduledFor := time.Now().UTC().Add(time.Hour)
	require.NoError(t, db.DB.Model(&queued).Updates(map[string]any{
		"scheduled_for": scheduledFor,
		"started_at":    nil,
	}).Error)

	payload := authSettingsValidPayload()
	payload["main_learning_language"] = "en"
	payload["ignored_description_languages"] = []string{"de"}
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

			_, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeDescriptionDirect)
			assert.ErrorIs(t, err, services.ErrNoVocabularyForExercise)
		})
	}
}

func TestDescriptionExerciseRejectsClueContainingDescribedWord(t *testing.T) {
	for _, test := range []struct {
		exerciseType enums.ExerciseType
		mainLanguage enums.Language
		clue         string
	}{
		{enums.ExerciseTypeDescriptionDirect, enums.LanguageEn, "Paper is used for writing."},
		{enums.ExerciseTypeDescriptionReversed, enums.LanguageIt, "La carta viene usata per scrivere."},
	} {
		t.Run(string(test.exerciseType), func(t *testing.T) {
			testkit.Truncate(t)
			user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{MainLearningLanguage: test.mainLanguage}))
			exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
			testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
				GenerateDescriptionFunc: func(string, string, string) (*openrouter.GeneratedDescription, error) {
					return &openrouter.GeneratedDescription{Description: test.clue}, nil
				},
			})

			_, err := services.CreateRandomExerciseOfTypes(user.ID, test.exerciseType)
			assert.ErrorIs(t, err, services.ErrDescriptionGenerationFailed)

			var count int64
			require.NoError(t, db.DB.Model(&models.WordDescription{}).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}

func TestDescriptionExerciseRejectsOversizedClue(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{MainLearningLanguage: enums.LanguageEn}))
	exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
		GenerateDescriptionFunc: func(string, string, string) (*openrouter.GeneratedDescription, error) {
			return &openrouter.GeneratedDescription{Description: strings.Repeat("x", 301)}, nil
		},
	})

	_, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeDescriptionDirect)
	assert.ErrorIs(t, err, services.ErrDescriptionGenerationFailed)

	var count int64
	require.NoError(t, db.DB.Model(&models.WordDescription{}).Count(&count).Error)
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

	_, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeDescriptionDirect)
	assert.ErrorIs(t, err, services.ErrDescriptionGenerationFailed)

	var count int64
	require.NoError(t, db.DB.Model(&models.Exercise{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestDescriptionExerciseRechecksEligibilityAfterGeneration(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{MainLearningLanguage: enums.LanguageEn}))
	exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	generationStarted := make(chan struct{})
	releaseGeneration := make(chan struct{})
	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
		GenerateDescriptionFunc: func(string, string, string) (*openrouter.GeneratedDescription, error) {
			close(generationStarted)
			<-releaseGeneration
			return &openrouter.GeneratedDescription{Description: "A thin material used for writing or printing."}, nil
		},
	})

	type creationResult struct {
		exercise *services.RandomExerciseResult
		err      error
	}
	created := make(chan creationResult, 1)
	go func() {
		exercise, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeDescriptionDirect)
		created <- creationResult{exercise: exercise, err: err}
	}()

	select {
	case <-generationStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("description generation did not start")
	}
	payload := authSettingsValidPayload()
	payload["main_learning_language"] = "en"
	payload["ignored_description_languages"] = []string{"en"}
	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/settings", payload)
	testkit.RequireStatus(t, rec, http.StatusOK)
	close(releaseGeneration)

	select {
	case result := <-created:
		assert.Nil(t, result.exercise)
		assert.ErrorIs(t, result.err, services.ErrNoVocabularyForExercise)
	case <-time.After(5 * time.Second):
		t.Fatal("description creation did not finish")
	}

	var count int64
	require.NoError(t, db.DB.Model(&models.Exercise{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestDescriptionCacheUniquePerWordAndModel(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	description := models.WordDescription{
		WordID:      vocabulary.Translation.Original.ID,
		Model:       config.GetOpenRouterModel(),
		Description: "First clue.",
	}
	require.NoError(t, db.DB.Create(&description).Error)

	duplicate := models.WordDescription{
		WordID:      vocabulary.Translation.Original.ID,
		Model:       config.GetOpenRouterModel(),
		Description: "Second clue.",
	}
	assert.Error(t, db.DB.Create(&duplicate).Error)
}

func TestStartingConcurrentlyCancelledDescriptionKeepsTelegramMessageReference(t *testing.T) {
	testkit.Truncate(t)
	const messageID int64 = 902
	user := testkit.CreateUser(t, testkit.WithTelegramID(700902))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeDescriptionDirect, enums.ExerciseStatusPending, vocabulary.ID)

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
	assert.Equal(t, enums.ExerciseTypeDescriptionDirect, loaded.ExerciseType)
}
