package tests

import (
	"fmt"
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
)

// ---------------------------------------------------------------------------
// Local helpers (unexported, prefixed `exercise` to avoid clashes with other
// test files in the shared `tests` package).
// ---------------------------------------------------------------------------

// exerciseSeedVocabulary inserts a Word pair, a user Translation and a
// Vocabulary row directly via the DB and returns the vocabulary. The progress
// is seeded below 100 so it counts as "eligible" for exercise generation.
func exerciseSeedVocabulary(t *testing.T, userID uint, original, translated string, fromLang, toLang enums.Language) models.Vocabulary {
	t.Helper()

	originalWord := models.Word{Word: original, Language: fromLang}
	require.NoError(t, db.DB.Create(&originalWord).Error)

	translatedWord := models.Word{Word: translated, Language: toLang}
	require.NoError(t, db.DB.Create(&translatedWord).Error)

	uid := userID
	translation := models.Translation{
		OriginalID:    originalWord.ID,
		TranslationID: translatedWord.ID,
		Source:        enums.TranslationSourceUser,
		UserID:        &uid,
	}
	require.NoError(t, db.DB.Create(&translation).Error)

	vocabulary := models.Vocabulary{
		UserID:        userID,
		TranslationID: translation.ID,
		Progress:      models.BuildDefaultProgress(),
	}
	require.NoError(t, db.DB.Create(&vocabulary).Error)

	// Reload with associations so callers can read words back.
	var loaded models.Vocabulary
	require.NoError(t, db.DB.
		Where("id = ?", vocabulary.ID).
		Preload("Translation").
		Preload("Translation.Original").
		Preload("Translation.Translation").
		First(&loaded).Error)

	return loaded
}

// exerciseSeedExercise inserts an Exercise row with the given type/status and a
// single correct vocabulary link (position 0). It returns the created exercise.
func exerciseSeedExercise(t *testing.T, userID uint, exerciseType enums.ExerciseType, status enums.ExerciseStatus, vocabularyID uuid.UUID) models.Exercise {
	t.Helper()

	now := time.Now().UTC()
	exercise := models.Exercise{
		Type:      exerciseType,
		Status:    status,
		UserID:    userID,
		StartedAt: &now,
	}
	require.NoError(t, db.DB.Create(&exercise).Error)

	link := models.ExerciseVocabulary{
		ExerciseID:   exercise.ID,
		VocabularyID: vocabularyID,
		IsCorrect:    true,
		Position:     0,
	}
	require.NoError(t, db.DB.Create(&link).Error)

	return exercise
}

// exerciseSeedChoiceExercise inserts a choice exercise with the required number
// of vocabulary links: one correct (the first) and the remaining distractors.
func exerciseSeedChoiceExercise(t *testing.T, userID uint, exerciseType enums.ExerciseType, status enums.ExerciseStatus, vocabularyIDs []uuid.UUID) models.Exercise {
	t.Helper()
	require.Len(t, vocabularyIDs, services.ChoiceExerciseVocabularyCount, "choice exercise has an invalid option count")

	now := time.Now().UTC()
	exercise := models.Exercise{
		Type:      exerciseType,
		Status:    status,
		UserID:    userID,
		StartedAt: &now,
	}
	require.NoError(t, db.DB.Create(&exercise).Error)

	for index, vocabularyID := range vocabularyIDs {
		link := models.ExerciseVocabulary{
			ExerciseID:   exercise.ID,
			VocabularyID: vocabularyID,
			IsCorrect:    index == 0,
			Position:     index,
		}
		require.NoError(t, db.DB.Create(&link).Error)
	}

	return exercise
}

// exerciseSeedMatchPairsExercise inserts a match/pairs exercise with the
// canonical 5 vocabulary links (all marked correct, as the runner does).
func exerciseSeedMatchPairsExercise(t *testing.T, userID uint, status enums.ExerciseStatus, vocabularyIDs []uuid.UUID) models.Exercise {
	t.Helper()
	require.Len(t, vocabularyIDs, 5, "match pairs exercise needs 5 vocabularies")

	now := time.Now().UTC()
	exercise := models.Exercise{
		Type:      enums.ExerciseTypeMatchPairs,
		Status:    status,
		UserID:    userID,
		StartedAt: &now,
	}
	require.NoError(t, db.DB.Create(&exercise).Error)

	for index, vocabularyID := range vocabularyIDs {
		link := models.ExerciseVocabulary{
			ExerciseID:   exercise.ID,
			VocabularyID: vocabularyID,
			IsCorrect:    true,
			Position:     index,
		}
		require.NoError(t, db.DB.Create(&link).Error)
	}

	return exercise
}

// exerciseReload loads an exercise row by ID.
func exerciseReload(t *testing.T, id uuid.UUID) models.Exercise {
	t.Helper()
	var exercise models.Exercise
	require.NoError(t, db.DB.Where("id = ?", id).First(&exercise).Error)
	return exercise
}

// exerciseReloadVocabulary loads a vocabulary row (including soft-deleted) by ID.
func exerciseReloadVocabulary(t *testing.T, id uuid.UUID) models.Vocabulary {
	t.Helper()
	var vocabulary models.Vocabulary
	require.NoError(t, db.DB.Where("id = ?", id).First(&vocabulary).Error)
	return vocabulary
}

// exerciseTranslationKnowledge returns the translation knowledge in a progress slice.
func exerciseTranslationKnowledge(t *testing.T, progress models.ProgressEntries) int {
	t.Helper()
	for _, entry := range progress {
		if entry.Type == enums.KnowledgeTypeTranslation {
			return entry.Knowledge
		}
	}
	t.Fatalf("no translation progress entry found")
	return 0
}

func exerciseSetTranslationKnowledge(t *testing.T, vocabularyID uuid.UUID, knowledge int) {
	t.Helper()

	var masteredAt *time.Time
	if knowledge >= 100 {
		now := time.Now().UTC()
		masteredAt = &now
	}

	require.NoError(t, db.DB.Model(&models.Vocabulary{}).
		Where("id = ?", vocabularyID).
		Updates(map[string]any{
			"progress": models.ProgressEntries{{
				Knowledge: knowledge,
				Type:      enums.KnowledgeTypeTranslation,
			}},
			"mastered_at": masteredAt,
		}).Error)
}

func exerciseMarkKnownVocabularyRepetition(t *testing.T, exerciseID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.DB.Model(&models.Exercise{}).
		Where("id = ?", exerciseID).
		Update("is_known_vocabulary_repetition", true).Error)
}

// exerciseLink loads the vocabulary_exercises link row for an exercise+vocabulary.
func exerciseLink(t *testing.T, exerciseID, vocabularyID uuid.UUID) models.ExerciseVocabulary {
	t.Helper()
	var link models.ExerciseVocabulary
	require.NoError(t, db.DB.
		Where("exercise_id = ? AND vocabulary_id = ?", exerciseID, vocabularyID).
		First(&link).Error)
	return link
}

// ===========================================================================
// GET /api/exercises (GetExercises)
// ===========================================================================

func TestGetExercisesRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/exercises", nil)
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestGetExercisesEmpty(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body services.ExerciseListResponse
	testkit.DecodeJSON(t, rec, &body)

	assert.Empty(t, body.Data)
	assert.Equal(t, 1, body.Pagination.Page)
	assert.Equal(t, 50, body.Pagination.PageSize) // controller default
	assert.Equal(t, int64(0), body.Pagination.Total)
	assert.Equal(t, 0, body.Pagination.TotalPages)
}

func TestGenerateExercisesUsesWeightedExerciseSelection(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
		SystemLanguage: enums.LanguageEn,
		TimeZone:       "UTC",
		Telegram: models.UserTelegramSettings{
			BotEnabled:             true,
			DailyQuestionsEnabled:  true,
			DailyQuestionsCount:    2,
			DailyQuestionsSchedule: []models.UserTelegramQuestionsScheduleItem{{From: "10:00", To: "10:30"}},
		},
	}))

	for _, pair := range []struct {
		original    string
		translation string
	}{
		{"release", "rilasciare"},
		{"cell", "la cella"},
		{"sentence", "la condanna"},
		{"prison", "la prigione"},
		{"guard", "la guardia"},
	} {
		exerciseSeedVocabulary(t, user.ID, pair.original, pair.translation, enums.LanguageEn, enums.LanguageIt)
	}

	require.Equal(t, 2, services.GenerateExercises(user, time.Date(2026, time.June, 21, 0, 0, 0, 0, time.UTC)))

	var generatedCount int64
	require.NoError(t, db.DB.Model(&models.Exercise{}).
		Where("user_id = ? AND status = ?", user.ID, enums.ExerciseStatusPending).
		Count(&generatedCount).Error)
	assert.EqualValues(t, 2, generatedCount)
}

