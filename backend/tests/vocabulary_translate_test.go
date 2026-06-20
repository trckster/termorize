package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/testkit"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Local helpers (unexported, prefixed `vocab` to avoid clashes with other test
// files in the `tests` package).
// ---------------------------------------------------------------------------

// vocabCreatePayload returns a valid CreateVocabulary request body.
func vocabCreatePayload() map[string]any {
	return map[string]any{
		"original":             "dog",
		"translation":          "Hund",
		"original_language":    "en",
		"translation_language": "de",
	}
}

// vocabTranslatePayload returns a valid Translate request body.
func vocabTranslatePayload() map[string]any {
	return map[string]any{
		"from_word":     "dog",
		"from_language": "en",
		"to_language":   "de",
	}
}

// vocabRawAuthedRequest issues an in-process request with a raw (possibly
// invalid) JSON body and the given user's auth cookie, so bind-level
// (non-validation) errors can be exercised.
func vocabRawAuthedRequest(t *testing.T, user models.User, method, path, rawBody string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(rawBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(testkit.AuthCookie(user))

	rec := httptest.NewRecorder()
	testkit.Router().ServeHTTP(rec, req)
	return rec
}

// vocabCountForUser returns the number of non-soft-deleted vocabulary rows for a
// user, queried directly against the DB.
func vocabCountForUser(t *testing.T, userID uint) int64 {
	t.Helper()
	var count int64
	err := db.DB.Model(&models.Vocabulary{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&count).Error
	require.NoError(t, err)
	return count
}

// vocabFindByID loads a vocabulary row (including soft-deleted) by ID.
func vocabFindByID(t *testing.T, id uuid.UUID) models.Vocabulary {
	t.Helper()
	var v models.Vocabulary
	err := db.DB.Where("id = ?", id).First(&v).Error
	require.NoError(t, err)
	return v
}

// vocabCreateForUser creates a vocabulary item for a user via the service path
// (through the API) and returns its ID. The Google mock is not used by this
// endpoint, so no mocking is required.
func vocabCreateForUser(t *testing.T, user models.User, payload map[string]any) uuid.UUID {
	t.Helper()
	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary", payload)
	require.Equal(t, http.StatusCreated, rec.Code, "vocabCreateForUser body=%s", rec.Body.String())

	var v models.Vocabulary
	testkit.DecodeJSON(t, rec, &v)
	require.NotEqual(t, uuid.Nil, v.ID)
	return v.ID
}

// vocabSeedTranslation inserts a Word pair and a non-user Translation directly,
// returning the translation ID. Used to exercise CreateVocabularyByTranslation.
func vocabSeedTranslation(t *testing.T, original, translated string, fromLang, toLang enums.Language) uuid.UUID {
	t.Helper()

	originalWord := models.Word{Word: original, Language: fromLang}
	require.NoError(t, db.DB.Create(&originalWord).Error)

	translatedWord := models.Word{Word: translated, Language: toLang}
	require.NoError(t, db.DB.Create(&translatedWord).Error)

	translation := models.Translation{
		OriginalID:    originalWord.ID,
		TranslationID: translatedWord.ID,
		Source:        enums.TranslationSourceGoogle,
	}
	require.NoError(t, db.DB.Create(&translation).Error)

	return translation.ID
}

// ===========================================================================
// GET /api/vocabulary (GetVocabulary)
// ===========================================================================

func TestGetVocabularyRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/vocabulary", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetVocabularyEmpty(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/vocabulary", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data       []models.Vocabulary `json:"data"`
		Pagination struct {
			Page       int   `json:"page"`
			PageSize   int   `json:"page_size"`
			Total      int64 `json:"total"`
			TotalPages int   `json:"total_pages"`
		} `json:"pagination"`
	}
	testkit.DecodeJSON(t, rec, &body)

	assert.Empty(t, body.Data)
	assert.Equal(t, 1, body.Pagination.Page)
	assert.Equal(t, 50, body.Pagination.PageSize) // controller default
	assert.Equal(t, int64(0), body.Pagination.Total)
	assert.Equal(t, 0, body.Pagination.TotalPages)
}

