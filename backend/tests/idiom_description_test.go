package tests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
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

func seedIdiomSelection(t *testing.T, word models.Word, date time.Time) models.DailyIdiom {
	t.Helper()
	selected := models.DailyIdiom{WordID: word.ID, Language: word.Language, Date: date}
	require.NoError(t, db.DB.Create(&selected).Error)
	return selected
}

func idiomDescriptionPath(id uuid.UUID) string {
	return "/api/daily-idiom/" + id.String() + "/description"
}

func readIdiomDescription(t *testing.T, user models.User, id uuid.UUID) string {
	t.Helper()
	rec := testkit.AuthedRequest(t, user, http.MethodGet, idiomDescriptionPath(id), nil)
	testkit.RequireStatus(t, rec, http.StatusOK)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	var result struct {
		Description string `json:"description"`
	}
	testkit.DecodeJSON(t, rec, &result)
	return result.Description
}

// Contract: authenticated selection IDs; shared descriptions, deterministic reuse,
// validated generation, retry without reselection, and independent selection locks.
func TestDailyIdiomDescriptionEndpoint(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	path := idiomDescriptionPath(uuid.New())
	testkit.RequireStatus(t, testkit.Request(t, http.MethodGet, path, nil), http.StatusUnauthorized)
	stale := testkit.CreateUser(t)
	require.NoError(t, db.DB.Delete(&stale).Error)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, stale, http.MethodGet, path, nil), http.StatusUnauthorized)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, user, http.MethodGet, path, nil), http.StatusNotFound)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, user, http.MethodGet, "/api/daily-idiom/invalid/description", nil), http.StatusBadRequest)
	ordinary := models.Word{Word: "ordinary", Language: enums.LanguageEn}
	require.NoError(t, db.DB.Create(&ordinary).Error)
	selection := seedIdiomSelection(t, ordinary, time.Now())
	testkit.RequireStatus(t, testkit.AuthedRequest(t, user, http.MethodGet, idiomDescriptionPath(selection.ID), nil), http.StatusNotFound)
	var count int64
	require.NoError(t, db.DB.Model(&models.WordDescription{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestDailyIdiomDescriptionReusesNewestAndPrefersNullContext(t *testing.T) {
	testkit.Truncate(t)
	user, other := testkit.CreateUser(t), testkit.CreateUser(t)
	word := seedDailyIdiomWord(t, "under the weather", enums.LanguageEn)
	translated := models.Word{Word: "нездоровый", Language: enums.LanguageRu}
	require.NoError(t, db.DB.Create(&translated).Error)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	first := seedIdiomSelection(t, word, today)
	later := seedIdiomSelection(t, word, today.AddDate(0, 0, 2))
	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{GenerateIdiomDescriptionFunc: func(context.Context, string, string) (*openrouter.GeneratedDescription, error) {
		t.Error("existing descriptions must not trigger generation")
		return nil, errors.New("unexpected generation")
	}})
	rows := []models.WordDescription{
		{ID: uuid.UUID{15: 3}, WordID: word.ID, TranslationWordID: &translated.ID, Model: "older", Description: "An older exercise clue.", CreatedAt: today.Add(-time.Hour)},
		{ID: uuid.UUID{15: 1}, WordID: word.ID, TranslationWordID: &translated.ID, Model: "one", Description: "Feeling sick.", CreatedAt: today},
		{ID: uuid.UUID{15: 2}, WordID: word.ID, TranslationWordID: &translated.ID, Model: "two", Description: "Feeling sick or unwell.", CreatedAt: today},
	}
	require.NoError(t, db.DB.Create(&rows).Error)
	assert.Equal(t, rows[2].Description, readIdiomDescription(t, user, first.ID))
	assert.Equal(t, rows[2].Description, readIdiomDescription(t, other, later.ID))
	nullRows := []models.WordDescription{
		{ID: uuid.UUID{15: 4}, WordID: word.ID, Model: "idiom", Description: "sick, unwell", CreatedAt: today.Add(-2 * time.Hour)},
		{ID: uuid.UUID{15: 5}, WordID: word.ID, Model: "idiom", Description: "sick, unwell, indisposed", CreatedAt: today.Add(-2 * time.Hour)},
	}
	require.NoError(t, db.DB.Create(&nullRows).Error)
	assert.Equal(t, nullRows[1].Description, readIdiomDescription(t, user, first.ID))
	exercise, err := services.GetOrCreateWordDescription(word.ID, translated.ID)
	require.NoError(t, err)
	assert.Equal(t, rows[2].ID, exercise.ID)
	var count int64
	require.NoError(t, db.DB.Model(&models.WordDescription{}).Count(&count).Error)
	assert.EqualValues(t, 5, count)
}

func TestDailyIdiomDescriptionGeneratesAndCachesInItsOwnLanguage(t *testing.T) {
	for _, tc := range []struct {
		language          enums.Language
		word, description string
	}{
		{enums.LanguageEn, "break the ice", "start a conversation; make people feel more comfortable"},
		{enums.LanguageRu, "бить баклуши", "бездельничать, лениться"},
		{enums.LanguageIt, "essere al settimo cielo", "felice, entusiasta, euforico"},
	} {
		t.Run(string(tc.language), func(t *testing.T) {
			testkit.Truncate(t)
			user := testkit.CreateUser(t)
			word := seedDailyIdiomWord(t, tc.word, tc.language)
			selection := seedIdiomSelection(t, word, time.Now())
			calls := 0
			testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{GenerateIdiomDescriptionFunc: func(ctx context.Context, idiom, language string) (*openrouter.GeneratedDescription, error) {
				calls++
				assert.Equal(t, tc.word, idiom)
				assert.Equal(t, tc.language.DisplayName(), language)
				return &openrouter.GeneratedDescription{Description: tc.description}, nil
			}})
			testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{DetectFunc: func(text string) (string, error) {
				assert.Equal(t, tc.description, text)
				return string(tc.language), nil
			}, TranslateFunc: func(string, string, string) (string, error) {
				t.Error("idioms should not be translated")
				return "", errors.New("unexpected translation")
			}})
			assert.Equal(t, tc.description, readIdiomDescription(t, user, selection.ID))
			assert.Equal(t, tc.description, readIdiomDescription(t, user, selection.ID))
			assert.Equal(t, 1, calls)
			var rows []models.WordDescription
			require.NoError(t, db.DB.Find(&rows).Error)
			require.Len(t, rows, 1)
			assert.Nil(t, rows[0].TranslationWordID)
			assert.Equal(t, word.ID, rows[0].WordID)
			assert.Equal(t, config.GetOpenRouterModel(), rows[0].Model)
			assert.False(t, rows[0].CreatedAt.IsZero())
		})
	}
}