func TestGenerateExercisesKnownVocabularyRepetitionRules(t *testing.T) {
	testCases := []struct {
		name                string
		dailyQuestionsCount uint
		diceRoll            int
		includeKnownWord    bool
		expectedGenerated   int
		expectedRepetitions int64
	}{
		{
			name:                "winning roll adds repetition",
			dailyQuestionsCount: 3,
			diceRoll:            6,
			includeKnownWord:    true,
			expectedGenerated:   4,
			expectedRepetitions: 1,
		},
		{
			name:                "fewer than three daily exercises disables repetition",
			dailyQuestionsCount: 2,
			diceRoll:            6,
			includeKnownWord:    true,
			expectedGenerated:   2,
			expectedRepetitions: 0,
		},
		{
			name:                "non-winning roll does not add repetition",
			dailyQuestionsCount: 3,
			diceRoll:            5,
			includeKnownWord:    true,
			expectedGenerated:   3,
			expectedRepetitions: 0,
		},
		{
			name:                "winning roll without a known word adds nothing",
			dailyQuestionsCount: 3,
			diceRoll:            6,
			includeKnownWord:    false,
			expectedGenerated:   3,
			expectedRepetitions: 0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testkit.Truncate(t)

			restoreDice := services.SetKnownVocabularyRepetitionDiceRollForTest(testCase.diceRoll)
			defer restoreDice()

			user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
				SystemLanguage: enums.LanguageEn,
				TimeZone:       "UTC",
				Telegram: models.UserTelegramSettings{
					BotEnabled:             true,
					DailyQuestionsEnabled:  true,
					DailyQuestionsCount:    testCase.dailyQuestionsCount,
					DailyQuestionsSchedule: []models.UserTelegramQuestionsScheduleItem{{From: "10:00", To: "10:30"}},
				},
			}))

			for index := 0; index < int(testCase.dailyQuestionsCount); index++ {
				exerciseSeedVocabulary(
					t,
					user.ID,
					fmt.Sprintf("learning-%d", index),
					fmt.Sprintf("imparare-%d", index),
					enums.LanguageEn,
					enums.LanguageIt,
				)
			}

			var knownVocabulary models.Vocabulary
			if testCase.includeKnownWord {
				knownVocabulary = exerciseSeedVocabulary(t, user.ID, "known", "conosciuto", enums.LanguageEn, enums.LanguageIt)
				exerciseSetTranslationKnowledge(t, knownVocabulary.ID, 100)
			}

			generated := services.GenerateExercises(user, time.Date(2026, time.June, 21, 0, 0, 0, 0, time.UTC))
			require.Equal(t, testCase.expectedGenerated, generated)

			var repetitions []models.Exercise
			require.NoError(t, db.DB.
				Where("user_id = ? AND is_known_vocabulary_repetition = ?", user.ID, true).
				Find(&repetitions).Error)
			require.Len(t, repetitions, int(testCase.expectedRepetitions))

			if testCase.expectedRepetitions == 0 {
				return
			}

			repetition := repetitions[0]
			assert.Contains(t, []enums.ExerciseType{
				enums.ExerciseTypeBasicDirect,
				enums.ExerciseTypeBasicReversed,
			}, repetition.Type)
			assert.Equal(t, enums.ExerciseStatusPending, repetition.Status)
			require.NotNil(t, repetition.ScheduledFor)

			link := exerciseLink(t, repetition.ID, knownVocabulary.ID)
			assert.True(t, link.IsCorrect)

			due, err := services.GetDuePendingExercises(time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC))
			require.NoError(t, err)

			var deliveredRepetition *services.PendingExercise
			for index := range due {
				if due[index].ExerciseID == repetition.ID {
					deliveredRepetition = &due[index]
					break
				}
			}
			require.NotNil(t, deliveredRepetition)
			assert.True(t, deliveredRepetition.IsKnownVocabularyRepetition)
		})
	}
}

func TestCreatePendingCharacterExerciseUsesRandomDirection(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
	scheduledFor := time.Now().UTC().Truncate(time.Microsecond)

	result, err := services.CreatePendingCharacterExercise(user.ID, scheduledFor)
	require.NoError(t, err)
	require.Contains(t, []enums.ExerciseType{
		enums.ExerciseTypeCharactersDirect,
		enums.ExerciseTypeCharactersReversed,
	}, result.Type)

	expectedAnswer := "carta"
	if result.Type == enums.ExerciseTypeCharactersReversed {
		assert.Equal(t, "carta", result.QuestionWord)
		assert.Equal(t, enums.LanguageIt, result.Language)
		assert.Equal(t, enums.LanguageEn, result.AnswerLanguage)
		expectedAnswer = "paper"
	} else {
		assert.Equal(t, "paper", result.QuestionWord)
		assert.Equal(t, enums.LanguageEn, result.Language)
		assert.Equal(t, enums.LanguageIt, result.AnswerLanguage)
	}
	assert.ElementsMatch(t, services.AnswerCharacters(expectedAnswer), result.Options)

	stored := exerciseReload(t, result.ExerciseID)
	assert.Equal(t, user.ID, stored.UserID)
	assert.Equal(t, enums.ExerciseStatusPending, stored.Status)
	require.NotNil(t, stored.ScheduledFor)
	assert.WithinDuration(t, scheduledFor, *stored.ScheduledFor, time.Microsecond)

	link := exerciseLink(t, result.ExerciseID, vocabulary.ID)
	assert.True(t, link.IsCorrect)

	answerOptions, err := services.GetExerciseAnswerOptions(result.ExerciseID, result.Type)
	require.NoError(t, err)
	require.Len(t, answerOptions, 1)
	assert.Equal(t, expectedAnswer, answerOptions[0].Label)
}

func TestCreateRandomExerciseOfTypesRestrictsExerciseFamily(t *testing.T) {
	testCases := []struct {
		name            string
		requestedTypes  []enums.ExerciseType
		vocabularyCount int
	}{
		{
			name: "basic",
			requestedTypes: []enums.ExerciseType{
				enums.ExerciseTypeBasicDirect,
				enums.ExerciseTypeBasicReversed,
			},
			vocabularyCount: 1,
		},
		{
			name: "choice",
			requestedTypes: []enums.ExerciseType{
				enums.ExerciseTypeChoiceDirect,
				enums.ExerciseTypeChoiceReversed,
			},
			vocabularyCount: services.ChoiceExerciseVocabularyCount,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testkit.Truncate(t)

			user := testkit.CreateUser(t)
			for i := 0; i < testCase.vocabularyCount; i++ {
				exerciseSeedVocabulary(
					t,
					user.ID,
					fmt.Sprintf("original-%d", i),
					fmt.Sprintf("translated-%d", i),
					enums.LanguageEn,
					enums.LanguageIt,
				)
			}

			result, err := services.CreateRandomExerciseOfTypes(user.ID, testCase.requestedTypes...)
			require.NoError(t, err)
			assert.Contains(t, testCase.requestedTypes, result.Type)
		})
	}
}

func TestIgnoreDuePendingExercisesIgnoresMatchPairsWithPartialDeletedVocabulary(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabularies := []models.Vocabulary{
		exerciseSeedVocabulary(t, user.ID, "release", "rilasciare", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "cell", "la cella", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "sentence", "la condanna", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "prison", "la prigione", enums.LanguageEn, enums.LanguageIt),
		exerciseSeedVocabulary(t, user.ID, "guard", "la guardia", enums.LanguageEn, enums.LanguageIt),
	}

	now := time.Now().UTC()
	exercise := models.Exercise{
		Type:         enums.ExerciseTypeMatchPairs,
		Status:       enums.ExerciseStatusPending,
		UserID:       user.ID,
		ScheduledFor: &now,
	}
	require.NoError(t, db.DB.Create(&exercise).Error)
	for index, vocabulary := range vocabularies {
		link := models.ExerciseVocabulary{
			ExerciseID:   exercise.ID,
			VocabularyID: vocabulary.ID,
			IsCorrect:    true,
			Position:     index,
		}
		require.NoError(t, db.DB.Create(&link).Error)
	}

	deletedAt := now.Add(-time.Minute)
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).
		Where("id = ?", vocabularies[0].ID).
		Update("deleted_at", deletedAt).Error)

	due, err := services.GetDuePendingMatchExercises(now)
	require.NoError(t, err)
	assert.Empty(t, due)

	require.NoError(t, services.IgnoreDuePendingExercisesWithoutActiveVocabulary(now))

	refreshed := exerciseReload(t, exercise.ID)
	assert.Equal(t, enums.ExerciseStatusIgnored, refreshed.Status)
	require.NotNil(t, refreshed.FinishedAt)

	var ignoredLinks int64
	require.NoError(t, db.DB.Model(&models.ExerciseVocabulary{}).
		Where("exercise_id = ?", exercise.ID).
		Where("result = ?", services.ExerciseVocabularyResultIgnored).
		Where("result_reason = ?", services.ExerciseVocabularyResultReasonDeletedVocabulary).
		Count(&ignoredLinks).Error)
	assert.EqualValues(t, services.MatchPairsVocabularyCount, ignoredLinks)
}