func TestGetVocabularyReturnsItems(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabCreateForUser(t, user, vocabCreatePayload())

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/vocabulary", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data       []models.Vocabulary `json:"data"`
		Pagination struct {
			Total int64 `json:"total"`
		} `json:"pagination"`
	}
	testkit.DecodeJSON(t, rec, &body)

	require.Len(t, body.Data, 1)
	assert.Equal(t, int64(1), body.Pagination.Total)
	require.NotNil(t, body.Data[0].Translation)
	require.NotNil(t, body.Data[0].Translation.Original)
	require.NotNil(t, body.Data[0].Translation.Translation)
	assert.Equal(t, "dog", body.Data[0].Translation.Original.Word)
	// Single-word translations are lowercased by NormalizeWordCasingForLanguage.
	assert.Equal(t, "hund", body.Data[0].Translation.Translation.Word)
}

func TestGetVocabularySearchFiltersResults(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabCreateForUser(t, user, vocabCreatePayload()) // dog / Hund
	vocabCreateForUser(t, user, map[string]any{
		"original":             "cat",
		"translation":          "Katze",
		"original_language":    "en",
		"translation_language": "de",
	})

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/vocabulary?search=cat", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []models.Vocabulary `json:"data"`
	}
	testkit.DecodeJSON(t, rec, &body)

	require.Len(t, body.Data, 1)
	assert.Equal(t, "cat", body.Data[0].Translation.Original.Word)
}

func TestGetVocabularyInvalidPagination(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	// page_size > 1000 triggers ErrInvalidPageSize → 400.
	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/vocabulary?page_size=5000", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Contains(t, body, "error")
}

