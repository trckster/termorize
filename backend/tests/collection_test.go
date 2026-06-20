package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/models"
	"termorize/src/testkit"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Local helpers (unexported, prefixed `collection` to avoid clashes with other
// test files in the shared `tests` package).
// ---------------------------------------------------------------------------

// collectionRawAuthedRequest issues an in-process request with a raw (possibly
// invalid) JSON body and the given user's auth cookie, so bind-level errors can
// be exercised.
func collectionRawAuthedRequest(t *testing.T, user models.User, method, path, rawBody string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(rawBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(testkit.AuthCookie(user))

	rec := httptest.NewRecorder()
	testkit.Router().ServeHTTP(rec, req)
	return rec
}

// collectionSeed inserts a collection row directly via db.DB and returns it.
func collectionSeed(t *testing.T, title string, ownerID *uint, isAdmin, isPublished bool) models.Collection {
	t.Helper()

	token, err := uuid.NewRandom()
	require.NoError(t, err)

	collection := models.Collection{
		Title:       title,
		OwnerID:     ownerID,
		IsAdmin:     isAdmin,
		IsPublished: isPublished,
		InviteToken: token.String(),
	}
	require.NoError(t, db.DB.Create(&collection).Error)
	return collection
}

// collectionSeedTranslation inserts a Word pair and a Translation, returning the
// translation ID.
func collectionSeedTranslation(t *testing.T, original, translated string, fromLang, toLang enums.Language) uuid.UUID {
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

// collectionLink links a translation to a collection at the given position.
func collectionLink(t *testing.T, collectionID, translationID uuid.UUID, position int) {
	t.Helper()
	link := models.CollectionTranslation{
		CollectionID:  collectionID,
		TranslationID: translationID,
		Position:      position,
	}
	require.NoError(t, db.DB.Create(&link).Error)
}

// collectionPosition reads the stored position of a translation in a collection.
func collectionPosition(t *testing.T, collectionID, translationID uuid.UUID) int {
	t.Helper()
	var link models.CollectionTranslation
	require.NoError(t, db.DB.
		Where("collection_id = ? AND translation_id = ?", collectionID, translationID).
		First(&link).Error)
	return link.Position
}

// collectionTranslationCount counts collection_translations rows for a collection.
func collectionTranslationCount(t *testing.T, collectionID uuid.UUID) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.DB.Model(&models.CollectionTranslation{}).
		Where("collection_id = ?", collectionID).
		Count(&count).Error)
	return count
}

// collectionFindByID loads a collection (including soft-deleted) by ID.
func collectionFindByID(t *testing.T, id uuid.UUID) models.Collection {
	t.Helper()
	var c models.Collection
	require.NoError(t, db.DB.Where("id = ?", id).First(&c).Error)
	return c
}

// collectionMemberCount counts members of a collection.
func collectionMemberCount(t *testing.T, collectionID uuid.UUID) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.DB.Model(&models.CollectionMember{}).
		Where("collection_id = ?", collectionID).
		Count(&count).Error)
	return count
}

func uintPtr(v uint) *uint { return &v }

// ===========================================================================
// GET /api/collections (GetCollections)
// ===========================================================================

func TestGetCollectionsRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/collections", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetCollectionsEmpty(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data       []map[string]any `json:"data"`
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
	assert.Equal(t, 50, body.Pagination.PageSize)
	assert.Equal(t, int64(0), body.Pagination.Total)
}

func TestGetCollectionsReturnsOwnedAndPublishedAdmin(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	other := testkit.CreateUser(t, testkit.WithName("Other"))

	owned := collectionSeed(t, "Mine", uintPtr(user.ID), false, true)
	publishedAdmin := collectionSeed(t, "GlobalPub", nil, true, true)
	// Should NOT be visible: another user's private collection and unpublished admin.
	collectionSeed(t, "Theirs", uintPtr(other.ID), false, true)
	collectionSeed(t, "GlobalDraft", nil, true, false)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []struct {
			ID    uuid.UUID `json:"id"`
			Title string    `json:"title"`
		} `json:"data"`
		Pagination struct {
			Total int64 `json:"total"`
		} `json:"pagination"`
	}
	testkit.DecodeJSON(t, rec, &body)

	ids := map[uuid.UUID]bool{}
	for _, d := range body.Data {
		ids[d.ID] = true
	}
	assert.True(t, ids[owned.ID], "owned collection should be visible")
	assert.True(t, ids[publishedAdmin.ID], "published admin collection should be visible")
	assert.Equal(t, int64(2), body.Pagination.Total)
}