func TestGetExercisesReturnsStartedExercises(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body services.ExerciseListResponse
	testkit.DecodeJSON(t, rec, &body)

	require.Len(t, body.Data, 1)
	assert.Equal(t, exercise.ID, body.Data[0].ID)
	assert.Equal(t, enums.ExerciseTypeBasicDirect, body.Data[0].Type)
	assert.Equal(t, enums.ExerciseStatusInProgress, body.Data[0].Status)
	require.Len(t, body.Data[0].Vocabulary, 1)
	assert.Equal(t, vocab.ID, body.Data[0].Vocabulary[0].ID)
	assert.Equal(t, int64(1), body.Pagination.Total)
}

// Exercises that have never been started (started_at IS NULL) are excluded.
func TestGetExercisesExcludesNotStarted(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)

	// pending exercise with no started_at
	scheduled := time.Now().UTC()
	exercise := models.Exercise{
		Type:         enums.ExerciseTypeBasicDirect,
		Status:       enums.ExerciseStatusPending,
		UserID:       user.ID,
		ScheduledFor: &scheduled,
	}
	require.NoError(t, db.DB.Create(&exercise).Error)
	require.NoError(t, db.DB.Create(&models.ExerciseVocabulary{
		ExerciseID:   exercise.ID,
		VocabularyID: vocab.ID,
		IsCorrect:    true,
		Position:     0,
	}).Error)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body services.ExerciseListResponse
	testkit.DecodeJSON(t, rec, &body)
	assert.Empty(t, body.Data, "not-started exercises must be excluded")
}

func TestGetExercisesInvalidPagination(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	// page_size > 1000 triggers ErrInvalidPageSize → 400.
	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises?page_size=5000", nil)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Contains(t, body, "error")
}

func TestGetExercisesOwnershipIsolation(t *testing.T) {
	testkit.Truncate(t)

	userA := testkit.CreateUser(t, testkit.WithName("A"))
	userB := testkit.CreateUser(t, testkit.WithName("B"))

	vocab := exerciseSeedVocabulary(t, userB.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	exerciseSeedExercise(t, userB.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, userA, http.MethodGet, "/api/exercises", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body services.ExerciseListResponse
	testkit.DecodeJSON(t, rec, &body)
	assert.Empty(t, body.Data, "user A must not see user B's exercises")
}

// ===========================================================================
// GET /api/exercises/by-ids (GetExercisesByIDs)
// ===========================================================================

func TestGetExercisesByIDsRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/exercises/by-ids?ids="+uuid.New().String(), nil)
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestGetExercisesByIDsHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex1 := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusCompleted, vocab.ID)
	ex2 := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicReversed, enums.ExerciseStatusFailed, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodGet,
		"/api/exercises/by-ids?ids="+ex1.ID.String()+","+ex2.ID.String(), nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body []services.ExerciseListExercise
	testkit.DecodeJSON(t, rec, &body)

	require.Len(t, body, 2)
	ids := map[uuid.UUID]bool{}
	for _, e := range body {
		ids[e.ID] = true
	}
	assert.True(t, ids[ex1.ID])
	assert.True(t, ids[ex2.ID])
}

func TestGetExercisesByIDsMissingParam(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises/by-ids", nil)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "ids parameter is required", body["error"])
}

func TestGetExercisesByIDsInvalidUUID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises/by-ids?ids=not-a-uuid", nil)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Contains(t, body["error"], "invalid id")
}

// Ownership isolation: requesting another user's exercise by id returns nothing.
func TestGetExercisesByIDsOwnershipIsolation(t *testing.T) {
	testkit.Truncate(t)

	userA := testkit.CreateUser(t, testkit.WithName("A"))
	userB := testkit.CreateUser(t, testkit.WithName("B"))

	vocab := exerciseSeedVocabulary(t, userB.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, userB.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusCompleted, vocab.ID)

	rec := testkit.AuthedRequest(t, userA, http.MethodGet, "/api/exercises/by-ids?ids="+ex.ID.String(), nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body []services.ExerciseListExercise
	testkit.DecodeJSON(t, rec, &body)
	assert.Empty(t, body, "user A must not fetch user B's exercise by id")
}

// ===========================================================================
// GET /api/exercises/statistics (GetExerciseStatistics)
// ===========================================================================

func TestGetExerciseStatisticsRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/exercises/statistics", nil)
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestGetExerciseStatisticsCounts(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)

	exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)
	exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusCompleted, vocab.ID)
	exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusCompleted, vocab.ID)
	exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusFailed, vocab.ID)
	exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusIgnored, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises/statistics", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body services.ExerciseStatistics
	testkit.DecodeJSON(t, rec, &body)

	assert.Equal(t, int64(1), body.InProgress)
	assert.Equal(t, int64(2), body.Done)
	assert.Equal(t, int64(1), body.Failed)
	assert.Equal(t, int64(1), body.Ignored)
}

func TestGetExerciseStatisticsOwnershipIsolation(t *testing.T) {
	testkit.Truncate(t)

	userA := testkit.CreateUser(t, testkit.WithName("A"))
	userB := testkit.CreateUser(t, testkit.WithName("B"))

	vocab := exerciseSeedVocabulary(t, userB.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	exerciseSeedExercise(t, userB.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusCompleted, vocab.ID)

	rec := testkit.AuthedRequest(t, userA, http.MethodGet, "/api/exercises/statistics", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body services.ExerciseStatistics
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, int64(0), body.Done, "user A must not count user B's exercises")
	assert.Equal(t, int64(0), body.InProgress)
}

func TestGetExerciseStatisticsDailyActivityUsesUserTimezone(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{TimeZone: "Europe/Rome"}))
	vocab := exerciseSeedVocabulary(t, user.ID, "today", "oggi", enums.LanguageEn, enums.LanguageIt)

	location, err := time.LoadLocation("Europe/Rome")
	require.NoError(t, err)
	now := time.Now().In(location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 30, 0, 0, location)
	yesterday := today.AddDate(0, 0, -1)

	completed := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusCompleted, vocab.ID)
	failed := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusFailed, vocab.ID)
	require.NoError(t, db.DB.Model(&completed).UpdateColumn("finished_at", today.UTC()).Error)
	require.NoError(t, db.DB.Model(&failed).UpdateColumn("finished_at", yesterday.UTC()).Error)
	require.NoError(t, db.DB.Model(&vocab).UpdateColumn("created_at", today.UTC()).Error)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises/statistics", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body services.ExerciseStatistics
	testkit.DecodeJSON(t, rec, &body)

	require.Len(t, body.ExerciseActivity, 8)
	assert.Equal(t, today.Format("2006-01-02"), body.ExerciseActivity[7].Date)
	assert.Equal(t, int64(1), body.ExerciseActivity[7].Completed)
	assert.Equal(t, int64(1), body.ExerciseActivity[6].Failed)

	require.NotEmpty(t, body.VocabularyActivity)
	lastVocabularyDay := body.VocabularyActivity[len(body.VocabularyActivity)-1]
	assert.Equal(t, today.Format("2006-01-02"), lastVocabularyDay.Date)
	assert.Equal(t, int64(1), lastVocabularyDay.Count)
}

// ===========================================================================
// POST /api/exercises/random (RandomExercise)
// ===========================================================================

func TestRandomExerciseRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/exercises/random", nil)
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestRandomExerciseNoVocabulary(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/random", nil)
	testkit.RequireStatus(t, rec, http.StatusUnprocessableEntity)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, services.ErrNoVocabularyForExercise.Error(), body["error"])
}

func TestRandomExerciseAllMastered(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)

	// Mark mastered: knowledge 100 + mastered_at set so it is not eligible.
	masteredAt := time.Now().UTC()
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).
		Where("id = ?", vocab.ID).
		Updates(map[string]any{
			"progress":    models.ProgressEntries{{Knowledge: 100, Type: enums.KnowledgeTypeTranslation}},
			"mastered_at": masteredAt,
		}).Error)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/random", nil)
	testkit.RequireStatus(t, rec, http.StatusUnprocessableEntity)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, services.ErrAllVocabularyMastered.Error(), body["error"])
}

