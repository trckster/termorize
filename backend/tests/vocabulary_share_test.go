package tests

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"termorize/src/data/db"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"
	"testing"
)

func TestShareVocabularyPublicPreviewAndImport(t *testing.T) {
	testkit.Truncate(t)
	owner := testkit.CreateUser(t)
	friend := testkit.CreateUser(t)
	item := seedVocabularyForWordDeletion(t, owner.ID, "forest", "Wald")
	path := "/api/vocabulary/" + item.ID.String() + "/share"
	rec := testkit.AuthedRequest(t, owner, http.MethodPost, path, nil)
	testkit.RequireStatus(t, rec, http.StatusOK)
	var share services.VocabularyShare
	testkit.DecodeJSON(t, rec, &share)
	require.Equal(t, item.TranslationID, share.Translation.ID)
	require.Equal(t, "forest", share.Translation.Original.Word)
	require.True(t, strings.HasPrefix(share.SharePath, "/words/"))
	again := testkit.AuthedRequest(t, owner, http.MethodPost, path, nil)
	testkit.RequireStatus(t, again, http.StatusOK)
	require.JSONEq(t, rec.Body.String(), again.Body.String())
	token := strings.TrimPrefix(share.SharePath, "/words/")
	publicPath := "/api/public/vocabulary/" + token
	public := testkit.Request(t, http.MethodGet, publicPath, nil)
	testkit.RequireStatus(t, public, http.StatusOK)
	require.Equal(t, "no-store", public.Header().Get("Cache-Control"))
	for _, field := range []string{"progress", "mastered_at", "user_id", "share_token", "username"} {
		require.NotContains(t, public.Body.String(), field)
	}
	var collections int64
	require.NoError(t, db.DB.Model(&models.Collection{}).Count(&collections).Error)
	require.Zero(t, collections)
	imported := testkit.AuthedRequest(t, friend, http.MethodPost, "/api/vocabulary/translation", map[string]any{"translation_id": share.Translation.ID})
	testkit.RequireStatus(t, imported, http.StatusCreated)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, friend, http.MethodPost, "/api/vocabulary/translation", map[string]any{"translation_id": share.Translation.ID}), http.StatusConflict)
	meta := testkit.Request(t, http.MethodGet, "/api/public/collection-share-metadata?path="+share.SharePath, nil)
	testkit.RequireStatus(t, meta, http.StatusOK)
	require.Contains(t, meta.Body.String(), "forest — Wald")
	require.Equal(t, "no-store", meta.Header().Get("Cache-Control"))
	testkit.RequireStatus(t, testkit.AuthedRequest(t, owner, http.MethodDelete, path, nil), http.StatusNoContent)
	testkit.RequireStatus(t, testkit.Request(t, http.MethodGet, publicPath, nil), http.StatusNotFound)
	next := testkit.AuthedRequest(t, owner, http.MethodPost, path, nil)
	testkit.RequireStatus(t, next, http.StatusOK)
	var nextShare services.VocabularyShare
	testkit.DecodeJSON(t, next, &nextShare)
	require.NotEqual(t, share.SharePath, nextShare.SharePath)
	require.NoError(t, services.DeleteVocabulary(owner.ID, item.ID))
	testkit.RequireStatus(t, testkit.Request(t, http.MethodGet, "/api/public/vocabulary/"+strings.TrimPrefix(nextShare.SharePath, "/words/"), nil), http.StatusNotFound)
	// Restoring a deleted word must not revive its old public link.
	_, err := services.CreateVocabularyByTranslation(owner.ID, item.TranslationID)
	require.NoError(t, err)
	testkit.RequireStatus(t, testkit.Request(t, http.MethodGet, "/api/public/vocabulary/"+strings.TrimPrefix(nextShare.SharePath, "/words/"), nil), http.StatusNotFound)
}

func TestShareVocabularyRequiresOwnership(t *testing.T) {
	testkit.Truncate(t)
	owner := testkit.CreateUser(t)
	other := testkit.CreateUser(t)
	item := seedVocabularyForWordDeletion(t, owner.ID, "tree", "Baum")
	for _, id := range []uuid.UUID{item.ID, uuid.New()} {
		path := "/api/vocabulary/" + id.String() + "/share"
		for _, method := range []string{http.MethodPost, http.MethodDelete} {
			testkit.RequireStatus(t, testkit.Request(t, method, path, nil), http.StatusUnauthorized)
			testkit.RequireStatus(t, testkit.AuthedRequest(t, other, method, path, nil), http.StatusNotFound)
		}
	}
	require.NoError(t, services.DeleteVocabulary(owner.ID, item.ID))
	testkit.RequireStatus(t, testkit.AuthedRequest(t, owner, http.MethodPost, "/api/vocabulary/"+item.ID.String()+"/share", nil), http.StatusNotFound)
	for _, token := range []string{"invalid", strings.Repeat("a", 22)} {
		testkit.RequireStatus(t, testkit.Request(t, http.MethodGet, "/api/public/vocabulary/"+token, nil), http.StatusNotFound)
	}
}