func TestGetCollectionsAdminViewerSeesAll(t *testing.T) {
	testkit.Truncate(t)

	admin := testkit.CreateUser(t, testkit.WithAdmin())
	other := testkit.CreateUser(t, testkit.WithName("Other"))

	collectionSeed(t, "Theirs", uintPtr(other.ID), false, true)
	collectionSeed(t, "GlobalDraft", nil, true, false)

	rec := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/collections", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Pagination struct {
			Total int64 `json:"total"`
		} `json:"pagination"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, int64(2), body.Pagination.Total, "admin viewer sees everything")
}

func TestGetCollectionsSearchFilters(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collectionSeed(t, "Animals", uintPtr(user.ID), false, true)
	collectionSeed(t, "Food", uintPtr(user.ID), false, true)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections?search=anim", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	testkit.DecodeJSON(t, rec, &body)
	require.Len(t, body.Data, 1)
	assert.Equal(t, "Animals", body.Data[0].Title)
}

func TestGetCollectionsInvalidPagination(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections?page_size=5000", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Contains(t, body, "error")
}

func TestGetCollectionsLanguageFilter(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	enDe := collectionSeed(t, "EnDe", uintPtr(user.ID), false, true)
	enFr := collectionSeed(t, "EnFr", uintPtr(user.ID), false, true)

	tr1 := collectionSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	collectionLink(t, enDe.ID, tr1, 0)

	tr2 := collectionSeedTranslation(t, "cat", "chat", enums.LanguageEn, enums.LanguageFr)
	collectionLink(t, enFr.ID, tr2, 0)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections?languages=de", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	testkit.DecodeJSON(t, rec, &body)
	require.Len(t, body.Data, 1)
	assert.Equal(t, enDe.ID, body.Data[0].ID)
}

// ===========================================================================
// POST /api/collections (CreateCollection)
// ===========================================================================

func TestCreateCollectionRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/collections", map[string]any{"title": "X"})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateCollectionHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections", map[string]any{"title": "  My Collection  "})
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		ID          uuid.UUID `json:"id"`
		Title       string    `json:"title"`
		IsAdmin     bool      `json:"is_admin"`
		IsOwner     bool      `json:"is_owner"`
		IsPublished bool      `json:"is_published"`
		InviteToken string    `json:"invite_token"`
	}
	testkit.DecodeJSON(t, rec, &body)

	assert.NotEqual(t, uuid.Nil, body.ID)
	assert.Equal(t, "My Collection", body.Title, "title is trimmed")
	assert.False(t, body.IsAdmin)
	assert.True(t, body.IsOwner)
	assert.True(t, body.IsPublished, "user collections are published by default")
	assert.NotEmpty(t, body.InviteToken, "owner sees the invite token for non-admin collections")

	stored := collectionFindByID(t, body.ID)
	require.NotNil(t, stored.OwnerID)
	assert.Equal(t, user.ID, *stored.OwnerID)
}

func TestCreateCollectionMissingTitle(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections", map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "required", body.Errors["Title"])
}

func TestCreateCollectionBlankTitle(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	// A whitespace-only title passes the binding:required check but is rejected by
	// the service with ErrCollectionTitleRequired → 400.
	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections", map[string]any{"title": "   "})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "collection title can't be empty", body["error"])
}

func TestCreateCollectionInvalidJSON(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := collectionRawAuthedRequest(t, user, http.MethodPost, "/api/collections", "}{ not json")
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Contains(t, body, "error")
}

func TestCreateCollectionAdminByNonAdminForbidden(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections", map[string]any{
		"title":    "Global",
		"is_admin": true,
	})
	require.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "only admins can manage global collections", body["error"])
}

func TestCreateCollectionAdminByAdmin(t *testing.T) {
	testkit.Truncate(t)

	admin := testkit.CreateUser(t, testkit.WithAdmin())

	rec := testkit.AuthedRequest(t, admin, http.MethodPost, "/api/collections", map[string]any{
		"title":    "Global",
		"is_admin": true,
	})
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		ID          uuid.UUID `json:"id"`
		IsAdmin     bool      `json:"is_admin"`
		InviteToken string    `json:"invite_token"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.True(t, body.IsAdmin)
	assert.Empty(t, body.InviteToken, "admin collections do not expose an invite token")

	stored := collectionFindByID(t, body.ID)
	assert.True(t, stored.IsAdmin)
}

// ===========================================================================
// GET /api/collections/:id (GetCollection)
// ===========================================================================

func TestGetCollectionRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/collections/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetCollectionInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections/not-a-uuid", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid collection ID", body["error"])
}

func TestGetCollectionNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections/"+uuid.New().String(), nil)
	require.Equal(t, http.StatusNotFound, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "collection not found", body["error"])
}