func TestRandomExerciseUserPathCompletesAndAppearsInHistory(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	// A single eligible vocabulary supports typed and character-building exercises.
	_ = exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/random", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body struct {
		ExerciseID     uuid.UUID                    `json:"exercise_id"`
		Type           enums.ExerciseType           `json:"type"`
		QuestionWord   string                       `json:"question_word"`
		Language       enums.Language               `json:"language"`
		AnswerLanguage enums.Language               `json:"answer_language"`
		AudioWordID    *uuid.UUID                   `json:"audio_word_id"`
		Description    string                       `json:"description"`
		Options        []string                     `json:"options"`
		Cards          []services.ExerciseMatchCard `json:"cards"`
	}
	testkit.DecodeJSON(t, rec, &body)

	require.NotEqual(t, uuid.Nil, body.ExerciseID)
	// Choice and match exercises need additional vocabulary; typed, audio, description,
	// and character exercises only need the correct word pair.
	assert.Contains(t, []enums.ExerciseType{
		enums.ExerciseTypeBasicDirect,
		enums.ExerciseTypeBasicReversed,
		enums.ExerciseTypeCharactersDirect,
		enums.ExerciseTypeCharactersReversed,
		enums.ExerciseTypeAudioDirect,
		enums.ExerciseTypeAudioReversed,
		enums.ExerciseTypeDescriptionDirect,
		enums.ExerciseTypeDescriptionReversed,
	}, body.Type)

	// DB side effect: the exercise exists, is in progress and belongs to user.
	stored := exerciseReload(t, body.ExerciseID)
	assert.Equal(t, user.ID, stored.UserID)
	assert.Equal(t, enums.ExerciseStatusInProgress, stored.Status)
	require.NotNil(t, stored.StartedAt)

	// Question word matches expected direction. Words are seeded with their raw
	// casing directly in the DB (no service-level normalization here).
	if body.Type == enums.ExerciseTypeDescriptionReversed {
		assert.Empty(t, body.QuestionWord)
		assert.NotEmpty(t, body.Description)
		assert.Equal(t, enums.LanguageDe, body.Language)
		assert.Equal(t, enums.LanguageDe, body.AnswerLanguage)
	} else if body.Type == enums.ExerciseTypeDescriptionDirect {
		assert.Equal(t, enums.LanguageEn, body.AnswerLanguage)
		assert.Empty(t, body.QuestionWord)
		assert.NotEmpty(t, body.Description)
		assert.Equal(t, enums.LanguageEn, body.Language)
	} else if body.Type == enums.ExerciseTypeBasicReversed ||
		body.Type == enums.ExerciseTypeCharactersReversed ||
		body.Type == enums.ExerciseTypeAudioReversed {
		assert.Equal(t, enums.LanguageEn, body.AnswerLanguage)
		assert.Equal(t, "Hund", body.QuestionWord)
		assert.Equal(t, enums.LanguageDe, body.Language)
		if body.Type == enums.ExerciseTypeCharactersReversed {
			assert.ElementsMatch(t, []string{"d", "o", "g"}, body.Options)
		}
		if body.Type == enums.ExerciseTypeAudioReversed {
			require.NotNil(t, body.AudioWordID)
		}
	} else {
		assert.Equal(t, "dog", body.QuestionWord)
		assert.Equal(t, enums.LanguageEn, body.Language)
		assert.Equal(t, enums.LanguageDe, body.AnswerLanguage)
		if body.Type == enums.ExerciseTypeCharactersDirect {
			assert.ElementsMatch(t, []string{"H", "u", "n", "d"}, body.Options)
		}
		if body.Type == enums.ExerciseTypeAudioDirect {
			require.NotNil(t, body.AudioWordID)
		}
	}

	answer := "Hund"
	if body.AnswerLanguage == enums.LanguageEn {
		answer = "dog"
	}
	verifyRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/exercises/"+body.ExerciseID.String()+"/verify",
		map[string]any{"answer": answer},
	)
	testkit.RequireStatus(t, verifyRec, http.StatusOK)
	var verification struct {
		Result        string `json:"result"`
		CorrectAnswer string `json:"correct_answer"`
		Knowledge     int    `json:"knowledge"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, verifyRec, &verification)
	assert.Equal(t, services.ExerciseVocabularyResultCorrect, verification.Result)
	assert.Equal(t, answer, verification.CorrectAnswer)
	expectedDelta := services.ExerciseBasicCorrectProgressDelta
	if body.Type == enums.ExerciseTypeCharactersDirect || body.Type == enums.ExerciseTypeCharactersReversed {
		expectedDelta = services.ExerciseCharacterCorrectProgressDelta
	}
	assert.Equal(t, expectedDelta, verification.ProgressDelta)
	assert.Equal(t, expectedDelta, verification.Knowledge)

	completed := exerciseReload(t, body.ExerciseID)
	assert.Equal(t, enums.ExerciseStatusCompleted, completed.Status)
	require.NotNil(t, completed.FinishedAt)

	historyRec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises", nil)
	testkit.RequireStatus(t, historyRec, http.StatusOK)
	var history services.ExerciseListResponse
	testkit.DecodeJSON(t, historyRec, &history)
	require.Len(t, history.Data, 1)
	assert.Equal(t, body.ExerciseID, history.Data[0].ID)
	assert.Equal(t, enums.ExerciseStatusCompleted, history.Data[0].Status)

	statisticsRec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises/statistics", nil)
	testkit.RequireStatus(t, statisticsRec, http.StatusOK)
	var statistics services.ExerciseStatistics
	testkit.DecodeJSON(t, statisticsRec, &statistics)
	assert.Equal(t, int64(1), statistics.Done)
	assert.Equal(t, int64(0), statistics.InProgress)
}

// ===========================================================================
// POST /api/exercises/:id/verify (VerifyExercise)
// ===========================================================================

func TestVerifyExerciseRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/exercises/"+uuid.New().String()+"/verify",
		map[string]any{"answer": "Hund"})
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestVerifyExerciseInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/not-a-uuid/verify",
		map[string]any{"answer": "Hund"})
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid exercise id", body["error"])
}

func TestVerifyExerciseMissingBody(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+ex.ID.String()+"/verify",
		map[string]any{})
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "answer is required", body["error"])
}

func TestVerifyExerciseMalformedJSON(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRawRequest(
		t,
		user,
		http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify",
		"}{ not json",
		nil,
	)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)
}

func TestVerifyExerciseNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+uuid.New().String()+"/verify", map[string]any{"answer": "Hund"})
	testkit.RequireStatus(t, rec, http.StatusNotFound)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "exercise not found", body["error"])
}

func TestVerifyExerciseCorrectAnswer(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "Hund"})
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body struct {
		Result        string `json:"result"`
		CorrectAnswer string `json:"correct_answer"`
		Knowledge     int    `json:"knowledge"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, rec, &body)

	assert.Equal(t, "correct", body.Result)
	assert.Equal(t, "Hund", body.CorrectAnswer)
	assert.Equal(t, services.ExerciseBasicCorrectProgressDelta, body.ProgressDelta)
	assert.Equal(t, services.ExerciseBasicCorrectProgressDelta, body.Knowledge) // 0 + 15

	// DB side effects.
	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusCompleted, stored.Status)
	require.NotNil(t, stored.FinishedAt)

	updatedVocab := exerciseReloadVocabulary(t, vocab.ID)
	assert.Equal(t, services.ExerciseBasicCorrectProgressDelta, exerciseTranslationKnowledge(t, updatedVocab.Progress))

	link := exerciseLink(t, ex.ID, vocab.ID)
	require.NotNil(t, link.Result)
	assert.Equal(t, services.ExerciseVocabularyResultCorrect, *link.Result)
	require.NotNil(t, link.ProgressDelta)
	assert.Equal(t, services.ExerciseBasicCorrectProgressDelta, *link.ProgressDelta)
	require.NotNil(t, link.AnsweredAt)
}

func TestCharacterExercisesSupportPortugueseAndUkrainianScripts(t *testing.T) {
	tests := []struct {
		language enums.Language
		answer   string
	}{
		{enums.LanguagePt, "olá"},
		{enums.LanguageUk, "привіт"},
	}

	for _, test := range tests {
		t.Run(string(test.language), func(t *testing.T) {
			testkit.Truncate(t)
			user := testkit.CreateUser(t)
			exerciseSeedVocabulary(t, user.ID, "hello", test.answer, enums.LanguageEn, test.language)

			result, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeCharactersDirect)
			require.NoError(t, err)
			assert.Equal(t, enums.ExerciseTypeCharactersDirect, result.Type)
			assert.Equal(t, enums.LanguageEn, result.Language)
			assert.Equal(t, test.language, result.AnswerLanguage)
			assert.ElementsMatch(t, services.AnswerCharacters(test.answer), result.Options)

			verified, err := services.VerifyExerciseAnswer(result.ExerciseID, user.ID, test.answer)
			require.NoError(t, err)
			assert.Equal(t, "correct", verified.Result)
		})
	}
}