func TestDailyIdiomDescriptionFailureIsRetryableWithoutReselection(t *testing.T) {
	for _, scenario := range []string{"provider", "nil", "empty", "long", "answer", "uppercase", "period", "newline", "empty meaning", "language", "language failure"} {
		t.Run(scenario, func(t *testing.T) {
			testkit.Truncate(t)
			user := testkit.CreateUser(t)
			seedDailyIdiomWord(t, "under the weather", enums.LanguageEn)
			rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/daily-idiom", nil)
			testkit.RequireStatus(t, rec, http.StatusOK)
			var selected services.DailyIdiomResponse
			testkit.DecodeJSON(t, rec, &selected)
			require.NotNil(t, selected.Idiom)
			failed := true
			testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{GenerateIdiomDescriptionFunc: func(context.Context, string, string) (*openrouter.GeneratedDescription, error) {
				text := "sick, unwell, indisposed"
				if failed {
					switch scenario {
					case "provider":
						return nil, errors.New("unavailable")
					case "nil":
						return nil, nil
					case "empty":
						text = " "
					case "long":
						text = strings.Repeat("a", 301)
					case "answer":
						text = "under the weather means sick"
					case "uppercase":
						text = "Sick"
					case "period":
						text = "sick."
					case "newline":
						text = "sick\nunwell"
					case "empty meaning":
						text = "sick; ; unwell"
					}
				}
				return &openrouter.GeneratedDescription{Description: text}, nil
			}})
			testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{DetectFunc: func(string) (string, error) {
				if failed && scenario == "language" {
					return "it", nil
				}
				if failed && scenario == "language failure" {
					return "", errors.New("unavailable")
				}
				return "en", nil
			}})
			testkit.RequireStatus(t, testkit.AuthedRequest(t, user, http.MethodGet, idiomDescriptionPath(selected.Idiom.ID), nil), http.StatusServiceUnavailable)
			var count int64
			require.NoError(t, db.DB.Model(&models.WordDescription{}).Count(&count).Error)
			assert.Zero(t, count)
			failed = false
			assert.Equal(t, "sick, unwell, indisposed", readIdiomDescription(t, user, selected.Idiom.ID))
			retry := testkit.AuthedRequest(t, user, http.MethodGet, "/api/daily-idiom", nil)
			testkit.RequireStatus(t, retry, http.StatusOK)
			assert.JSONEq(t, rec.Body.String(), retry.Body.String())
			assertDailyIdiomCount(t, 1)
		})
	}
}