func TestGetCollectionHappyPathWithTranslations(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)

	tr1 := collectionSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	tr2 := collectionSeedTranslation(t, "cat", "Katze", enums.LanguageEn, enums.LanguageDe)
	collectionLink(t, collection.ID, tr1, 0)
	collectionLink(t, collection.ID, tr2, 1)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections/"+collection.ID.String(), nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		ID               uuid.UUID `json:"id"`
		Title            string    `json:"title"`
		IsOwner          bool      `json:"is_owner"`
		TranslationCount int       `json:"translation_count"`
		InviteToken      string    `json:"invite_token"`
		Translations     []struct {
			ID       uuid.UUID `json:"id"`
			Original struct {
				Word string `json:"word"`
			} `json:"original"`
		} `json:"translations"`
	}
	testkit.DecodeJSON(t, rec, &body)

	assert.Equal(t, collection.ID, body.ID)
	assert.True(t, body.IsOwner)
	assert.Equal(t, 2, body.TranslationCount)
	require.Len(t, body.Translations, 2)
	// Ordered by position ASC.
	assert.Equal(t, tr1, body.Translations[0].ID)
	assert.Equal(t, tr2, body.Translations[1].ID)
	assert.NotEmpty(t, body.InviteToken, "owner of non-admin collection sees invite token")
}

func TestGetCollectionUnpublishedAdminHidden(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	// Unpublished admin collection, owned by nobody the user knows.
	collection := collectionSeed(t, "Draft", nil, true, false)

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections/"+collection.ID.String(), nil)
	require.Equal(t, http.StatusNotFound, rec.Code, "unpublished admin collection is not accessible to a normal user")
}

func TestGetCollectionMemberCanAccess(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	member := testkit.CreateUser(t, testkit.WithName("Member"))
	collection := collectionSeed(t, "Shared", uintPtr(owner.ID), false, true)

	require.NoError(t, db.DB.Create(&models.CollectionMember{
		CollectionID: collection.ID,
		UserID:       member.ID,
	}).Error)

	rec := testkit.AuthedRequest(t, member, http.MethodGet, "/api/collections/"+collection.ID.String(), nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		IsOwner     bool   `json:"is_owner"`
		InviteToken string `json:"invite_token"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.False(t, body.IsOwner, "member is not owner")
	assert.Empty(t, body.InviteToken, "non-owner does not see invite token")
}

// ===========================================================================
// PUT /api/collections/:id (UpdateCollection)
// ===========================================================================

func TestUpdateCollectionRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPut, "/api/collections/"+uuid.New().String(), map[string]any{"title": "X"})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateCollectionHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Old", uintPtr(user.ID), false, true)

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/collections/"+collection.ID.String(),
		map[string]any{"title": "New Title"})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		Title string `json:"title"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "New Title", body.Title)

	stored := collectionFindByID(t, collection.ID)
	assert.Equal(t, "New Title", stored.Title)
}

func TestUpdateCollectionInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/collections/not-a-uuid", map[string]any{"title": "X"})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid collection ID", body["error"])
}

func TestUpdateCollectionMissingTitle(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Old", uintPtr(user.ID), false, true)

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/collections/"+collection.ID.String(), map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "required", body.Errors["Title"])
}

func TestUpdateCollectionNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/collections/"+uuid.New().String(),
		map[string]any{"title": "X"})
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateCollectionForbiddenForNonOwner(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	member := testkit.CreateUser(t, testkit.WithName("Member"))
	collection := collectionSeed(t, "Shared", uintPtr(owner.ID), false, true)

	// Member can access but not edit.
	require.NoError(t, db.DB.Create(&models.CollectionMember{
		CollectionID: collection.ID,
		UserID:       member.ID,
	}).Error)

	rec := testkit.AuthedRequest(t, member, http.MethodPut, "/api/collections/"+collection.ID.String(),
		map[string]any{"title": "Hijacked"})
	require.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "you don't have access to this collection", body["error"])

	stored := collectionFindByID(t, collection.ID)
	assert.Equal(t, "Shared", stored.Title, "title unchanged")
}

// ===========================================================================
// DELETE /api/collections/:id (DeleteCollection)
// ===========================================================================

func TestDeleteCollectionRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodDelete, "/api/collections/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDeleteCollectionHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "ToDelete", uintPtr(user.ID), false, true)

	rec := testkit.AuthedRequest(t, user, http.MethodDelete, "/api/collections/"+collection.ID.String(), nil)
	require.Equal(t, http.StatusOK, rec.Code)

	// Soft-deleted: deleted_at set, no longer accessible.
	stored := collectionFindByID(t, collection.ID)
	assert.NotNil(t, stored.DeletedAt)

	getRec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections/"+collection.ID.String(), nil)
	assert.Equal(t, http.StatusNotFound, getRec.Code)
}

func TestDeleteCollectionInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodDelete, "/api/collections/not-a-uuid", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteCollectionNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodDelete, "/api/collections/"+uuid.New().String(), nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteCollectionForbiddenForNonOwner(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	member := testkit.CreateUser(t, testkit.WithName("Member"))
	collection := collectionSeed(t, "Shared", uintPtr(owner.ID), false, true)
	require.NoError(t, db.DB.Create(&models.CollectionMember{
		CollectionID: collection.ID,
		UserID:       member.ID,
	}).Error)

	rec := testkit.AuthedRequest(t, member, http.MethodDelete, "/api/collections/"+collection.ID.String(), nil)
	require.Equal(t, http.StatusForbidden, rec.Code)

	stored := collectionFindByID(t, collection.ID)
	assert.Nil(t, stored.DeletedAt, "non-owner must not delete")
}

// ===========================================================================
// POST /api/collections/:id/translations (AddCollectionTranslation)
// ===========================================================================

func collectionAddTranslationPayload() map[string]any {
	return map[string]any{
		"original":             "dog",
		"translation":          "Hund",
		"original_language":    "en",
		"translation_language": "de",
	}
}

func TestAddCollectionTranslationRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/collections/"+uuid.New().String()+"/translations",
		collectionAddTranslationPayload())
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAddCollectionTranslationHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/"+collection.ID.String()+"/translations",
		collectionAddTranslationPayload())
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		TranslationCount int `json:"translation_count"`
		Translations     []struct {
			Original struct {
				Word string `json:"word"`
			} `json:"original"`
		} `json:"translations"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, 1, body.TranslationCount)
	require.Len(t, body.Translations, 1)
	assert.Equal(t, "dog", body.Translations[0].Original.Word)

	assert.Equal(t, int64(1), collectionTranslationCount(t, collection.ID))
}

func TestAddCollectionTranslationAssignsIncreasingPositions(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)

	first := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/"+collection.ID.String()+"/translations",
		collectionAddTranslationPayload())
	require.Equal(t, http.StatusCreated, first.Code)

	second := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/"+collection.ID.String()+"/translations",
		map[string]any{
			"original":             "cat",
			"translation":          "Katze",
			"original_language":    "en",
			"translation_language": "de",
		})
	require.Equal(t, http.StatusCreated, second.Code)

	var body struct {
		Translations []struct {
			ID uuid.UUID `json:"id"`
		} `json:"translations"`
	}
	testkit.DecodeJSON(t, second, &body)
	require.Len(t, body.Translations, 2)

	// Positions are 0 and 1 in insertion order.
	assert.Equal(t, 0, collectionPosition(t, collection.ID, body.Translations[0].ID))
	assert.Equal(t, 1, collectionPosition(t, collection.ID, body.Translations[1].ID))
}

func TestAddCollectionTranslationInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/not-a-uuid/translations",
		collectionAddTranslationPayload())
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid collection ID", body["error"])
}

func TestAddCollectionTranslationMissingFields(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/"+collection.ID.String()+"/translations",
		map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "required", body.Errors["Original"])
}

func TestAddCollectionTranslationSameLanguage(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)

	payload := collectionAddTranslationPayload()
	payload["translation_language"] = "en" // equals original_language → nefield

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/"+collection.ID.String()+"/translations", payload)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "nefield", body.Errors["TranslationLanguage"])
}

func TestAddCollectionTranslationNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/"+uuid.New().String()+"/translations",
		collectionAddTranslationPayload())
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAddCollectionTranslationForbiddenForNonOwner(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	member := testkit.CreateUser(t, testkit.WithName("Member"))
	collection := collectionSeed(t, "Shared", uintPtr(owner.ID), false, true)
	require.NoError(t, db.DB.Create(&models.CollectionMember{
		CollectionID: collection.ID,
		UserID:       member.ID,
	}).Error)

	rec := testkit.AuthedRequest(t, member, http.MethodPost, "/api/collections/"+collection.ID.String()+"/translations",
		collectionAddTranslationPayload())
	require.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, int64(0), collectionTranslationCount(t, collection.ID))
}

// ===========================================================================
// DELETE /api/collections/:id/translations/:translationId (RemoveCollectionTranslation)
// ===========================================================================

func TestRemoveCollectionTranslationRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodDelete,
		"/api/collections/"+uuid.New().String()+"/translations/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRemoveCollectionTranslationHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)
	tr := collectionSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	collectionLink(t, collection.ID, tr, 0)
	require.Equal(t, int64(1), collectionTranslationCount(t, collection.ID))

	rec := testkit.AuthedRequest(t, user, http.MethodDelete,
		"/api/collections/"+collection.ID.String()+"/translations/"+tr.String(), nil)
	require.Equal(t, http.StatusOK, rec.Code)

	assert.Equal(t, int64(0), collectionTranslationCount(t, collection.ID))
}

func TestRemoveCollectionTranslationInvalidCollectionID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodDelete,
		"/api/collections/not-a-uuid/translations/"+uuid.New().String(), nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid collection ID", body["error"])
}

func TestRemoveCollectionTranslationInvalidTranslationID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)

	rec := testkit.AuthedRequest(t, user, http.MethodDelete,
		"/api/collections/"+collection.ID.String()+"/translations/not-a-uuid", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid translation ID", body["error"])
}

func TestRemoveCollectionTranslationNotFoundCollection(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodDelete,
		"/api/collections/"+uuid.New().String()+"/translations/"+uuid.New().String(), nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRemoveCollectionTranslationForbiddenForNonOwner(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	member := testkit.CreateUser(t, testkit.WithName("Member"))
	collection := collectionSeed(t, "Shared", uintPtr(owner.ID), false, true)
	tr := collectionSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	collectionLink(t, collection.ID, tr, 0)
	require.NoError(t, db.DB.Create(&models.CollectionMember{
		CollectionID: collection.ID,
		UserID:       member.ID,
	}).Error)

	rec := testkit.AuthedRequest(t, member, http.MethodDelete,
		"/api/collections/"+collection.ID.String()+"/translations/"+tr.String(), nil)
	require.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, int64(1), collectionTranslationCount(t, collection.ID), "link must survive")
}

// ===========================================================================
// PUT /api/collections/:id/translations/order (ReorderCollectionTranslations)
// ===========================================================================

func TestReorderCollectionTranslationsRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPut, "/api/collections/"+uuid.New().String()+"/translations/order",
		map[string]any{"translation_ids": []string{uuid.New().String()}})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestReorderCollectionTranslationsHappyPath(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)

	tr1 := collectionSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	tr2 := collectionSeedTranslation(t, "cat", "Katze", enums.LanguageEn, enums.LanguageDe)
	tr3 := collectionSeedTranslation(t, "bird", "Vogel", enums.LanguageEn, enums.LanguageDe)
	collectionLink(t, collection.ID, tr1, 0)
	collectionLink(t, collection.ID, tr2, 1)
	collectionLink(t, collection.ID, tr3, 2)

	// New order: tr3, tr1, tr2.
	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/collections/"+collection.ID.String()+"/translations/order",
		map[string]any{"translation_ids": []string{tr3.String(), tr1.String(), tr2.String()}})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	// DB positions reflect the new order.
	assert.Equal(t, 0, collectionPosition(t, collection.ID, tr3))
	assert.Equal(t, 1, collectionPosition(t, collection.ID, tr1))
	assert.Equal(t, 2, collectionPosition(t, collection.ID, tr2))

	// And the response (and a subsequent GET) is ordered accordingly.
	var body struct {
		Translations []struct {
			ID uuid.UUID `json:"id"`
		} `json:"translations"`
	}
	testkit.DecodeJSON(t, rec, &body)
	require.Len(t, body.Translations, 3)
	assert.Equal(t, tr3, body.Translations[0].ID)
	assert.Equal(t, tr1, body.Translations[1].ID)
	assert.Equal(t, tr2, body.Translations[2].ID)

	getRec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections/"+collection.ID.String(), nil)
	require.Equal(t, http.StatusOK, getRec.Code)
	var getBody struct {
		Translations []struct {
			ID uuid.UUID `json:"id"`
		} `json:"translations"`
	}
	testkit.DecodeJSON(t, getRec, &getBody)
	require.Len(t, getBody.Translations, 3)
	assert.Equal(t, tr3, getBody.Translations[0].ID)
	assert.Equal(t, tr1, getBody.Translations[1].ID)
	assert.Equal(t, tr2, getBody.Translations[2].ID)
}

func TestReorderCollectionTranslationsInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/collections/not-a-uuid/translations/order",
		map[string]any{"translation_ids": []string{uuid.New().String()}})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid collection ID", body["error"])
}

func TestReorderCollectionTranslationsMissingIDs(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/collections/"+collection.ID.String()+"/translations/order",
		map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "required", body.Errors["TranslationIDs"])
}

func TestReorderCollectionTranslationsNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPut, "/api/collections/"+uuid.New().String()+"/translations/order",
		map[string]any{"translation_ids": []string{uuid.New().String()}})
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestReorderCollectionTranslationsForbiddenForNonOwner(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	member := testkit.CreateUser(t, testkit.WithName("Member"))
	collection := collectionSeed(t, "Shared", uintPtr(owner.ID), false, true)
	tr := collectionSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	collectionLink(t, collection.ID, tr, 0)
	require.NoError(t, db.DB.Create(&models.CollectionMember{
		CollectionID: collection.ID,
		UserID:       member.ID,
	}).Error)

	rec := testkit.AuthedRequest(t, member, http.MethodPut, "/api/collections/"+collection.ID.String()+"/translations/order",
		map[string]any{"translation_ids": []string{tr.String()}})
	require.Equal(t, http.StatusForbidden, rec.Code)
}

// ===========================================================================
// POST /api/collections/:id/add-to-vocabulary (AddCollectionToVocabulary)
// ===========================================================================

func TestAddCollectionToVocabularyRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/collections/"+uuid.New().String()+"/add-to-vocabulary", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAddCollectionToVocabularyAllTranslations(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)
	tr1 := collectionSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	tr2 := collectionSeedTranslation(t, "cat", "Katze", enums.LanguageEn, enums.LanguageDe)
	collectionLink(t, collection.ID, tr1, 0)
	collectionLink(t, collection.ID, tr2, 1)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/add-to-vocabulary", map[string]any{})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		Added        int `json:"added"`
		Skipped      int `json:"skipped"`
		Total        int `json:"total"`
		UserAddCount int `json:"user_add_count"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, 2, body.Total)
	assert.Equal(t, 2, body.Added)
	assert.Equal(t, 0, body.Skipped)
	assert.Equal(t, 1, body.UserAddCount)

	// DB side effects: vocabulary rows + a user-add record.
	var vocabCount int64
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).
		Where("user_id = ? AND deleted_at IS NULL", user.ID).Count(&vocabCount).Error)
	assert.Equal(t, int64(2), vocabCount)

	var userAdds int64
	require.NoError(t, db.DB.Model(&models.CollectionUserAdd{}).
		Where("collection_id = ? AND user_id = ?", collection.ID, user.ID).Count(&userAdds).Error)
	assert.Equal(t, int64(1), userAdds)
}