func TestVerifyCharacterExerciseDirectAndReversed(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		exerciseType  enums.ExerciseType
		answer        string
		correctAnswer string
	}{
		{
			name:          "direct",
			exerciseType:  enums.ExerciseTypeCharactersDirect,
			answer:        "Hund",
			correctAnswer: "Hund",
		},
		{
			name:          "reversed",
			exerciseType:  enums.ExerciseTypeCharactersReversed,
			answer:        "dog",
			correctAnswer: "dog",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			testkit.Truncate(t)

			user := testkit.CreateUser(t)
			vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
			exercise := exerciseSeedExercise(t, user.ID, testCase.exerciseType, enums.ExerciseStatusInProgress, vocab.ID)

			result, err := services.VerifyExerciseAnswer(exercise.ID, user.ID, testCase.answer)
			require.NoError(t, err)
			assert.Equal(t, "correct", result.Result)
			assert.Equal(t, testCase.correctAnswer, result.CorrectAnswer)
			assert.Equal(t, services.ExerciseCharacterCorrectProgressDelta, result.ProgressDelta)
			assert.Equal(t, services.ExerciseCharacterCorrectProgressDelta, result.Knowledge)

			link := exerciseLink(t, exercise.ID, vocab.ID)
			require.NotNil(t, link.ResultReason)
			assert.Equal(t, services.ExerciseVocabularyResultReasonCharacterAnswer, *link.ResultReason)
		})
	}
}

func TestApplyCharacterTapTracksDuplicateCharacters(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "letter", "lettera", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeCharactersDirect, enums.ExerciseStatusPending, vocab.ID)
	order := []int{6, -1, 0, 5, 2, 1, 4, 3, -1}
	require.NoError(t, services.StartCharacterExercise(exercise.ID, 901, order))

	var board *services.CharacterBoardState
	for index := range []rune("lettera") {
		var finished bool
		var err error
		board, finished, err = services.ApplyCharacterTap(exercise.ID, user.ID, index)
		require.NoError(t, err)
		assert.Equal(t, index == len([]rune("lettera"))-1, finished)
	}

	require.NotNil(t, board)
	assert.Equal(t, "lettera", board.Answer)
	assert.Equal(t, order, board.Order)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6}, board.Chosen)

	stored := exerciseReload(t, exercise.ID)
	require.NotNil(t, stored.CharacterState)
	assert.Equal(t, enums.ExerciseStatusInProgress, stored.Status)
}

func TestBuildCharacterBoardRandomizesCharactersAndPaddingSlots(t *testing.T) {
	board := services.BuildCharacterBoardForAnswer("letter")

	assert.Equal(t, services.AnswerCharacters("letter"), board.Characters)
	require.Len(t, board.Order, 9, "a 3x3 character grid keeps actions in a separate row")
	assert.ElementsMatch(t, []int{0, 1, 2, 3, 4, 5, -1, -1, -1}, board.Order)
	assert.Empty(t, board.Chosen)
	assert.Empty(t, board.Answer)
}

func TestRemoveLastCharacterSelectionRestoresOnlyLastCharacter(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "letter", "lettera", enums.LanguageEn, enums.LanguageIt)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeCharactersDirect, enums.ExerciseStatusPending, vocab.ID)
	order := []int{6, -1, 0, 5, 2, 1, 4, 3, -1}
	require.NoError(t, services.StartCharacterExercise(exercise.ID, 902, order))

	_, _, err := services.ApplyCharacterTap(exercise.ID, user.ID, 0)
	require.NoError(t, err)
	_, _, err = services.ApplyCharacterTap(exercise.ID, user.ID, 2)
	require.NoError(t, err)

	board, err := services.RemoveLastCharacterSelection(exercise.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "l", board.Answer)
	assert.Equal(t, []int{0}, board.Chosen)
	assert.Equal(t, order, board.Order)
	assert.Equal(t, services.AnswerCharacters("lettera"), board.Characters)

	board, err = services.RemoveLastCharacterSelection(exercise.ID, user.ID)
	require.NoError(t, err)
	assert.Empty(t, board.Answer)
	assert.Empty(t, board.Chosen)

	board, err = services.RemoveLastCharacterSelection(exercise.ID, user.ID)
	require.NoError(t, err)
	assert.Empty(t, board.Answer)
	assert.Empty(t, board.Chosen)
}