func TestDailyIdiomDescriptionConcurrentMissesDoNotBlockSelection(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	word := seedDailyIdiomWord(t, "under the weather", enums.LanguageEn)
	now := time.Now().UTC().Truncate(24 * time.Hour)
	selection := seedIdiomSelection(t, word, now)
	started, release := make(chan struct{}, 1), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	var calls atomic.Int32
	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{GenerateIdiomDescriptionFunc: func(ctx context.Context, idiom, language string) (*openrouter.GeneratedDescription, error) {
		calls.Add(1)
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return &openrouter.GeneratedDescription{Description: "sick, unwell"}, nil
	}})
	const requests = 8
	results := make(chan int, requests)
	for range requests {
		go func() {
			rec := testkit.AuthedRequest(t, user, http.MethodGet, idiomDescriptionPath(selection.ID), nil)
			results <- rec.Code
		}()
	}
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("generation did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	next, err := services.GetDailyIdiom(ctx, user.ID, now.AddDate(0, 0, 1))
	assert.NoError(t, err, "generation must release the daily selection lock")
	if next != nil {
		assert.NotNil(t, next.Idiom)
	}
	unblock()
	for range requests {
		select {
		case status := <-results:
			assert.Equal(t, http.StatusOK, status)
		case <-time.After(5 * time.Second):
			t.Fatal("description request did not finish")
		}
	}
	assert.EqualValues(t, 1, calls.Load())
	var count int64
	require.NoError(t, db.DB.Model(&models.WordDescription{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestIdiomDescriptionDoesNotReplaceExercisePairGeneration(t *testing.T) {
	testkit.Truncate(t)
	word := seedDailyIdiomWord(t, "under the weather", enums.LanguageEn)
	translated := models.Word{Word: "нездоровый", Language: enums.LanguageRu}
	require.NoError(t, db.DB.Create(&translated).Error)
	cached := models.WordDescription{WordID: word.ID, Model: "idiom", Description: "sick, unwell"}
	require.NoError(t, db.DB.Create(&cached).Error)
	calls := 0
	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{GenerateDescriptionFunc: func(w, lang, tr, trlang, descLang string) (*openrouter.GeneratedDescription, error) {
		calls++
		assert.Equal(t, word.Word, w)
		assert.Equal(t, translated.Word, tr)
		return &openrouter.GeneratedDescription{Description: "Feeling slightly ill."}, nil
	}})
	exercise, err := services.GetOrCreateWordDescription(word.ID, translated.ID)
	require.NoError(t, err)
	assert.NotEqual(t, cached.ID, exercise.ID)
	assert.Equal(t, &translated.ID, exercise.TranslationWordID)
	assert.Equal(t, 1, calls)
}

func TestAdminIdiomDescriptionPreviewAndReplacement(t *testing.T) {
	testkit.Truncate(t)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	word := seedDailyIdiomWord(t, "under the weather", enums.LanguageEn)
	existing := models.WordDescription{WordID: word.ID, Model: config.GetOpenRouterModel(), Description: "sick, unwell"}
	require.NoError(t, db.DB.Create(&existing).Error)
	calls := 0
	mockAdminDescription(t, &testkit.FakeOpenRouter{GenerateIdiomDescriptionFunc: func(context.Context, string, string) (*openrouter.GeneratedDescription, error) {
		calls++
		return &openrouter.GeneratedDescription{Description: "sick, unwell, indisposed"}, nil
	}}, config.GetOpenRouterModel())
	list := testkit.AuthedRequest(t, admin, http.MethodGet, "/api/admin/word-descriptions", nil)
	testkit.RequireStatus(t, list, http.StatusOK)
	var response services.AdminWordDescriptionsResponse
	testkit.DecodeJSON(t, list, &response)
	require.Len(t, response.Data, 1)
	assert.Empty(t, response.Data[0].Translation)
	preview := previewAdminDescription(t, admin, existing, config.GetOpenRouterModel())
	assert.Nil(t, preview.TranslationWordID)
	assert.Empty(t, preview.Translation)
	assert.Equal(t, existing.Description, preview.OriginalDescription)
	var stored models.WordDescription
	require.NoError(t, db.DB.First(&stored, "id = ?", existing.ID).Error)
	assert.Equal(t, existing.Description, stored.Description)
	path := fmt.Sprintf("/api/admin/word-descriptions/%s/approve", existing.ID)
	invalid := preview
	otherID := uuid.New()
	invalid.TranslationWordID = &otherID
	testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, path, invalid), http.StatusBadRequest)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, admin, http.MethodPost, path, preview), http.StatusOK)
	require.NoError(t, db.DB.First(&stored, "id = ?", existing.ID).Error)
	assert.Equal(t, preview.Description, stored.Description)
	assert.Nil(t, stored.TranslationWordID)
	assert.Equal(t, 1, calls)
	selection := seedIdiomSelection(t, word, time.Now())
	assert.Equal(t, preview.Description, readIdiomDescription(t, admin, selection.ID))
}