func TestAddCollectionToVocabularySelectedAndSkips(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)
	tr1 := collectionSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	tr2 := collectionSeedTranslation(t, "cat", "Katze", enums.LanguageEn, enums.LanguageDe)
	collectionLink(t, collection.ID, tr1, 0)
	collectionLink(t, collection.ID, tr2, 1)

	// Pre-add tr1 to the user's vocabulary so it is skipped.
	first := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/add-to-vocabulary",
		map[string]any{"translation_ids": []string{tr1.String()}})
	require.Equal(t, http.StatusOK, first.Code)

	// Now add both: tr1 skipped, tr2 added.
	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/add-to-vocabulary",
		map[string]any{"translation_ids": []string{tr1.String(), tr2.String()}})
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Added   int `json:"added"`
		Skipped int `json:"skipped"`
		Total   int `json:"total"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, 2, body.Total)
	assert.Equal(t, 1, body.Added)
	assert.Equal(t, 1, body.Skipped)
}

func TestAddCollectionToVocabularyInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/not-a-uuid/add-to-vocabulary", map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid collection ID", body["error"])
}

func TestAddCollectionToVocabularyNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost,
		"/api/collections/"+uuid.New().String()+"/add-to-vocabulary", map[string]any{})
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAddCollectionToVocabularyAccessForbidden(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	stranger := testkit.CreateUser(t, testkit.WithName("Stranger"))
	// Private, non-published... actually published but owned by someone else and
	// not admin → stranger has no access → treated as not found (404).
	collection := collectionSeed(t, "Private", uintPtr(owner.ID), false, true)
	tr := collectionSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	collectionLink(t, collection.ID, tr, 0)

	rec := testkit.AuthedRequest(t, stranger, http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/add-to-vocabulary", map[string]any{})
	require.Equal(t, http.StatusNotFound, rec.Code, "stranger cannot access a private user collection")
}

// ===========================================================================
// POST /api/collections/:id/publish (PublishCollection)
// ===========================================================================

func TestPublishCollectionRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/collections/"+uuid.New().String()+"/publish",
		map[string]any{"is_published": true})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestPublishCollectionUnpublishThenPublish(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Animals", uintPtr(user.ID), false, true)

	// Unpublish.
	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/"+collection.ID.String()+"/publish",
		map[string]any{"is_published": false})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		IsPublished bool `json:"is_published"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.False(t, body.IsPublished)
	assert.False(t, collectionFindByID(t, collection.ID).IsPublished)

	// Re-publish.
	rec2 := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/"+collection.ID.String()+"/publish",
		map[string]any{"is_published": true})
	require.Equal(t, http.StatusOK, rec2.Code)
	assert.True(t, collectionFindByID(t, collection.ID).IsPublished)
}

