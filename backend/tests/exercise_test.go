package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

// exerciseLink loads the vocabulary_exercises link row for an exercise+vocabulary.
func exerciseLink(t *testing.T, exerciseID, vocabularyID uuid.UUID) models.ExerciseVocabulary {
	t.Helper()
	var link models.ExerciseVocabulary
	require.NoError(t, db.DB.
		Where("exercise_id = ? AND vocabulary_id = ?", exerciseID, vocabularyID).
		First(&link).Error)
	return link
}

// exerciseRawAuthedRequest issues an in-process request with a raw (possibly
// malformed) JSON body, carrying the user's auth cookie.
func exerciseRawAuthedRequest(t *testing.T, user models.User, method, path, rawBody string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(rawBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(testkit.AuthCookie(user))

	rec := httptest.NewRecorder()
	testkit.Router().ServeHTTP(rec, req)
	return rec
}

// ===========================================================================
// GET /api/exercises (GetExercises)
// ===========================================================================

func TestGetExercisesRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/exercises", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetExercisesEmpty(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises", nil)
	require.Equal(t, http.StatusOK, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

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
	require.Equal(t, http.StatusOK, rec.Code)

	var body services.ExerciseListResponse
	testkit.DecodeJSON(t, rec, &body)
	assert.Empty(t, body.Data, "not-started exercises must be excluded")
}

func TestGetExercisesInvalidPagination(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	// page_size > 1000 triggers ErrInvalidPageSize → 400.
	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises?page_size=5000", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code)

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
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetExercisesByIDsHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex1 := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusCompleted, vocab.ID)
	ex2 := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicReversed, enums.ExerciseStatusFailed, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodGet,
		"/api/exercises/by-ids?ids="+ex1.ID.String()+","+ex2.ID.String(), nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

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
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "ids parameter is required", body["error"])
}

func TestGetExercisesByIDsInvalidUUID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises/by-ids?ids=not-a-uuid", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code)

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
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
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
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

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
	require.Equal(t, http.StatusOK, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

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
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRandomExerciseNoVocabulary(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/random", nil)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body=%s", rec.Body.String())

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
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body=%s", rec.Body.String())

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, services.ErrAllVocabularyMastered.Error(), body["error"])
}

func TestRandomExerciseHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	// A single eligible vocabulary supports typed and character-building exercises.
	_ = exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/random", nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		ExerciseID     uuid.UUID                    `json:"exercise_id"`
		Type           enums.ExerciseType           `json:"type"`
		QuestionWord   string                       `json:"question_word"`
		Language       enums.Language               `json:"language"`
		AnswerLanguage enums.Language               `json:"answer_language"`
		Options        []string                     `json:"options"`
		Cards          []services.ExerciseMatchCard `json:"cards"`
	}
	testkit.DecodeJSON(t, rec, &body)

	require.NotEqual(t, uuid.Nil, body.ExerciseID)
	// Choice and match exercises need additional vocabulary; typed and character
	// exercises only need the correct word pair.
	assert.Contains(t, []enums.ExerciseType{
		enums.ExerciseTypeBasicDirect,
		enums.ExerciseTypeBasicReversed,
		enums.ExerciseTypeCharactersDirect,
		enums.ExerciseTypeCharactersReversed,
	}, body.Type)

	// DB side effect: the exercise exists, is in progress and belongs to user.
	stored := exerciseReload(t, body.ExerciseID)
	assert.Equal(t, user.ID, stored.UserID)
	assert.Equal(t, enums.ExerciseStatusInProgress, stored.Status)
	require.NotNil(t, stored.StartedAt)

	// Question word matches expected direction. Words are seeded with their raw
	// casing directly in the DB (no service-level normalization here).
	if body.Type == enums.ExerciseTypeBasicReversed || body.Type == enums.ExerciseTypeCharactersReversed {
		assert.Equal(t, "Hund", body.QuestionWord)
		assert.Equal(t, enums.LanguageDe, body.Language)
		assert.Equal(t, enums.LanguageEn, body.AnswerLanguage)
		if body.Type == enums.ExerciseTypeCharactersReversed {
			assert.ElementsMatch(t, []string{"d", "o", "g"}, body.Options)
		}
	} else {
		assert.Equal(t, "dog", body.QuestionWord)
		assert.Equal(t, enums.LanguageEn, body.Language)
		assert.Equal(t, enums.LanguageDe, body.AnswerLanguage)
		if body.Type == enums.ExerciseTypeCharactersDirect {
			assert.ElementsMatch(t, []string{"H", "u", "n", "d"}, body.Options)
		}
	}
}