func TestGetVocabularyOwnershipIsolation(t *testing.T) {
	testkit.Truncate(t)

	userA := testkit.CreateUser(t, testkit.WithName("A"))
	userB := testkit.CreateUser(t, testkit.WithName("B"))

	vocabCreateForUser(t, userB, vocabCreatePayload()) // belongs to B only

	rec := testkit.AuthedRequest(t, userA, http.MethodGet, "/api/vocabulary", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []models.Vocabulary `json:"data"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Empty(t, body.Data, "user A must not see user B's vocabulary")
}

// ===========================================================================
// POST /api/vocabulary (CreateVocabulary)
// ===========================================================================

func TestCreateVocabularyRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/vocabulary", vocabCreatePayload())
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateVocabularyHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary", vocabCreatePayload())
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var v models.Vocabulary
	testkit.DecodeJSON(t, rec, &v)

	// Response shape.
	require.NotEqual(t, uuid.Nil, v.ID)
	require.NotNil(t, v.Translation)
	require.NotNil(t, v.Translation.Original)
	require.NotNil(t, v.Translation.Translation)
	assert.Equal(t, "dog", v.Translation.Original.Word)
	// Single-word translations are lowercased by NormalizeWordCasingForLanguage.
	assert.Equal(t, "hund", v.Translation.Translation.Word)
	assert.Equal(t, enums.TranslationSourceUser, v.Translation.Source)
	require.Len(t, v.Progress, 1)
	assert.Equal(t, 0, v.Progress[0].Knowledge)
	assert.Equal(t, enums.KnowledgeTypeTranslation, v.Progress[0].Type)

	// DB side effect: a row exists for this user.
	assert.Equal(t, int64(1), vocabCountForUser(t, user.ID))
	stored := vocabFindByID(t, v.ID)
	assert.Equal(t, user.ID, stored.UserID)
	assert.Nil(t, stored.DeletedAt)
}

func TestCreateVocabularyDuplicateConflicts(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	vocabCreateForUser(t, user, vocabCreatePayload())

	// Creating the same pair again → 409 Conflict.
	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary", vocabCreatePayload())
	require.Equal(t, http.StatusConflict, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "vocabulary already exists", body["error"])

	// No duplicate row created.
	assert.Equal(t, int64(1), vocabCountForUser(t, user.ID))
}

func TestCreateVocabularyInvalidJSON(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := vocabRawAuthedRequest(t, user, http.MethodPost, "/api/vocabulary", "}{ not json")
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Contains(t, body, "error")
}

func TestCreateVocabularyMissingFields(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary", map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	require.NotEmpty(t, body.Errors)
	assert.Equal(t, "required", body.Errors["Original"])
}

func TestCreateVocabularyInvalidLanguageEnum(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	payload := vocabCreatePayload()
	payload["original_language"] = "xx" // not a valid Language

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary", payload)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "enum", body.Errors["OriginalLanguage"])
}

func TestCreateVocabularySameSourceAndTargetLanguage(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	payload := vocabCreatePayload()
	payload["translation_language"] = "en" // equals original_language → nefield violation

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary", payload)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "nefield", body.Errors["TranslationLanguage"])
}

// ===========================================================================
// POST /api/vocabulary/translation (CreateVocabularyByTranslation)
// ===========================================================================

func TestCreateVocabularyByTranslationRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/vocabulary/translation", map[string]any{
		"translation_id": uuid.New().String(),
	})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateVocabularyByTranslationHappyPath(t *testing.T) {
	testkit.Truncate(t)

	// Google mock guards against any accidental network call (this path should
	// not actually call Google, but be safe).
	testkit.MockGoogleTranslate(t, nil)

	user := testkit.CreateUser(t)
	translationID := vocabSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary/translation", map[string]any{
		"translation_id": translationID.String(),
	})
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var v models.Vocabulary
	testkit.DecodeJSON(t, rec, &v)
	require.NotEqual(t, uuid.Nil, v.ID)
	require.NotNil(t, v.Translation)
	assert.Equal(t, translationID, v.Translation.ID)
	require.NotNil(t, v.Translation.Original)
	assert.Equal(t, "dog", v.Translation.Original.Word)

	// DB side effect.
	assert.Equal(t, int64(1), vocabCountForUser(t, user.ID))
	stored := vocabFindByID(t, v.ID)
	assert.Equal(t, translationID, stored.TranslationID)
}

func TestCreateVocabularyByTranslationDuplicateConflicts(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	translationID := vocabSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)

	first := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary/translation", map[string]any{
		"translation_id": translationID.String(),
	})
	require.Equal(t, http.StatusCreated, first.Code)

	second := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary/translation", map[string]any{
		"translation_id": translationID.String(),
	})
	require.Equal(t, http.StatusConflict, second.Code)

	var body map[string]any
	testkit.DecodeJSON(t, second, &body)
	assert.Equal(t, "vocabulary already exists", body["error"])
}

// TestCreateVocabularyByTranslationNotFound verifies that a non-existent
// translation_id returns 404: the service maps gorm.ErrRecordNotFound to
// ErrTranslationNotFound, which the controller's TranslationNotFoundError check
// translates into a 404.
func TestCreateVocabularyByTranslationNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary/translation", map[string]any{
		"translation_id": uuid.New().String(),
	})
	require.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
}

func TestCreateVocabularyByTranslationInvalidJSON(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := vocabRawAuthedRequest(t, user, http.MethodPost, "/api/vocabulary/translation", "}{ not json")
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Contains(t, body, "error")
}

func TestCreateVocabularyByTranslationMissingID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary/translation", map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "required", body.Errors["TranslationID"])
}

// ===========================================================================
// DELETE /api/vocabulary/:id (DeleteVocabulary)
// ===========================================================================

func TestDeleteVocabularyRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodDelete, "/api/vocabulary/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDeleteVocabularyHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	id := vocabCreateForUser(t, user, vocabCreatePayload())

	rec := testkit.AuthedRequest(t, user, http.MethodDelete, "/api/vocabulary/"+id.String(), nil)
	require.Equal(t, http.StatusOK, rec.Code)

	// Soft-delete semantics: row still exists but deleted_at is set; it no longer
	// counts as active and disappears from GET.
	assert.Equal(t, int64(0), vocabCountForUser(t, user.ID))
	stored := vocabFindByID(t, id)
	assert.NotNil(t, stored.DeletedAt, "vocabulary should be soft-deleted")

	listRec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/vocabulary", nil)
	require.Equal(t, http.StatusOK, listRec.Code)
	var body struct {
		Data []models.Vocabulary `json:"data"`
	}
	testkit.DecodeJSON(t, listRec, &body)
	assert.Empty(t, body.Data)
}

func TestDeleteVocabularyInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodDelete, "/api/vocabulary/not-a-uuid", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid vocabulary ID", body["error"])
}

func TestDeleteVocabularyNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodDelete, "/api/vocabulary/"+uuid.New().String(), nil)
	require.Equal(t, http.StatusNotFound, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "vocabulary item not found", body["error"])
}

func TestDeleteVocabularyOwnershipIsolation(t *testing.T) {
	testkit.Truncate(t)

	userA := testkit.CreateUser(t, testkit.WithName("A"))
	userB := testkit.CreateUser(t, testkit.WithName("B"))

	// Item belongs to B.
	id := vocabCreateForUser(t, userB, vocabCreatePayload())

	// A tries to delete B's item → 404 (scoped by user_id), and B's item survives.
	rec := testkit.AuthedRequest(t, userA, http.MethodDelete, "/api/vocabulary/"+id.String(), nil)
	require.Equal(t, http.StatusNotFound, rec.Code)

	assert.Equal(t, int64(1), vocabCountForUser(t, userB.ID))
	stored := vocabFindByID(t, id)
	assert.Nil(t, stored.DeletedAt, "user A must not be able to delete user B's vocabulary")
}

// ===========================================================================
// POST /api/translate (Translate)
// ===========================================================================

func TestTranslateRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/translate", vocabTranslatePayload())
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestTranslateHappyPath(t *testing.T) {
	testkit.Truncate(t)

	testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
		TranslateFunc: func(text, src, dst string) (string, error) {
			assert.Equal(t, "dog", text)
			assert.Equal(t, "en", src)
			assert.Equal(t, "de", dst)
			return "hund", nil
		},
	})

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/translate", vocabTranslatePayload())
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		ID          uuid.UUID               `json:"id"`
		Translation string                  `json:"translation"`
		Source      enums.TranslationSource `json:"source"`
	}
	testkit.DecodeJSON(t, rec, &body)

	assert.NotEqual(t, uuid.Nil, body.ID)
	assert.Equal(t, "hund", body.Translation)
	assert.Equal(t, enums.TranslationSourceGoogle, body.Source)

	// DB side effect: a google-sourced translation row was persisted.
	var translation models.Translation
	require.NoError(t, db.DB.Where("id = ?", body.ID).First(&translation).Error)
	assert.Equal(t, enums.TranslationSourceGoogle, translation.Source)
}

func TestTranslateUsesExistingTranslation(t *testing.T) {
	testkit.Truncate(t)

	// Seed an existing google translation; the handler should return it WITHOUT
	// calling Google again. The mock fails the test if invoked.
	translationID := vocabSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)

	testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
		TranslateFunc: func(text, src, dst string) (string, error) {
			t.Fatalf("Translate should not be called when a translation already exists")
			return "", nil
		},
	})

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/translate", vocabTranslatePayload())
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		ID          uuid.UUID `json:"id"`
		Translation string    `json:"translation"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, translationID, body.ID)
	assert.Equal(t, "Hund", body.Translation)
}

func TestTranslateGoogleFailure(t *testing.T) {
	testkit.Truncate(t)

	testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
		TranslateFunc: func(text, src, dst string) (string, error) {
			return "", assert.AnError
		},
	})

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/translate", vocabTranslatePayload())
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "Internal error", body["error"])
}

func TestTranslateInvalidJSON(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := vocabRawAuthedRequest(t, user, http.MethodPost, "/api/translate", "}{ not json")
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Contains(t, body, "error")
}

func TestTranslateMissingFields(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/translate", map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	require.NotEmpty(t, body.Errors)
	assert.Equal(t, "required", body.Errors["FromWord"])
}

func TestTranslateInvalidLanguageEnum(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	payload := vocabTranslatePayload()
	payload["from_language"] = "xx"

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/translate", payload)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "enum", body.Errors["FromLanguage"])
}

func TestTranslateSameSourceAndTargetLanguage(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	payload := vocabTranslatePayload()
	payload["to_language"] = "en" // equals from_language → nefield violation

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/translate", payload)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "nefield", body.Errors["ToLanguage"])
}