func TestVerifyExerciseWrongAnswer(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "completely-wrong"})
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body struct {
		Result        string `json:"result"`
		CorrectAnswer string `json:"correct_answer"`
		Knowledge     int    `json:"knowledge"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, rec, &body)

	assert.Equal(t, "wrong", body.Result)
	assert.Equal(t, "Hund", body.CorrectAnswer)
	assert.Equal(t, services.ExerciseBasicWrongProgressDelta, body.ProgressDelta)
	assert.Equal(t, 0, body.Knowledge) // clamp(0 - 15) = 0

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusFailed, stored.Status)

	link := exerciseLink(t, ex.ID, vocab.ID)
	require.NotNil(t, link.Result)
	assert.Equal(t, services.ExerciseVocabularyResultWrong, *link.Result)
}

// A one-character typo is treated as "almost" correct (Damerau-Levenshtein within threshold).
func TestVerifyExerciseAlmostAnswer(t *testing.T) {
	testCases := []struct {
		name          string
		exerciseType  enums.ExerciseType
		translation   string
		answer        string
		expectedDelta int
	}{
		{
			name:          "substitution",
			exerciseType:  enums.ExerciseTypeBasicDirect,
			translation:   "Hund",
			answer:        "Hand",
			expectedDelta: services.ExerciseBasicAlmostProgressDelta,
		},
		{
			name:          "adjacent transposition in character exercise",
			exerciseType:  enums.ExerciseTypeCharactersDirect,
			translation:   "peach",
			answer:        "peahc",
			expectedDelta: services.ExerciseCharacterAlmostProgressDelta,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testkit.Truncate(t)

			user := testkit.CreateUser(t)
			vocab := exerciseSeedVocabulary(t, user.ID, "dog", testCase.translation, enums.LanguageEn, enums.LanguageDe)
			exercise := exerciseSeedExercise(t, user.ID, testCase.exerciseType, enums.ExerciseStatusInProgress, vocab.ID)

			rec := testkit.AuthedRequest(t, user, http.MethodPost,
				"/api/exercises/"+exercise.ID.String()+"/verify", map[string]any{"answer": testCase.answer})
			testkit.RequireStatus(t, rec, http.StatusOK)

			var body struct {
				Result        string `json:"result"`
				ProgressDelta int    `json:"progress_delta"`
			}
			testkit.DecodeJSON(t, rec, &body)

			assert.Equal(t, "almost", body.Result)
			assert.Equal(t, testCase.expectedDelta, body.ProgressDelta)

			stored := exerciseReload(t, exercise.ID)
			assert.Equal(t, enums.ExerciseStatusCompleted, stored.Status)
		})
	}
}

func TestVerifyKnownVocabularyRepetitionProgress(t *testing.T) {
	testCases := []struct {
		name              string
		exerciseType      enums.ExerciseType
		currentKnowledge  int
		answer            string
		expectedResult    string
		expectedDelta     int
		expectedKnowledge int
		expectedStatus    enums.ExerciseStatus
	}{
		{
			name:              "correct keeps mastered knowledge",
			exerciseType:      enums.ExerciseTypeBasicDirect,
			currentKnowledge:  100,
			answer:            "Hund",
			expectedResult:    "correct",
			expectedDelta:     0,
			expectedKnowledge: 100,
			expectedStatus:    enums.ExerciseStatusCompleted,
		},
		{
			name:              "almost keeps mastered knowledge",
			exerciseType:      enums.ExerciseTypeBasicDirect,
			currentKnowledge:  100,
			answer:            "Hand",
			expectedResult:    "almost",
			expectedDelta:     0,
			expectedKnowledge: 100,
			expectedStatus:    enums.ExerciseStatusCompleted,
		},
		{
			name:              "wrong subtracts twenty five",
			exerciseType:      enums.ExerciseTypeBasicReversed,
			currentKnowledge:  100,
			answer:            "completely wrong",
			expectedResult:    "wrong",
			expectedDelta:     services.KnownVocabularyRepetitionFailProgressDelta,
			expectedKnowledge: 75,
			expectedStatus:    enums.ExerciseStatusFailed,
		},
		{
			name:              "correct restores the normal basic amount after knowledge changed",
			exerciseType:      enums.ExerciseTypeBasicDirect,
			currentKnowledge:  60,
			answer:            "Hund",
			expectedResult:    "correct",
			expectedDelta:     services.ExerciseBasicCorrectProgressDelta,
			expectedKnowledge: 75,
			expectedStatus:    enums.ExerciseStatusCompleted,
		},
		{
			name:              "almost restores the partial amount after knowledge changed",
			exerciseType:      enums.ExerciseTypeBasicReversed,
			currentKnowledge:  60,
			answer:            "dug",
			expectedResult:    "almost",
			expectedDelta:     services.ExerciseBasicAlmostProgressDelta,
			expectedKnowledge: 65,
			expectedStatus:    enums.ExerciseStatusCompleted,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testkit.Truncate(t)

			user := testkit.CreateUser(t)
			vocabulary := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)

			exerciseSetTranslationKnowledge(t, vocabulary.ID, 100)
			exercise := exerciseSeedExercise(t, user.ID, testCase.exerciseType, enums.ExerciseStatusInProgress, vocabulary.ID)
			exerciseMarkKnownVocabularyRepetition(t, exercise.ID)

			exerciseSetTranslationKnowledge(t, vocabulary.ID, testCase.currentKnowledge)

			result, err := services.VerifyExerciseAnswer(exercise.ID, user.ID, testCase.answer)
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedResult, result.Result)
			assert.Equal(t, testCase.expectedDelta, result.ProgressDelta)
			assert.Equal(t, testCase.expectedKnowledge, result.Knowledge)

			storedExercise := exerciseReload(t, exercise.ID)
			assert.Equal(t, testCase.expectedStatus, storedExercise.Status)

			storedVocabulary := exerciseReloadVocabulary(t, vocabulary.ID)
			assert.Equal(t, testCase.expectedKnowledge, exerciseTranslationKnowledge(t, storedVocabulary.Progress))
			if testCase.expectedKnowledge == 100 {
				assert.NotNil(t, storedVocabulary.MasteredAt)
			} else {
				assert.Nil(t, storedVocabulary.MasteredAt)
			}

			link := exerciseLink(t, exercise.ID, vocabulary.ID)
			require.NotNil(t, link.ProgressDelta)
			assert.Equal(t, testCase.expectedDelta, *link.ProgressDelta)
			require.NotNil(t, link.KnowledgeAfter)
			assert.Equal(t, testCase.expectedKnowledge, *link.KnowledgeAfter)
		})
	}
}

func TestKnownVocabularyRepetitionSkippedAnswerSubtractsTwentyFive(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	exerciseSetTranslationKnowledge(t, vocabulary.ID, 100)

	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocabulary.ID)
	exerciseMarkKnownVocabularyRepetition(t, exercise.ID)

	updated, knowledge, err := services.FinishExercise(
		exercise.ID,
		enums.ExerciseStatusFailed,
		services.ExerciseVocabularyResultIgnored,
		services.ExerciseVocabularyResultReasonSkipped,
		services.ExerciseBasicWrongProgressDelta,
	)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, 75, knowledge)

	link := exerciseLink(t, exercise.ID, vocabulary.ID)
	require.NotNil(t, link.ProgressDelta)
	assert.Equal(t, services.KnownVocabularyRepetitionFailProgressDelta, *link.ProgressDelta)
}

// Verifying an exercise that is not in progress returns 409 Conflict.
func TestVerifyExerciseNotInProgress(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusCompleted, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "Hund"})
	testkit.RequireStatus(t, rec, http.StatusConflict)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "exercise is not in progress", body["error"])
}

func TestVerifyExerciseIgnoredAfterVocabularyDeletion(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocabulary.ID)
	require.NoError(t, services.DeleteVocabulary(user.ID, vocabulary.ID))

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+exercise.ID.String()+"/verify", map[string]any{"answer": "Hund"})
	testkit.RequireStatus(t, rec, http.StatusConflict)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, services.ErrExerciseVocabularyDeleted.Error(), body["error"])
}

func TestGetDueExerciseRemindersExcludesDeletedVocabulary(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
		Telegram: models.UserTelegramSettings{BotEnabled: true},
	}))
	deletedVocabulary := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	activeVocabulary := exerciseSeedVocabulary(t, user.ID, "cat", "Katze", enums.LanguageEn, enums.LanguageDe)
	deletedExercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, deletedVocabulary.ID)
	activeExercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, activeVocabulary.ID)

	startedAt := time.Now().UTC().Add(-25 * time.Hour)
	for index, exerciseID := range []uuid.UUID{deletedExercise.ID, activeExercise.ID} {
		messageID := int64(index + 1)
		require.NoError(t, db.DB.Model(&models.Exercise{}).
			Where("id = ?", exerciseID).
			Updates(map[string]any{"started_at": startedAt, "telegram_message_id": messageID}).Error)
	}
	require.NoError(t, db.DB.Delete(&models.Vocabulary{}, "id = ?", deletedVocabulary.ID).Error)

	reminders, err := services.GetDueExerciseReminders(time.Now().UTC())
	require.NoError(t, err)
	require.Len(t, reminders, 1)
	assert.Equal(t, activeExercise.ID, reminders[0].ExerciseID)
}

func TestTelegramExerciseQueriesExcludeDeletedUsers(t *testing.T) {
	testkit.Truncate(t)

	const telegramID int64 = 880055
	user := testkit.CreateUser(t,
		testkit.WithTelegramID(telegramID),
		testkit.WithSettings(models.UserSettings{
			Telegram: models.UserTelegramSettings{
				BotEnabled:            true,
				DailyQuestionsEnabled: true,
			},
		}),
	)
	vocabularyIDs := exerciseSeedFiveVocabularies(t, user.ID)
	now := time.Now().UTC()

	pendingExercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusPending, vocabularyIDs[0])
	require.NoError(t, db.DB.Model(&pendingExercise).Update("scheduled_for", now).Error)
	pendingMatch := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusPending, vocabularyIDs)
	require.NoError(t, db.DB.Model(&pendingMatch).Update("scheduled_for", now).Error)

	reminderExercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocabularyIDs[1])
	messageID := int64(901)
	require.NoError(t, db.DB.Model(&reminderExercise).Updates(map[string]any{
		"started_at":          now.Add(-25 * time.Hour),
		"telegram_message_id": messageID,
	}).Error)

	require.NoError(t, db.DB.Delete(&user).Error)

	dailyUsers, err := services.GetUsersWithEnabledDailyQuestions()
	require.NoError(t, err)
	assert.Empty(t, dailyUsers)
	due, err := services.GetDuePendingExercises(now)
	require.NoError(t, err)
	assert.Empty(t, due)
	dueMatches, err := services.GetDuePendingMatchExercises(now)
	require.NoError(t, err)
	assert.Empty(t, dueMatches)
	reminders, err := services.GetDueExerciseReminders(now)
	require.NoError(t, err)
	assert.Empty(t, reminders)

	byMessage, err := services.GetExerciseByTelegramMessage(messageID, telegramID)
	require.NoError(t, err)
	assert.Nil(t, byMessage)
	byID, err := services.GetExerciseByTelegramExerciseID(reminderExercise.ID, telegramID)
	require.NoError(t, err)
	assert.Nil(t, byID)
	words, err := services.GetExerciseWordsByTelegram(reminderExercise.ID, telegramID)
	require.NoError(t, err)
	assert.Nil(t, words)

	var preservedExercises int64
	require.NoError(t, db.DB.Model(&models.Exercise{}).Where("user_id = ?", user.ID).Count(&preservedExercises).Error)
	assert.Equal(t, int64(3), preservedExercises)
}

// Verifying a match/pairs exercise via the typed endpoint is a 400.
func TestVerifyExerciseMatchPairsRejected(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabs := exerciseSeedFiveVocabularies(t, user.ID)
	ex := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusInProgress, vocabs)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "A"})
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, services.ErrInvalidMatchPairResults.Error(), body["error"])
}

// Ownership: user A cannot verify user B's exercise (treated as not found).
func TestVerifyExerciseOwnershipIsolation(t *testing.T) {
	testkit.Truncate(t)

	userA := testkit.CreateUser(t, testkit.WithName("A"))
	userB := testkit.CreateUser(t, testkit.WithName("B"))

	vocab := exerciseSeedVocabulary(t, userB.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, userB.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, userA, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "Hund"})
	testkit.RequireStatus(t, rec, http.StatusNotFound)

	// B's exercise is untouched.
	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusInProgress, stored.Status)
}

// Choice exercise: a correct selection completes; a valid-but-wrong option fails.
func TestVerifyExerciseChoiceCorrectWithDeletedDistractor(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	correct := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	d1 := exerciseSeedVocabulary(t, user.ID, "cat", "Katze", enums.LanguageEn, enums.LanguageDe)
	d2 := exerciseSeedVocabulary(t, user.ID, "bird", "Vogel", enums.LanguageEn, enums.LanguageDe)
	d3 := exerciseSeedVocabulary(t, user.ID, "fish", "Fisch", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedChoiceExercise(t, user.ID, enums.ExerciseTypeChoiceDirect, enums.ExerciseStatusInProgress,
		[]uuid.UUID{correct.ID, d1.ID, d2.ID, d3.ID})
	require.NoError(t, services.DeleteVocabulary(user.ID, d1.ID))

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "Hund"})
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body struct {
		Result        string `json:"result"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "correct", body.Result)
	assert.Equal(t, services.ExerciseChoiceCorrectProgressDelta, body.ProgressDelta)

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusCompleted, stored.Status)
}

