package tests

import (
	"net/http"
	"net/url"
	"testing"

	"termorize/src/enums"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicCollectionByIDShowsPublishedPersonalCollection(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	collection := collectionSeed(t, "Forest best words!!!", uintPtr(owner.ID), false, true)
	translationID := collectionSeedTranslation(t, "forest", "Wald", enums.LanguageEn, enums.LanguageDe)
	collectionLink(t, collection.ID, translationID, 0)

	rec := testkit.Request(t, http.MethodGet, "/api/public/collections/"+collection.ID.String(), nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body struct {
		ID               uuid.UUID `json:"id"`
		Title            string    `json:"title"`
		TranslationCount int       `json:"translation_count"`
		SharePath        string    `json:"share_path"`
		Translations     []struct {
			Original struct {
				Word string `json:"word"`
			} `json:"original"`
		} `json:"translations"`
	}
	testkit.DecodeJSON(t, rec, &body)

	assert.Equal(t, collection.ID, body.ID)
	assert.Equal(t, "Forest best words!!!", body.Title)
	assert.Equal(t, 1, body.TranslationCount)
	assert.Equal(t, "/collections/join/"+services.CollectionShareIdentifier(&collection), body.SharePath)
	require.Len(t, body.Translations, 1)
	assert.Equal(t, "forest", body.Translations[0].Original.Word)
}

func TestPublicCollectionByShareIdentifierRequiresCanonicalNewFormat(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t)
	collection := collectionSeed(t, "Forest best words!!!", uintPtr(owner.ID), false, true)
	identifier := services.CollectionShareIdentifier(&collection)

	rec := testkit.Request(t, http.MethodGet, "/api/public/collection-invites/"+identifier, nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	legacy := testkit.Request(t, http.MethodGet, "/api/public/collection-invites/"+collection.InviteToken, nil)
	testkit.RequireStatus(t, legacy, http.StatusNotFound)

	staleSlug := testkit.Request(t, http.MethodGet, "/api/public/collection-invites/old-title-"+collection.InviteToken, nil)
	testkit.RequireStatus(t, staleSlug, http.StatusNotFound)
}

func TestPublicCollectionByShareIdentifierSupportsUnicodeSlug(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t)
	collection := collectionSeed(t, "Лучшие слова!", uintPtr(owner.ID), false, true)
	identifier := services.CollectionShareIdentifier(&collection)

	rec := testkit.Request(t, http.MethodGet,
		"/api/public/collection-invites/"+url.PathEscape(identifier), nil)
	testkit.RequireStatus(t, rec, http.StatusOK)
}

func TestPublicCollectionEndpointsHideUnpublishedCollections(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t)
	collection := collectionSeed(t, "Draft words", uintPtr(owner.ID), false, false)

	direct := testkit.Request(t, http.MethodGet, "/api/public/collections/"+collection.ID.String(), nil)
	testkit.RequireStatus(t, direct, http.StatusNotFound)

	shared := testkit.Request(t, http.MethodGet,
		"/api/public/collection-invites/"+services.CollectionShareIdentifier(&collection), nil)
	testkit.RequireStatus(t, shared, http.StatusNotFound)
}

func TestAuthenticatedUserCanOpenPublishedPersonalCollectionDirectly(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	viewer := testkit.CreateUser(t, testkit.WithName("Viewer"))
	collection := collectionSeed(t, "Shared reading", uintPtr(owner.ID), false, true)

	rec := testkit.AuthedRequest(t, viewer, http.MethodGet, "/api/collections/"+collection.ID.String(), nil)
	testkit.RequireStatus(t, rec, http.StatusOK)
}

func TestPublicCollectionByIDShowsPublishedGlobalCollection(t *testing.T) {
	testkit.Truncate(t)

	admin := testkit.CreateUser(t, testkit.WithAdmin())
	collection := collectionSeed(t, "Global words", uintPtr(admin.ID), true, true)

	rec := testkit.Request(t, http.MethodGet, "/api/public/collections/"+collection.ID.String(), nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	var body struct {
		IsAdmin   bool   `json:"is_admin"`
		SharePath string `json:"share_path"`
	}
	testkit.DecodeJSON(t, rec, &body)
	assert.True(t, body.IsAdmin)
	assert.Equal(t, "/collections/join/"+services.CollectionShareIdentifier(&collection), body.SharePath)
}

func TestCollectionShareMetadataContainsEscapedDynamicOpenGraphTags(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t)
	collection := collectionSeed(t, "<Forest & Trees>", uintPtr(owner.ID), false, true)
	requestPath := "/collections/join/" + services.CollectionShareIdentifier(&collection)

	rec := testkit.Request(t, http.MethodGet,
		"/api/public/collection-share-metadata?path="+url.QueryEscape(requestPath), nil)
	testkit.RequireStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	assert.Contains(t, body, `property="og:title" content="&lt;Forest &amp; Trees&gt; · Termorize"`)
	assert.Contains(t, body, `property="og:type" content="website"`)
	assert.Contains(t, body, `name="twitter:card" content="summary_large_image"`)
	assert.Contains(t, body, "/collection-share.png")
	assert.NotContains(t, body, "<Forest & Trees>")
}

func TestCollectionShareMetadataSupportsDirectCollectionURL(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t)
	collection := collectionSeed(t, "Direct preview", uintPtr(owner.ID), false, true)
	requestPath := "/collections/" + collection.ID.String()

	rec := testkit.Request(t, http.MethodGet,
		"/api/public/collection-share-metadata?path="+url.QueryEscape(requestPath), nil)
	testkit.RequireStatus(t, rec, http.StatusOK)
	assert.Contains(t, rec.Body.String(), `property="og:title" content="Direct preview · Termorize"`)
}