// ===========================================================================
// POST /api/exercises/:id/verify (VerifyExercise)
// ===========================================================================

func TestVerifyExerciseRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/exercises/"+uuid.New().String()+"/verify",
		map[string]any{"answer": "Hund"})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestVerifyExerciseInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/not-a-uuid/verify",
		map[string]any{"answer": "Hund"})
	require.Equal(t, http.StatusBadRequest, rec.Code)

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
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "answer is required", body["error"])
}

func TestVerifyExerciseMalformedJSON(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := exerciseRawAuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", "}{ not json")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVerifyExerciseNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+uuid.New().String()+"/verify", map[string]any{"answer": "Hund"})
	require.Equal(t, http.StatusNotFound, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		Result        string `json:"result"`
		CorrectAnswer string `json:"correct_answer"`
		Knowledge     int    `json:"knowledge"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, rec, &body)

	assert.Equal(t, "correct", body.Result)
	assert.Equal(t, "Hund", body.CorrectAnswer)
	assert.Equal(t, services.ExerciseCompleteProgressDelta, body.ProgressDelta)
	assert.Equal(t, services.ExerciseCompleteProgressDelta, body.Knowledge) // 0 + 15

	// DB side effects.
	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusCompleted, stored.Status)
	require.NotNil(t, stored.FinishedAt)

	updatedVocab := exerciseReloadVocabulary(t, vocab.ID)
	assert.Equal(t, services.ExerciseCompleteProgressDelta, exerciseTranslationKnowledge(t, updatedVocab.Progress))

	link := exerciseLink(t, ex.ID, vocab.ID)
	require.NotNil(t, link.Result)
	assert.Equal(t, services.ExerciseVocabularyResultCorrect, *link.Result)
	require.NotNil(t, link.ProgressDelta)
	assert.Equal(t, services.ExerciseCompleteProgressDelta, *link.ProgressDelta)
	require.NotNil(t, link.AnsweredAt)
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
	order := []int{6, 0, 5, 2, 1, 4, 3}
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

func TestVerifyExerciseWrongAnswer(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "completely-wrong"})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		Result        string `json:"result"`
		CorrectAnswer string `json:"correct_answer"`
		Knowledge     int    `json:"knowledge"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, rec, &body)

	assert.Equal(t, "wrong", body.Result)
	assert.Equal(t, "Hund", body.CorrectAnswer)
	assert.Equal(t, services.ExerciseFailProgressDelta, body.ProgressDelta)
	assert.Equal(t, 0, body.Knowledge) // clamp(0 - 20) = 0

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusFailed, stored.Status)

	link := exerciseLink(t, ex.ID, vocab.ID)
	require.NotNil(t, link.Result)
	assert.Equal(t, services.ExerciseVocabularyResultWrong, *link.Result)
}