func TestPublishCollectionInvalidID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/not-a-uuid/publish",
		map[string]any{"is_published": true})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid collection ID", body["error"])
}

func TestPublishCollectionNotFound(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collections/"+uuid.New().String()+"/publish",
		map[string]any{"is_published": true})
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPublishCollectionForbiddenForNonOwner(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	member := testkit.CreateUser(t, testkit.WithName("Member"))
	collection := collectionSeed(t, "Shared", uintPtr(owner.ID), false, true)
	require.NoError(t, db.DB.Create(&models.CollectionMember{
		CollectionID: collection.ID,
		UserID:       member.ID,
	}).Error)

	rec := testkit.AuthedRequest(t, member, http.MethodPost, "/api/collections/"+collection.ID.String()+"/publish",
		map[string]any{"is_published": false})
	require.Equal(t, http.StatusForbidden, rec.Code)
	assert.True(t, collectionFindByID(t, collection.ID).IsPublished, "state unchanged")
}

func TestPublishCollectionAdminCanEditGlobal(t *testing.T) {
	testkit.Truncate(t)

	admin := testkit.CreateUser(t, testkit.WithAdmin())
	// Global admin collection with no specific owner.
	collection := collectionSeed(t, "Global", nil, true, false)

	rec := testkit.AuthedRequest(t, admin, http.MethodPost, "/api/collections/"+collection.ID.String()+"/publish",
		map[string]any{"is_published": true})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.True(t, collectionFindByID(t, collection.ID).IsPublished)
}