func TestDeleteVocabularyCancelsAffectedInProgressExercisesWithoutProgress(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	correct := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	distractor := exerciseSeedVocabulary(t, user.ID, "cat", "Katze", enums.LanguageEn, enums.LanguageDe)
	d2 := exerciseSeedVocabulary(t, user.ID, "bird", "Vogel", enums.LanguageEn, enums.LanguageDe)
	d3 := exerciseSeedVocabulary(t, user.ID, "fish", "Fisch", enums.LanguageEn, enums.LanguageDe)

	basic := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, correct.ID)
	choice := exerciseSeedChoiceExercise(t, user.ID, enums.ExerciseTypeChoiceDirect, enums.ExerciseStatusInProgress,
		[]uuid.UUID{correct.ID, distractor.ID, d2.ID, d3.ID})
	unaffectedChoice := exerciseSeedChoiceExercise(t, user.ID, enums.ExerciseTypeChoiceDirect, enums.ExerciseStatusInProgress,
		[]uuid.UUID{d2.ID, correct.ID, distractor.ID, d3.ID})

	before := exerciseTranslationKnowledge(t, exerciseReloadVocabulary(t, correct.ID).Progress)
	require.NoError(t, services.DeleteVocabulary(user.ID, correct.ID))

	for _, exerciseID := range []uuid.UUID{basic.ID, choice.ID} {
		stored := exerciseReload(t, exerciseID)
		assert.Equal(t, enums.ExerciseStatusIgnored, stored.Status)
		assert.NotNil(t, stored.FinishedAt)

		link := exerciseLink(t, exerciseID, correct.ID)
		require.NotNil(t, link.Result)
		assert.Equal(t, services.ExerciseVocabularyResultIgnored, *link.Result)
		require.NotNil(t, link.ResultReason)
		assert.Equal(t, services.ExerciseVocabularyResultReasonDeletedVocabulary, *link.ResultReason)
		assert.Nil(t, link.ProgressDelta)
	}

	assert.Equal(t, enums.ExerciseStatusInProgress, exerciseReload(t, unaffectedChoice.ID).Status,
		"deleting only a choice distractor must not cancel the exercise")
	after := exerciseTranslationKnowledge(t, exerciseReloadVocabulary(t, correct.ID).Progress)
	assert.Equal(t, before, after)
}