// A one-character typo is treated as "almost" correct (Levenshtein within threshold).
func TestVerifyExerciseAlmostAnswer(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "Hand"}) // 1 edit from "hund"
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		Result        string `json:"result"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, rec, &body)

	assert.Equal(t, "almost", body.Result)
	assert.Equal(t, services.ExerciseAlmostCorrectProgressDelta, body.ProgressDelta)

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusCompleted, stored.Status)
}

// Verifying an exercise that is not in progress returns 409 Conflict.
func TestVerifyExerciseNotInProgress(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusCompleted, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "Hund"})
	require.Equal(t, http.StatusConflict, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "exercise is not in progress", body["error"])
}

// Verifying a match/pairs exercise via the typed endpoint is a 400.
func TestVerifyExerciseMatchPairsRejected(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabs := exerciseSeedFiveVocabularies(t, user.ID)
	ex := exerciseSeedMatchPairsExercise(t, user.ID, enums.ExerciseStatusInProgress, vocabs)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "A"})
	require.Equal(t, http.StatusBadRequest, rec.Code)

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
	require.Equal(t, http.StatusNotFound, rec.Code)

	// B's exercise is untouched.
	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusInProgress, stored.Status)
}

// Choice exercise: a correct selection completes; a valid-but-wrong option fails.
func TestVerifyExerciseChoiceCorrect(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	correct := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	d1 := exerciseSeedVocabulary(t, user.ID, "cat", "Katze", enums.LanguageEn, enums.LanguageDe)
	d2 := exerciseSeedVocabulary(t, user.ID, "bird", "Vogel", enums.LanguageEn, enums.LanguageDe)
	d3 := exerciseSeedVocabulary(t, user.ID, "fish", "Fisch", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedChoiceExercise(t, user.ID, enums.ExerciseTypeChoiceDirect, enums.ExerciseStatusInProgress,
		[]uuid.UUID{correct.ID, d1.ID, d2.ID, d3.ID})

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+ex.ID.String()+"/verify", map[string]any{"answer": "Hund"})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		Result        string `json:"result"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "correct", body.Result)
	assert.Equal(t, services.ExerciseChoiceCompleteProgressDelta, body.ProgressDelta)

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusCompleted, stored.Status)
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
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		Result        string `json:"result"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "wrong", body.Result)
	assert.Equal(t, services.ExerciseChoiceFailProgressDelta, body.ProgressDelta)

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusFailed, stored.Status)
}

// ===========================================================================
// POST /api/exercises/:id/ignore (IgnoreExercise)
// ===========================================================================

func TestIgnoreExerciseRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/exercises/"+uuid.New().String()+"/ignore", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestIgnoreExerciseInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/not-a-uuid/ignore", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid exercise id", body["error"])
}

func TestIgnoreExerciseNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+uuid.New().String()+"/ignore", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "exercise not found", body["error"])
}

func TestIgnoreExerciseHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+ex.ID.String()+"/ignore", nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "ignored", body["status"])

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusIgnored, stored.Status)
	require.NotNil(t, stored.FinishedAt)
}

// Ignoring an already-finished exercise → 409 Conflict.
func TestIgnoreExerciseNotInProgress(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocab := exerciseSeedVocabulary(t, user.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusCompleted, vocab.ID)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+ex.ID.String()+"/ignore", nil)
	require.Equal(t, http.StatusConflict, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "exercise is not in progress", body["error"])
}

func TestIgnoreExerciseOwnershipIsolation(t *testing.T) {
	testkit.Truncate(t)

	userA := testkit.CreateUser(t, testkit.WithName("A"))
	userB := testkit.CreateUser(t, testkit.WithName("B"))

	vocab := exerciseSeedVocabulary(t, userB.ID, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	ex := exerciseSeedExercise(t, userB.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocab.ID)

	rec := testkit.AuthedRequest(t, userA, http.MethodPost, "/api/exercises/"+ex.ID.String()+"/ignore", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)

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
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCompleteMatchPairsInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/not-a-uuid/match-pairs/complete",
		map[string]any{"attempts": []any{map[string]any{"first_card_id": "a", "second_card_id": "b"}}})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid exercise id", body["error"])
}

func TestCompleteMatchPairsMissingBody(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/exercises/"+uuid.New().String()+"/match-pairs/complete", map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

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
	require.Equal(t, http.StatusNotFound, rec.Code)

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
	require.Equal(t, http.StatusBadRequest, rec.Code)

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
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

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
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body services.MatchPairsCompleteResult
	testkit.DecodeJSON(t, rec, &body)

	assert.Equal(t, enums.ExerciseStatusFailed, body.Status)

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusFailed, stored.Status)

	// vocab[0] was wrong: progress delta is the fail delta (clamped at 0).
	wrongVocab := exerciseReloadVocabulary(t, vocabs[0])
	assert.Equal(t, 0, exerciseTranslationKnowledge(t, wrongVocab.Progress)) // clamp(0 - 10)
	wrongLink := exerciseLink(t, ex.ID, vocabs[0])
	require.NotNil(t, wrongLink.Result)
	assert.Equal(t, services.ExerciseVocabularyResultWrong, *wrongLink.Result)
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
	require.Equal(t, http.StatusBadRequest, rec.Code)
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
	require.Equal(t, http.StatusConflict, rec.Code)

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
	require.Equal(t, http.StatusNotFound, rec.Code)

	stored := exerciseReload(t, ex.ID)
	assert.Equal(t, enums.ExerciseStatusInProgress, stored.Status, "B's exercise must be untouched")
}