// ===========================================================================
// POST /api/collection-generate (GenerateCollection)
// ===========================================================================

func TestGenerateCollectionRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/collection-generate", map[string]any{"prompt": "animals"})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGenerateCollectionHappyPath(t *testing.T) {
	testkit.Truncate(t)

	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
		GenerateFunc: func(prompt string, allowedLanguages []string) (*openrouter.GeneratedCollection, error) {
			assert.Equal(t, "animals in german", prompt)
			return &openrouter.GeneratedCollection{
				Title: "Animals",
				Translations: []openrouter.GeneratedTranslation{
					{Original: "dog", OriginalLanguage: "en", Translation: "der Hund", TranslationLanguage: "de"},
					{Original: "cat", OriginalLanguage: "en", Translation: "die Katze", TranslationLanguage: "de"},
				},
			}, nil
		},
	})

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collection-generate",
		map[string]any{"prompt": "animals in german"})
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		ID               uuid.UUID `json:"id"`
		Title            string    `json:"title"`
		IsAdmin          bool      `json:"is_admin"`
		IsPublished      bool      `json:"is_published"`
		TranslationCount int       `json:"translation_count"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "Animals", body.Title)
	assert.False(t, body.IsAdmin)
	assert.True(t, body.IsPublished, "non-admin generated collection is published")
	assert.Equal(t, 2, body.TranslationCount)

	// Persisted with the generated translations.
	assert.Equal(t, int64(2), collectionTranslationCount(t, body.ID))
	stored := collectionFindByID(t, body.ID)
	require.NotNil(t, stored.OwnerID)
	assert.Equal(t, user.ID, *stored.OwnerID)
}

func TestGenerateCollectionAdminProducesUnpublishedGlobal(t *testing.T) {
	testkit.Truncate(t)

	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
		GenerateFunc: func(prompt string, allowedLanguages []string) (*openrouter.GeneratedCollection, error) {
			return &openrouter.GeneratedCollection{
				Title: "Global Animals",
				Translations: []openrouter.GeneratedTranslation{
					{Original: "dog", OriginalLanguage: "en", Translation: "der Hund", TranslationLanguage: "de"},
				},
			}, nil
		},
	})

	admin := testkit.CreateUser(t, testkit.WithAdmin())

	rec := testkit.AuthedRequest(t, admin, http.MethodPost, "/api/collection-generate",
		map[string]any{"prompt": "animals"})
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		ID          uuid.UUID `json:"id"`
		IsAdmin     bool      `json:"is_admin"`
		IsPublished bool      `json:"is_published"`
	}
	testkit.DecodeJSON(t, rec, &body)
	// For admins the generated collection is admin-owned and starts unpublished.
	assert.True(t, body.IsAdmin)
	assert.False(t, body.IsPublished)

	stored := collectionFindByID(t, body.ID)
	assert.True(t, stored.IsAdmin)
	assert.False(t, stored.IsPublished)
}

func TestGenerateCollectionMissingPrompt(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collection-generate", map[string]any{})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body struct {
		Errors map[string]string `json:"errors"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "required", body.Errors["Prompt"])
}

func TestGenerateCollectionBlankPrompt(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	// Whitespace-only prompt passes binding:required but is rejected by the service.
	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collection-generate", map[string]any{"prompt": "   "})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "prompt can't be empty", body["error"])
}

