package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
)

func importAllLanguageFixtures(t *testing.T) {
	t.Helper()
	seedAllDictionarySources(t)
	sources := map[string]string{"enwiktionary": "en", "ruwiktionary": "ru", "itwiktionary": "it", "dewiktionary": "de", "enwiktionary-es": "es", "frwiktionary": "fr", "plwiktionary": "pl", "trwiktionary": "tr", "enwiktionary-pt": "pt", "ukwiktionary": "uk"}
	categories := map[string]string{"es": "Category:Spanish idioms", "pt": "Category:Portuguese idioms", "uk": "Категорія:Фразеологізми/uk"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		edition := strings.TrimPrefix(r.URL.Path, "/")
		lang, ok := sources[edition]
		require.True(t, ok)
		if category, ok := categories[lang]; ok {
			assert.Equal(t, category, r.URL.Query().Get("gcmtitle"))
			json.NewEncoder(w).Encode(map[string]any{"query": map[string]any{"pages": []map[string]any{{"ns": 0, "title": "first " + lang + " idiom"}, {"ns": 0, "title": "second " + lang + " idiom"}}}})
			return
		}
		classification := `"tags":["idiomatic"]`
		switch edition {
		case "enwiktionary":
			classification = `"categories":["English idioms"]`
		case "ruwiktionary":
			classification = `"categories":["Фразеологизмы/ru"]`
		case "itwiktionary":
			classification = `"raw_tags":["idiomatico"]`
		case "dewiktionary":
			classification = `"categories":["Redewendung (Deutsch)"]`
		case "frwiktionary":
			classification = `"categories":["Idiotismes animaliers en français"]`
		case "trwiktionary":
			classification = `"categories":["Türkçe deyimler"]`
		}
		payload := fmt.Sprintf(`{"word":"first %s idiom","lang_code":"%s",%s}`+"\n"+`{"word":"second %s idiom","lang_code":"%s","senses":[{%s}]}`, lang, lang, classification, lang, lang, classification)
		w.Write(gzipDictionary(t, payload))
	}))
	defer server.Close()
	var dictionaries []models.Dictionary
	require.NoError(t, db.DB.Find(&dictionaries).Error)
	require.Len(t, dictionaries, len(enums.AllLanguages()))
	worker := newDictionaryWorker(t, server.Client())
	for _, source := range dictionaries {
		require.NoError(t, db.DB.Model(&source).Update("download_url", server.URL+"/"+source.Edition).Error)
		for attempt := 0; attempt < 2; attempt++ {
			job, err := services.StartDictionaryImport(source.ID)
			require.NoError(t, err)
			require.NoError(t, worker.Run(context.Background()))
			result := loadDictionaryJob(t, job.ID)
			require.Equal(t, "succeeded", result.Status, result.Error)
			require.Zero(t, result.Failed)
			if attempt == 0 {
				require.EqualValues(t, 2, result.Inserted)
			} else {
				require.EqualValues(t, 2, result.Skipped)
			}
		}
	}
	var words []models.Word
	require.NoError(t, db.DB.Find(&words).Error)
	require.Len(t, words, 2*len(enums.AllLanguages()))
	for _, lang := range enums.AllLanguageValues() {
		count := 0
		for _, word := range words {
			if word.Language == lang && word.Type == enums.TypeIdiom {
				count++
			}
		}
		assert.Equal(t, 2, count, lang)
	}
}