func TestVerifyExerciseChoiceWrong(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	correct := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	d1 := exerciseSeedVocabulary(t, user.ID, "cat", "Katze", enums.LanguageEn, enums.LanguageDe)
	d2 := exerciseSeedVocabulary(t, user.ID, "bird", "Vogel", enums.LanguageEn, enums.LanguageDe)
	d3 := exerciseSeedVocabulary(t, user.ID, "fish", "Fisch", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedChoiceExercise(t, user.ID, enums.ExerciseTypeChoiceDirect, enums.ExerciseStatusInProgress,
		[]uuid.UUID{correct.ID, d1.ID, d2.ID, d3.ID})

	// "Katze" is a valid option but not the correct one.
	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "Katze"})
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body struct {
		Result        string `json:"result"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "wrong", body.Result)
	assert.Equal(t, services.ExerciseChoiceWrongProgressDelta, body.ProgressDelta)

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusFailed, stored.Status)
}

func TestVerifyExerciseChoiceWithDeletedDistractor(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	correct := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	d1 := exerciseSeedVocabulary(t, user.ID, "cat", "Katze", enums.LanguageEn, enums.LanguageDe)
	d2 := exerciseSeedVocabulary(t, user.ID, "bird", "Vogel", enums.LanguageEn, enums.LanguageDe)
	d3 := exerciseSeedVocabulary(t, user.ID, "fish", "Fisch", enums.LanguageEn, enums.LanguageDe)

	for _, testCase := range []struct {
		name       string
		selection  uuid.UUID
		wantResult string
		wantStatus enums.ExerciseStatus
	}{
		{name: "correct answer", selection: correct.ID, wantResult: "correct", wantStatus: enums.ExerciseStatusCompleted},
		{name: "deleted distractor", selection: d1.ID, wantResult: "wrong", wantStatus: enums.ExerciseStatusFailed},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ex := exerciseSeedChoiceExercise(t, user.ID, enums.ExerciseTypeChoiceDirect, enums.ExerciseStatusInProgress,
				[]uuid.UUID{correct.ID, d1.ID, d2.ID, d3.ID})
			require.NoError(t, services.DeleteVocabulary(user.ID, d1.ID))

			result, err := services.VerifyExerciseChoice(ex.ID, user.ID, testCase.selection)
			require.NoError(t, err)
			assert.Equal(t, testCase.wantResult, result.Result)
			assert.Equal(t, testCase.wantStatus, exerciseReload(t, ex.ID).Status)

			// Restore the distractor so the second case can build the same fixture.
			require.NoError(t, db.DB.Model(&models.Vocabulary{}).
				Where("id = ?", d1.ID).
				Update("deleted_at", nil).Error)
		})
	}
}

// ===========================================================================
// POST /api/exercises/:id/ignore (IgnoreExercise)
// ===========================================================================

func TestIgnoreExerciseRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/exercises/"+uuid.New().String()+"/ignore", nil)
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestIgnoreExerciseInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/not-a-uuid/ignore", nil)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid exercise id", body["error"])
}

func TestIgnoreExerciseNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+uuid.New().String()+"/ignore", nil)
	testkit.RequireStatus(t, rec, http.StatusNotFound)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "exercise not found", body["error"])
}

func TestIgnoreExerciseHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	exerciseSetTranslationKnowledge(t, vocab.ID, 50)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+ex.ID.String()+"/ignore", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "ignored", body["status"])

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusIgnored, stored.Status)
	require.NotNil(t, stored.FinishedAt)

	updatedVocabulary := exerciseReloadVocabulary(t, vocab.ID)
	assert.Equal(t, 35, exerciseTranslationKnowledge(t, updatedVocabulary.Progress))

	link := exerciseLink(t, ex.ID, vocab.ID)
	require.NotNil(t, link.Result)
	assert.Equal(t, services.ExerciseVocabularyResultIgnored, *link.Result)
	require.NotNil(t, link.ResultReason)
	assert.Equal(t, services.ExerciseVocabularyResultReasonSkipped, *link.ResultReason)
	require.NotNil(t, link.ProgressDelta)
	assert.Equal(t, services.ExerciseBasicWrongProgressDelta, *link.ProgressDelta)
	require.NotNil(t, link.KnowledgeAfter)
	assert.Equal(t, 35, *link.KnowledgeAfter)
	require.NotNil(t, link.AnsweredAt)
}

func TestIgnoreKnownVocabularyRepetitionSubtractsTwentyFive(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	exerciseSetTranslationKnowledge(t, vocabulary.ID, 100)

	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocabulary.ID)
	exerciseMarkKnownVocabularyRepetition(t, exercise.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+exercise.ID.String()+"/ignore", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	storedExercise := exerciseReload(t, exercise.ID)
	assert.Equal(t, enums.ExerciseStatusIgnored, storedExercise.Status)

	storedVocabulary := exerciseReloadVocabulary(t, vocabulary.ID)
	assert.Equal(t, 75, exerciseTranslationKnowledge(t, storedVocabulary.Progress))
	assert.Nil(t, storedVocabulary.MasteredAt)

	link := exerciseLink(t, exercise.ID, vocabulary.ID)
	require.NotNil(t, link.Result)
	assert.Equal(t, services.ExerciseVocabularyResultIgnored, *link.Result)
	require.NotNil(t, link.ResultReason)
	assert.Equal(t, services.ExerciseVocabularyResultReasonSkipped, *link.ResultReason)
	require.NotNil(t, link.ProgressDelta)
	assert.Equal(t, services.KnownVocabularyRepetitionFailProgressDelta, *link.ProgressDelta)
	require.NotNil(t, link.KnowledgeAfter)
	assert.Equal(t, 75, *link.KnowledgeAfter)
}

// Ignoring an already-finished exercise → 409 Conflict.
func TestIgnoreExerciseNotInProgress(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusCompleted, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+ex.ID.String()+"/ignore", nil)
	testkit.RequireStatus(t, rec, http.StatusConflict)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "exercise is not in progress", body["error"])
}

func TestIgnoreExerciseIgnoredAfterVocabularyDeletion(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocabulary.ID)
	require.NoError(t, services.DeleteVocabulary(user.ID, vocabulary.ID))

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+exercise.ID.String()+"/ignore", nil)
	testkit.RequireStatus(t, rec, http.StatusConflict)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, services.ErrExerciseVocabularyDeleted.Error(), body["error"])
}

func TestIgnoreExerciseOwnershipIsolation(t *testing.T) {
	testkit.Truncate(t)

	userA := testkit.CreateUser(t, testkit.WithName("A"))
	userB := testkit.CreateUser(t, testkit.WithName("B"))

	vocab := exerciseSeedVocabulary(t, userB.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, userB.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, userA, http.MethodPost, "/api/exercises/"+ex.ID.String()+"/ignore", nil)
	testkit.RequireStatus(t, rec, http.StatusNotFound)

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusInProgress, stored.Status, "B's exercise must be untouched")
}

// ===========================================================================
// POST /api/exercises/:id/match-pairs/complete (CompleteMatchPairsExercise)
// ===========================================================================

// exerciseMatchAttempts builds the all-correct attempt list for the 5 seeded
// vocabularies, pairing each original card with its translation card.
func exerciseMatchAttempts(vocabularyIDs []uuid.UUID) []map[string]any {
	attempts := make([]map[string]any, 0, len(vocabularyIDs))
	for _, id := range vocabularyIDs {
		attempts = append(attempts, map[string]any{
			"first_card_id":  id.String() + ":original",
			"second_card_id": id.String() + ":translation",
		})
	}
	return attempts
}

func TestCompleteMatchPairsRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost,
		"/api/exercises/"+uuid.New().String()+"/match-pairs/complete",
		map[string]any{"attempts": []any{map[string]any{"first_card_id": "a", "second_card_id": "b"}}})
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestCompleteMatchPairsInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/not-a-uuid/match-pairs/complete",
		map[string]any{"attempts": []any{map[string]any{"first_card_id": "a", "second_card_id": "b"}}})
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid exercise id", body["error"])
}

func TestCompleteMatchPairsMissingBody(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+uuid.New().String()+"/match-pairs/complete", map[string]any{})
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "attempts are required", body["error"])
}

func TestCompleteMatchPairsNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+uuid.New().String()+"/match-pairs/complete",
		map[string]any{"attempts": []any{map[string]any{"first_card_id": "a", "second_card_id": "b"}}})
	testkit.RequireStatus(t, rec, http.StatusNotFound)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "exercise not found", body["error"])
}

// Completing a non-match-pairs exercise type → 400 invalid match pair results.
func TestCompleteMatchPairsWrongType(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/match-pairs/complete",
		map[string]any{"attempts": []any{map[string]any{"first_card_id": "a", "second_card_id": "b"}}})
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, services.ErrInvalidMatchPairResults.Error(), body["error"])
}

func exerciseSeedFiveVocabularies(t *testing.T, userID uint) []uuid.UUID {
	t.Helper()
	pairs := [][2]string{{"dog", "Hund"}, {"cat", "Katze"}, {"bird", "Vogel"}, {"fish", "Fisch"}, {"horse", "Pferd"}}
	ids := make([]uuid.UUID, 0, len(pairs))
	for _, pair := range pairs {
		v := exerciseSeedVocabulary(t, userID, pair[0], pair[1], enums.LanguageEn, enums.LanguageDe)
		ids = append(ids, v.ID)
	}
	return ids
}

func TestCompleteMatchPairsAllCorrect(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabs := exerciseSeedFiveVocabularies(t, user.ID)
	ex := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusInProgress, vocabs)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/match-pairs/complete",
		map[string]any{"attempts": exerciseMatchAttempts(vocabs)})
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body services.MatchPairsCompleteResult
	testkit.DecodeJSON(t, rec, &body)

	assert.Equal(t, enums.ExerciseStatusCompleted, body.Status)
	require.Len(t, body.Results, 5)
	for _, r := range body.Results {
		require.NotNil(t, r.ExerciseResult)
		assert.Equal(t, services.ExerciseVocabularyResultCorrect, *r.ExerciseResult)
	}

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusCompleted, stored.Status)

	// Progress incremented for each vocabulary.
	updated := exerciseReloadVocabulary(t, vocabs[0])
	assert.Equal(t, services.ExerciseMatchCorrectProgressDelta, exerciseTranslationKnowledge(t, updated.Progress))
}

func TestCompleteMatchPairsWithWrong(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabs := exerciseSeedFiveVocabularies(t, user.ID)
	ex := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusInProgress, vocabs)

	// Pair vocab[0]'s original with vocab[1]'s translation twice -> wrong (2 misses)
	// for both vocab[0] and vocab[1]; the rest paired correctly.
	attempts := []map[string]any{
		{"first_card_id": vocabs[0].String() + ":original", "second_card_id": vocabs[1].String() + ":translation"},
		{"first_card_id": vocabs[0].String() + ":original", "second_card_id": vocabs[1].String() + ":translation"},
		{"first_card_id": vocabs[2].String() + ":original", "second_card_id": vocabs[2].String() + ":translation"},
		{"first_card_id": vocabs[3].String() + ":original", "second_card_id": vocabs[3].String() + ":translation"},
		{"first_card_id": vocabs[4].String() + ":original", "second_card_id": vocabs[4].String() + ":translation"},
	}

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/match-pairs/complete",
		map[string]any{"attempts": attempts})
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body services.MatchPairsCompleteResult
	testkit.DecodeJSON(t, rec, &body)

	assert.Equal(t, enums.ExerciseStatusFailed, body.Status)

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusFailed, stored.Status)

	// vocab[0] was wrong: progress is clamped at 0, while the full delta is recorded.
	wrongVocab := exerciseReloadVocabulary(t, vocabs[0])
	assert.Equal(t, 0, exerciseTranslationKnowledge(t, wrongVocab.Progress)) // clamp(0 - 15)
	wrongLink := exerciseLink(t, ex.ID, vocabs[0])
	require.NotNil(t, wrongLink.Result)
	assert.Equal(t, services.ExerciseVocabularyResultWrong, *wrongLink.Result)
	require.NotNil(t, wrongLink.ProgressDelta)
	assert.Equal(t, services.ExerciseMatchWrongProgressDelta, *wrongLink.ProgressDelta)
}

// Empty attempts list → 400 invalid match pair results.
func TestCompleteMatchPairsEmptyAttempts(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabs := exerciseSeedFiveVocabularies(t, user.ID)
	ex := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusInProgress, vocabs)

	// Bind requires non-empty attempts; an empty slice fails binding → 400.
	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/match-pairs/complete",
		map[string]any{"attempts": []any{}})
	testkit.RequireStatus(t, rec, http.StatusBadRequest)
}

// Not in progress → 409.
func TestCompleteMatchPairsNotInProgress(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabs := exerciseSeedFiveVocabularies(t, user.ID)
	ex := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusCompleted, vocabs)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/match-pairs/complete",
		map[string]any{"attempts": exerciseMatchAttempts(vocabs)})
	testkit.RequireStatus(t, rec, http.StatusConflict)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "exercise is not in progress", body["error"])
}

func TestCompleteMatchPairsOwnershipIsolation(t *testing.T) {
	testkit.Truncate(t)

	userA := testkit.CreateUser(t, testkit.WithName("A"))
	userB := testkit.CreateUser(t, testkit.WithName("B"))

	vocabs := exerciseSeedFiveVocabularies(t, userB.ID)
	ex := exerciseSeedMatchPairsExercise(t, userB.ID, enums.ExerciseStatusInProgress, vocabs)

	rec := testkit.AuthedRequest(t, userA, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/match-pairs/complete",
		map[string]any{"attempts": exerciseMatchAttempts(vocabs)})
	testkit.RequireStatus(t, rec, http.StatusNotFound)

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusInProgress, stored.Status, "B's exercise must be untouched")
}