func TestGenerateCollectionOpenRouterFailure(t *testing.T) {
	testkit.Truncate(t)

	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
		GenerateFunc: func(prompt string, allowedLanguages []string) (*openrouter.GeneratedCollection, error) {
			return nil, assert.AnError
		},
	})

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collection-generate",
		map[string]any{"prompt": "animals"})
	// AIGenerationFailed → ServerError → 500.
	require.Equal(t, http.StatusInternalServerError, rec.Code, "body=%s", rec.Body.String())
}

func TestGenerateCollectionNotConfigured(t *testing.T) {
	testkit.Truncate(t)

	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
		GenerateFunc: func(prompt string, allowedLanguages []string) (*openrouter.GeneratedCollection, error) {
			return nil, openrouter.ErrNotConfigured
		},
	})

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collection-generate",
		map[string]any{"prompt": "animals"})
	// ErrNotConfigured → AIGenerationUnavailable → 503.
	require.Equal(t, http.StatusServiceUnavailable, rec.Code, "body=%s", rec.Body.String())

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "ai generation is not configured", body["error"])
}

func TestGenerateCollectionNoUsablePairs(t *testing.T) {
	testkit.Truncate(t)

	// The mock returns translations that all get filtered out (same language),
	// leaving zero usable pairs → AIGenerationFailed → 500.
	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
		GenerateFunc: func(prompt string, allowedLanguages []string) (*openrouter.GeneratedCollection, error) {
			return &openrouter.GeneratedCollection{
				Title: "Bad",
				Translations: []openrouter.GeneratedTranslation{
					{Original: "dog", OriginalLanguage: "en", Translation: "dog", TranslationLanguage: "en"},
				},
			}, nil
		},
	})

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collection-generate",
		map[string]any{"prompt": "animals"})
	require.Equal(t, http.StatusInternalServerError, rec.Code, "body=%s", rec.Body.String())
}

// ===========================================================================
// POST /api/collection-invites/:token (JoinCollection)
// ===========================================================================

func TestJoinCollectionRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/collection-invites/sometoken", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJoinCollectionHappyPath(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	joiner := testkit.CreateUser(t, testkit.WithName("Joiner"))
	collection := collectionSeed(t, "Shared", uintPtr(owner.ID), false, true)

	rec := testkit.AuthedRequest(t, joiner, http.MethodPost, "/api/collection-invites/"+collection.InviteToken, nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var body struct {
		ID      uuid.UUID `json:"id"`
		IsOwner bool      `json:"is_owner"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, collection.ID, body.ID)
	assert.False(t, body.IsOwner)

	// Membership row created.
	assert.Equal(t, int64(1), collectionMemberCount(t, collection.ID))

	// Joiner now sees the collection in their list.
	listRec := testkit.AuthedRequest(t, joiner, http.MethodGet, "/api/collections", nil)
	require.Equal(t, http.StatusOK, listRec.Code)
	var list struct {
		Data []struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	testkit.DecodeJSON(t, listRec, &list)
	found := false
	for _, d := range list.Data {
		if d.ID == collection.ID {
			found = true
		}
	}
	assert.True(t, found, "joiner should see the joined collection")
}

func TestJoinCollectionIdempotent(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	joiner := testkit.CreateUser(t, testkit.WithName("Joiner"))
	collection := collectionSeed(t, "Shared", uintPtr(owner.ID), false, true)

	first := testkit.AuthedRequest(t, joiner, http.MethodPost, "/api/collection-invites/"+collection.InviteToken, nil)
	require.Equal(t, http.StatusOK, first.Code)
	second := testkit.AuthedRequest(t, joiner, http.MethodPost, "/api/collection-invites/"+collection.InviteToken, nil)
	require.Equal(t, http.StatusOK, second.Code)

	// Still only one membership row (OnConflict DoNothing).
	assert.Equal(t, int64(1), collectionMemberCount(t, collection.ID))
}

func TestJoinCollectionOwnerDoesNotCreateMembership(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	collection := collectionSeed(t, "Shared", uintPtr(owner.ID), false, true)

	rec := testkit.AuthedRequest(t, owner, http.MethodPost, "/api/collection-invites/"+collection.InviteToken, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		IsOwner bool `json:"is_owner"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.True(t, body.IsOwner)
	assert.Equal(t, int64(0), collectionMemberCount(t, collection.ID), "owner is not added as a member")
}

func TestJoinCollectionInvalidToken(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collection-invites/does-not-exist", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)

	var body map[string]any
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid invite link", body["error"])
}

func TestJoinCollectionDeletedCollectionToken(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Gone", uintPtr(owner.ID), false, true)

	now := time.Now().UTC()
	require.NoError(t, db.DB.Model(&models.Collection{}).
		Where("id = ?", collection.ID).Update("deleted_at", now).Error)

	rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/collection-invites/"+collection.InviteToken, nil)
	require.Equal(t, http.StatusNotFound, rec.Code, "token of a deleted collection is invalid")
}
