package tests

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"termorize/src/data/db"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"
	"testing"
	"time"
)

func TestShareVocabularyPublicPreviewAndImport(t *testing.T) {
	testkit.Truncate(t)
	owner := testkit.CreateUser(t)
	friend := testkit.CreateUser(t)
	first := seedVocabularyForWordDeletion(t, owner.ID, "forest", "Wald")
	second := seedVocabularyForWordDeletion(t, owner.ID, "river", "Fluss")
	rec := testkit.AuthedRequest(t, owner, http.MethodPost, "/api/vocabulary/share", map[string]any{
		"title": " Walk outside ", "vocabulary_ids": []uuid.UUID{second.ID, first.ID, second.ID},
	})
	testkit.RequireStatus(t, rec, http.StatusCreated)
	var collection services.CollectionDetail
	testkit.DecodeJSON(t, rec, &collection)
	require.Equal(t, "Walk outside", collection.Title)
	require.True(t, collection.IsOwner)
	require.False(t, collection.IsAdmin)
	require.True(t, collection.IsPublished)
	require.Len(t, collection.Translations, 2)
	require.Equal(t, second.TranslationID, collection.Translations[0].ID)
	require.Equal(t, first.TranslationID, collection.Translations[1].ID)
	public := testkit.Request(t, http.MethodGet, "/api/public/collections/"+collection.ID.String(), nil)
	testkit.RequireStatus(t, public, http.StatusOK)
	require.NotContains(t, public.Body.String(), "progress")
	imported := testkit.AuthedRequest(t, friend, http.MethodPost, "/api/collections/"+collection.ID.String()+"/add-to-vocabulary", nil)
	testkit.RequireStatus(t, imported, http.StatusOK)
	var result services.AddCollectionToVocabularyResult
	testkit.DecodeJSON(t, imported, &result)
	require.Equal(t, 2, result.Added)
	// Removing a word from the owner's vocabulary does not alter the shared snapshot.
	require.NoError(t, db.DB.Model(&first).Update("deleted_at", time.Now()).Error)
	public = testkit.Request(t, http.MethodGet, "/api/public/collections/"+collection.ID.String(), nil)
	testkit.RequireStatus(t, public, http.StatusOK)
	var detail services.PublicCollectionDetail
	testkit.DecodeJSON(t, public, &detail)
	require.Len(t, detail.Translations, 2)
	_, err := services.SetCollectionIsPublished(owner.ID, collection.ID, false)
	require.NoError(t, err)
	testkit.RequireStatus(t, testkit.Request(t, http.MethodGet, "/api/public/collections/"+collection.ID.String(), nil), http.StatusNotFound)
}

func TestShareVocabularyRejectsInvalidSelectionsAtomically(t *testing.T) {
	testkit.Truncate(t)
	owner := testkit.CreateUser(t)
	other := testkit.CreateUser(t)
	own := seedVocabularyForWordDeletion(t, owner.ID, "tree", "Baum")
	foreign := seedVocabularyForWordDeletion(t, other.ID, "lake", "See")
	deleted := seedVocabularyForWordDeletion(t, owner.ID, "bird", "Vogel")
	require.NoError(t, db.DB.Model(&deleted).Update("deleted_at", time.Now()).Error)
	for _, ids := range [][]uuid.UUID{nil, {}, {own.ID, foreign.ID}, {own.ID, deleted.ID}, {own.ID, uuid.New()}} {
		rec := testkit.AuthedRequest(t, owner, http.MethodPost, "/api/vocabulary/share", map[string]any{"title": "Share", "vocabulary_ids": ids})
		testkit.RequireStatus(t, rec, http.StatusBadRequest)
	}
	for _, title := range []string{"", "   "} {
		rec := testkit.AuthedRequest(t, owner, http.MethodPost, "/api/vocabulary/share", map[string]any{"title": title, "vocabulary_ids": []uuid.UUID{own.ID}})
		testkit.RequireStatus(t, rec, http.StatusBadRequest)
	}
	var count int64
	require.NoError(t, db.DB.Model(&models.Collection{}).Count(&count).Error)
	require.Zero(t, count)
	testkit.RequireStatus(t, testkit.Request(t, http.MethodPost, "/api/vocabulary/share", map[string]any{"title": "Share", "vocabulary_ids": []uuid.UUID{own.ID}}), http.StatusUnauthorized)
}
