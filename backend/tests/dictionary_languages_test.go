package tests

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"
	"testing"
)

func seedAllDictionarySources(t *testing.T) {
	t.Helper()
	old, err := os.ReadFile("src/data/migrations/0023_add_dictionary_imports.sql")
	require.NoError(t, err)
	statement := string(old[strings.Index(string(old), "INSERT INTO"):])
	require.NoError(t, db.DB.Exec(statement[:strings.Index(statement, ";\n")+1]).Error)
	for _, file := range []string{"0029_add_language_idiom_imports.sql", "0030_remove_dictionary_target_language.sql"} {
		migration, err := os.ReadFile("src/data/migrations/" + file)
		require.NoError(t, err)
		require.NoError(t, db.DB.Exec(string(migration)).Error)
	}
}

func TestAllLanguageSourcesAreSeededAndStartable(t *testing.T) {
	testkit.Truncate(t)
	seedAllDictionarySources(t)
	var sources []models.Dictionary
	require.NoError(t, db.DB.Find(&sources).Error)
	require.Len(t, sources, len(enums.AllLanguages()))
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	for _, source := range sources {
		rec := testkit.AuthedRequest(t, admin, http.MethodPost, "/api/admin/dictionaries/"+source.ID.String()+"/imports", nil)
		testkit.RequireStatus(t, rec, http.StatusAccepted)
		var job models.DictionaryImportJob
		testkit.DecodeJSON(t, rec, &job)
		assert.Equal(t, source.Edition, job.Edition)
		assert.Equal(t, source.DownloadURL, job.DownloadURL)
		assert.NotContains(t, rec.Body.String(), "target_language")
	}
}

func TestLanguageImportFiltersBySnapshottedEdition(t *testing.T) {
	testkit.Truncate(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(gzipDictionary(t, `{"word":"złote usta","lang_code":"pl","tags":["idiomatic"]}`+"\n"+`{"word":"break the ice","lang_code":"en","tags":["idiomatic"]}`))
	}))
	defer server.Close()
	source := seedDictionary(t, "plwiktionary", server.URL)
	job, err := services.StartDictionaryImport(source.ID)
	require.NoError(t, err)
	require.NoError(t, db.DB.Model(&source).Update("edition", "enwiktionary").Error)
	require.NoError(t, newDictionaryWorker(t, server.Client()).Run(context.Background()))
	loaded := loadDictionaryJob(t, job.ID)
	assert.Equal(t, "succeeded", loaded.Status)
	assert.EqualValues(t, 1, loaded.Inserted)
	assert.EqualValues(t, 1, loaded.Skipped)
	var words []models.Word
	require.NoError(t, db.DB.Find(&words).Error)
	require.Len(t, words, 1)
	assert.Equal(t, enums.LanguagePl, words[0].Language)
}

func TestCategoryImportsPaginateFilterAndDeduplicate(t *testing.T) {
	for _, tc := range []struct{ edition, lang, category string }{{"ukwiktionary", "uk", "Категорія:Фразеологізми/uk"}, {"enwiktionary-es", "es", "Category:Spanish idioms"}, {"enwiktionary-pt", "pt", "Category:Portuguese idioms"}} {
		t.Run(tc.lang, func(t *testing.T) {
			testkit.Truncate(t)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				assert.Equal(t, tc.category, r.URL.Query().Get("gcmtitle"))
				assert.Equal(t, "0", r.URL.Query().Get("gcmnamespace"))
				assert.NotEmpty(t, r.Header.Get("User-Agent"))
				if r.URL.Query().Get("gcmcontinue") == "" {
					w.Write([]byte(`{"continue":{"gcmcontinue":"next","continue":"gcmcontinue||"},"query":{"pages":[{"ns":0,"title":"first idiom"},{"ns":14,"title":"Category:ignore"},{"ns":0,"title":"redirect","redirect":true},{"ns":0,"title":"proverb","categories":[{"title":"proverb category"}]}]}}`))
					return
				}
				assert.Equal(t, "next", r.URL.Query().Get("gcmcontinue"))
				json.NewEncoder(w).Encode(map[string]any{"query": map[string]any{"pages": []map[string]any{{"ns": 0, "title": "second idiom"}, {"ns": 0, "title": "FIRST IDIOM"}}}})
			}))
			defer server.Close()
			source := seedDictionary(t, tc.edition, server.URL)
			worker := newDictionaryWorker(t, server.Client())
			for attempt := 0; attempt < 2; attempt++ {
				job, err := services.StartDictionaryImport(source.ID)
				require.NoError(t, err)
				require.NoError(t, worker.Run(context.Background()))
				loaded := loadDictionaryJob(t, job.ID)
				assert.Equal(t, "succeeded", loaded.Status, loaded.Error)
				assert.EqualValues(t, 0, loaded.Failed)
				expected := int64(2)
				if attempt == 1 {
					expected = 0
				}
				assert.Equal(t, expected, loaded.Inserted)
			}
			assert.Equal(t, 4, requests)
			var words []models.Word
			require.NoError(t, db.DB.Find(&words).Error)
			require.Len(t, words, 2)
			for _, word := range words {
				assert.Equal(t, enums.Language(tc.lang), word.Language)
				assert.Equal(t, enums.TypeIdiom, word.Type)
			}
		})
	}
}

func TestCategoryImportErrorsDoNotImportPartialData(t *testing.T) {
	for _, body := range []string{`{"error":{"code":"maxlag"}}`, `not json`, `{"query":{}}`, `{"continue":{"gcmcontinue":"same"},"query":{"pages":[{"ns":0,"title":"valid"}]}}`} {
		t.Run(body, func(t *testing.T) {
			testkit.Truncate(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
			defer server.Close()
			source := seedDictionary(t, "ukwiktionary", server.URL)
			job, err := services.StartDictionaryImport(source.ID)
			require.NoError(t, err)
			require.NoError(t, newDictionaryWorker(t, server.Client()).Run(context.Background()))
			loaded := loadDictionaryJob(t, job.ID)
			assert.Equal(t, "failed", loaded.Status)
			assert.NotEmpty(t, loaded.Error)
			var count int64
			require.NoError(t, db.DB.Model(&models.Word{}).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}
