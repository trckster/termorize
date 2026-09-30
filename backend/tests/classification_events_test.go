package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryWordCreationPathEmitsCommittedIDs(t *testing.T) {
	for _, name := range []string{"vocabulary", "google translation", "guest seed", "collection", "generated collection", "idiom translation", "dictionary import"} {
		t.Run(name, func(t *testing.T) {
			testkit.Truncate(t)
			user := testkit.CreateUser(t, testkit.WithAdmin())
			var events []uuid.UUID
			t.Cleanup(db.SetWordCreatedHandler(func(id uuid.UUID) bool {
				var visible models.Word
				assert.NoError(t, db.DB.First(&visible, "id = ?", id).Error)
				assert.Nil(t, visible.PartOfSpeech)
				events = append(events, id)
				return true
			}))
			var excluded uuid.UUID
			switch name {
			case "vocabulary":
				rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/vocabulary", services.CreateVocabularyRequest{Original: "book", Translation: "libro", OriginalLanguage: enums.LanguageEn, TranslationLanguage: enums.LanguageIt})
				testkit.RequireStatus(t, rec, http.StatusCreated)
			case "google translation":
				_, err := services.Translate("book", enums.LanguageEn, enums.LanguageIt)
				require.NoError(t, err)
			case "guest seed":
				_, err := services.CreateGuestUser("UTC", enums.LanguageEn)
				require.NoError(t, err)
			case "collection":
				collection, err := services.CreateCollection(user.ID, services.CreateCollectionRequest{Title: "Words"})
				require.NoError(t, err)
				_, err = services.AddTranslationToCollection(user.ID, collection.ID, services.AddCollectionTranslationRequest{Original: "book", Translation: "libro", OriginalLanguage: enums.LanguageEn, TranslationLanguage: enums.LanguageIt})
				require.NoError(t, err)
			case "generated collection":
				testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{GenerateFunc: func(string, []string) (*openrouter.GeneratedCollection, error) {
					return &openrouter.GeneratedCollection{Title: "Words", Translations: []openrouter.GeneratedTranslation{{Original: "book", Translation: "libro", OriginalLanguage: "en", TranslationLanguage: "it"}}}, nil
				}})
				_, err := services.GenerateCollection(user.ID, "words")
				require.NoError(t, err)
			case "idiom translation":
				word := seedDailyIdiomWord(t, "spill the beans", enums.LanguageEn)
				excluded = word.ID
				selection := seedIdiomSelection(t, word, time.Now().UTC())
				_, err := services.TranslateDailyIdiom(context.Background(), selection.ID, enums.LanguageIt)
				require.NoError(t, err)
			case "dictionary import":
				data := gzipDictionary(t, `{"word":"spill the beans","lang_code":"en","senses":[{"tags":["idiomatic"]}]}`+"\n")
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(data) }))
				defer server.Close()
				dictionary := seedDictionary(t, "enwiktionary", server.URL)
				_, err := services.StartDictionaryImport(dictionary.ID)
				require.NoError(t, err)
				require.NoError(t, newDictionaryWorker(t, server.Client()).Run(context.Background()))
			}
			var ids []uuid.UUID
			require.NoError(t, db.DB.Model(&models.Word{}).Where("id <> ?", excluded).Pluck("id", &ids).Error)
			require.NotEmpty(t, ids)
			assert.ElementsMatch(t, ids, events)
		})
	}
}

func TestFailedEventDeliveryLeavesRecoverableNULL(t *testing.T) {
	testkit.Truncate(t)
	t.Cleanup(db.SetWordCreatedHandler(func(uuid.UUID) bool { return false }))
	word, err := services.GetOrCreateWord(db.DB, "missed event", enums.LanguageEn)
	require.NoError(t, err)
	var saved models.Word
	require.NoError(t, db.DB.First(&saved, "id = ?", word.ID).Error)
	assert.Nil(t, saved.PartOfSpeech)
}
