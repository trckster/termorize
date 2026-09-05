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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedAdminDescription(t *testing.T, word, description string) models.WordDescription {
	t.Helper()
	w := models.Word{Word: word, Language: enums.LanguageEn}
	require.NoError(t, db.DB.Create(&w).Error)
	d := models.WordDescription{WordID: w.ID, Model: config.GetOpenRouterModel(), Description: description}
	require.NoError(t, db.DB.Create(&d).Error)
	return d
}

func TestAdminDescriptionsAuthorization(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	for _, endpoint := range []struct{ method, path string }{
		{http.MethodGet, "/api/admin/word-descriptions"},
		{http.MethodGet, "/api/admin/description-models"},
		{http.MethodPost, "/api/admin/word-descriptions/" + uuid.NewString() + "/preview"},
		{http.MethodPost, "/api/admin/word-descriptions/" + uuid.NewString() + "/approve"},
	} {
		testkit.RequireStatus(t, testkit.Request(t, endpoint.method, endpoint.path, nil), http.StatusUnauthorized)
		testkit.RequireStatus(t, testkit.AuthedRequest(t, user, endpoint.method, endpoint.path, nil), http.StatusForbidden)
	}
}

func TestAdminDescriptionsSearchPaginationAndModels(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	seedAdminDescription(t, "cat", "A small pet that purrs.")
	seedAdminDescription(t, "dog", "A loyal pet that barks.")
	seedAdminDescription(t, "car", "A road vehicle.")
	for _, search := range []string{"PET", "cat"} {
		rec := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/word-descriptions?page=1&page_size=1&search="+search, nil)
		testkit.RequireStatus(t, rec, http.StatusOK)
		var response services.AdminWordDescriptionsResponse
		testkit.DecodeJSON(t, rec, &response)
		require.Len(t, response.Data, 1)
		if search == "PET" {
			assert.EqualValues(t, 2, response.Pagination.Total)
			assert.Equal(t, 2, response.Pagination.TotalPages)
		} else {
			assert.EqualValues(t, 1, response.Pagination.Total)
		}
	}
	for _, query := range []string{"page=0", "page_size=101", "page=abc", "page_size=-1"} {
		testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/word-descriptions?"+query, nil), http.StatusBadRequest)
	}
	rec := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/description-models", nil)
	testkit.RequireStatus(t, rec, http.StatusOK)
	var options []struct{ ID, Tier string }
	testkit.DecodeJSON(t, rec, &options)
	require.Len(t, options, 3)
	assert.Equal(t, config.GetOpenRouterModel(), options[0].ID)
	assert.Equal(t, "moonshotai/kimi-k2.6", options[1].ID)
	assert.Equal(t, "openai/gpt-5.6-sol", options[2].ID)
}

func mockAdminDescription(t *testing.T, fake *testkit.FakeOpenRouter, expectedModel string) {
	t.Helper()
	original := openrouter.NewClientWithModel
	openrouter.NewClientWithModel = func(model string) openrouter.Client { assert.Equal(t, expectedModel, model); return fake }
	t.Cleanup(func() { openrouter.NewClientWithModel = original })
}

func previewAdminDescription(t *testing.T, admin models.User, d models.WordDescription, model string) services.WordDescriptionPreview {
	t.Helper()
	rec := testkit.AuthedRequest(t, admin, http.MethodPost, "/api/admin/word-descriptions/"+d.ID.String()+"/preview", map[string]any{"model": model})
	testkit.RequireStatus(t, rec, http.StatusOK)
	var preview services.WordDescriptionPreview
	testkit.DecodeJSON(t, rec, &preview)
	return preview
}

func TestAdminDescriptionPreviewRequiresApprovalAndUsesSelectedModel(t *testing.T) {
	for _, model := range []string{config.GetOpenRouterModel(), "moonshotai/kimi-k2.6", "openai/gpt-5.6-sol"} {
		t.Run(model, func(t *testing.T) {
			testkit.Truncate(t)
			admin := testkit.CreateUser(t, testkit.WithAdmin())
			existing := seedAdminDescription(t, "cat", "Old clue.")
			calls := 0
			mockAdminDescription(t, &testkit.FakeOpenRouter{GenerateDescriptionFunc: func(word, wordLanguage, descriptionLanguage string) (*openrouter.GeneratedDescription, error) {
				calls++
				assert.Equal(t, "cat", word)
				assert.Equal(t, "English", descriptionLanguage)
				return &openrouter.GeneratedDescription{Description: "A small pet that purrs."}, nil
			}}, model)
			preview := previewAdminDescription(t, admin, existing, model)
			require.Equal(t, model, preview.Model)
			cached, err := services.GetOrCreateWordDescription(existing.WordID)
			require.NoError(t, err)
			assert.Equal(t, "Old clue.", cached.Description)
			testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, "/api/admin/word-descriptions/"+existing.ID.String()+"/approve", map[string]any{"preview_id": preview.ID}), http.StatusOK)
			cached, err = services.GetOrCreateWordDescription(existing.WordID)
			require.NoError(t, err)
			assert.Equal(t, preview.Description, cached.Description)
			assert.Equal(t, model, cached.Model)
			assert.NotNil(t, cached.ApprovedAt)
			assert.Equal(t, 1, calls, "approval must save the preview without generating another clue")
			testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, "/api/admin/word-descriptions/"+existing.ID.String()+"/approve", map[string]any{"preview_id": preview.ID}), http.StatusNotFound)
		})
	}
}

func TestAdminDescriptionRejectsInvalidPreviewAndPreservesCache(t *testing.T) {
	for _, scenario := range []string{"provider", "answer", "morphology", "language", "empty", "long"} {
		t.Run(scenario, func(t *testing.T) {
			testkit.Truncate(t)
			admin := testkit.CreateUser(t, testkit.WithAdmin())
			existing := seedAdminDescription(t, "cat", "Old clue.")
			fake := &testkit.FakeOpenRouter{GenerateDescriptionFunc: func(string, string, string) (*openrouter.GeneratedDescription, error) {
				if scenario == "provider" {
					return nil, errors.New("unavailable")
				}
				description := "A small pet that purrs."
				switch scenario {
				case "answer":
					description = "A cat."
				case "empty":
					description = " "
				case "long":
					description = strings.Repeat("x", 301)
				}
				return &openrouter.GeneratedDescription{Description: description}, nil
			}, DescriptionContainsAnswerFormFunc: func(string, string, string) (bool, error) { return scenario == "morphology", nil }}
			if scenario == "language" {
				testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{DetectFunc: func(string) (string, error) { return "it", nil }})
			}
			mockAdminDescription(t, fake, config.GetOpenRouterModel())
			testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, "/api/admin/word-descriptions/"+existing.ID.String()+"/preview", map[string]any{"model": config.GetOpenRouterModel()}), http.StatusInternalServerError)
			cached, err := services.GetOrCreateWordDescription(existing.WordID)
			require.NoError(t, err)
			assert.Equal(t, "Old clue.", cached.Description)
			var count int64
			require.NoError(t, db.DB.Model(&services.WordDescriptionPreview{}).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}

func TestAdminDescriptionPreviewOwnershipExpiryAndConflicts(t *testing.T) {
	for _, scenario := range []string{"owner", "expired", "changed", "wrong-description", "unsupported"} {
		t.Run(scenario, func(t *testing.T) {
			testkit.Truncate(t)
			admin := testkit.CreateUser(t, testkit.WithAdmin())
			existing := seedAdminDescription(t, "cat", "Old clue.")
			path := "/api/admin/word-descriptions/" + existing.ID.String()
			if scenario == "unsupported" {
				testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, path+"/preview", map[string]any{"model": "arbitrary/model"}), http.StatusBadRequest)
				return
			}
			preview := previewAdminDescription(t, admin, existing, config.GetOpenRouterModel())
			expected := http.StatusConflict
			switch scenario {
			case "owner":
				admin = testkit.CreateUser(t, testkit.WithAdmin())
				expected = http.StatusNotFound
			case "expired":
				require.NoError(t, db.DB.Model(&services.WordDescriptionPreview{}).Where("id = ?", preview.ID).Update("created_at", time.Now().Add(-25*time.Hour)).Error)
			case "changed":
				require.NoError(t, db.DB.Model(&existing).Update("description", "Newer clue.").Error)
			case "wrong-description":
				other := seedAdminDescription(t, "dog", "Another clue.")
				path = "/api/admin/word-descriptions/" + other.ID.String()
				expected = http.StatusNotFound
			}
			testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, path+"/approve", map[string]any{"preview_id": preview.ID, "description": "Tampered clue"}), expected)
			cached, err := services.GetOrCreateWordDescription(existing.WordID)
			require.NoError(t, err)
			if scenario == "changed" {
				assert.Equal(t, "Newer clue.", cached.Description)
			} else {
				assert.Equal(t, "Old clue.", cached.Description)
			}
		})
	}
}

func TestAdminDescriptionApprovalReplacesExistingModelCache(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	existing := seedAdminDescription(t, "cat", "Old clue.")
	other := models.WordDescription{WordID: existing.WordID, Model: "moonshotai/kimi-k2.6", Description: "Previous Kimi clue."}
	require.NoError(t, db.DB.Create(&other).Error)
	preview := previewAdminDescription(t, admin, existing, other.Model)
	require.NoError(t, services.ApproveWordDescriptionForAdmin(existing.ID, preview.ID, admin.ID))
	cached, err := services.GetOrCreateWordDescription(existing.WordID)
	require.NoError(t, err)
	assert.Equal(t, preview.Description, cached.Description)
	assert.Equal(t, other.Model, cached.Model)
	var count int64
	require.NoError(t, db.DB.Model(&models.WordDescription{}).Where("word_id = ?", existing.WordID).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}
