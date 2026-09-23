package tests

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Contract: authenticated GET with a required supported language; shared local-date
// assignments, explicit empty results, stable replay, exact below-maximum balancing,
// and serialized first requests across the same or different dates in a language.
func TestDailyIdiomEndpoint(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{TimeZone: "Pacific/Kiritimati"}))
	other := testkit.CreateUser(t, testkit.WithSettings(user.Settings))
	path := "/api/daily-idiom?language=en"
	testkit.RequireStatus(t, testkit.Request(t, http.MethodGet, path, nil), http.StatusUnauthorized)
	stale := testkit.CreateUser(t)
	require.NoError(t, db.DB.Delete(&stale).Error)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, stale, http.MethodGet, path, nil), http.StatusUnauthorized)
	for _, query := range []string{"", "?language=", "?language=zz", "?language=EN"} {
		rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/daily-idiom"+query, nil)
		testkit.RequireStatus(t, rec, http.StatusBadRequest)
		assert.JSONEq(t, `{"error":"language must be a supported language"}`, rec.Body.String())
	}

	unknown := models.Word{Word: "ordinary", Language: enums.LanguageEn}
	require.NoError(t, db.DB.Create(&unknown).Error)
	seedDailyIdiomWord(t, "бить баклуши", enums.LanguageRu)
	rec := testkit.AuthedRequest(t, user, http.MethodGet, path, nil)
	testkit.RequireStatus(t, rec, http.StatusOK)
	var empty services.DailyIdiomResponse
	testkit.DecodeJSON(t, rec, &empty)
	assert.Nil(t, empty.Idiom)
	assert.Contains(t, rec.Body.String(), `"idiom":null`)
	assert.Equal(t, enums.LanguageEn, empty.Language)
	assertDailyIdiomCount(t, 0)

	word := seedDailyIdiomWord(t, "piece of cake", enums.LanguageEn)
	before := time.Now()
	rec = testkit.AuthedRequest(t, user, http.MethodGet, path, nil)
	after := time.Now()
	testkit.RequireStatus(t, rec, http.StatusOK)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	var selected services.DailyIdiomResponse
	testkit.DecodeJSON(t, rec, &selected)
	require.NotNil(t, selected.Idiom)
	assert.NotEqual(t, uuid.Nil, selected.Idiom.ID)
	assert.Equal(t, word.ID, selected.Idiom.WordID)
	assert.Equal(t, word.Word, selected.Idiom.Word)
	location, err := time.LoadLocation(user.Settings.TimeZone)
	require.NoError(t, err)
	assert.Contains(t, []string{before.In(location).Format(time.DateOnly), after.In(location).Format(time.DateOnly)}, selected.Date)

	seedDailyIdiomWord(t, "break the ice", enums.LanguageEn)
	rec = testkit.AuthedRequest(t, other, http.MethodGet, path, nil)
	testkit.RequireStatus(t, rec, http.StatusOK)
	var replay services.DailyIdiomResponse
	testkit.DecodeJSON(t, rec, &replay)
	assert.Equal(t, selected, replay)
	assertDailyIdiomCount(t, 1)
	var saved models.DailyIdiom
	require.NoError(t, db.DB.First(&saved, "id = ?", selected.Idiom.ID).Error)
	assert.Equal(t, selected.Date, saved.Date.Format(time.DateOnly))
	assert.Equal(t, word.ID, saved.WordID)
	assert.Equal(t, enums.LanguageEn, saved.Language)
	for _, table := range []string{"vocabulary", "translations", "word_descriptions", "dictionary_import_jobs"} {
		var count int64
		require.NoError(t, db.DB.Table(table).Count(&count).Error)
		assert.Zero(t, count, table)
	}
}

func TestDailyIdiomDatesAndCycles(t *testing.T) {
	testkit.Truncate(t)
	west := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{TimeZone: "America/Los_Angeles"}))
	east := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{TimeZone: "Pacific/Kiritimati"}))
	for i := range 3 {
		seedDailyIdiomWord(t, fmt.Sprintf("idiom %d", i), enums.LanguageEn)
	}
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	first := requestDailyIdiomAt(t, west, enums.LanguageEn, now)
	second := requestDailyIdiomAt(t, east, enums.LanguageEn, now)
	assert.Equal(t, "2026-09-23", first.Date)
	assert.Equal(t, "2026-09-24", second.Date)
	assert.NotEqual(t, first.Idiom.WordID, second.Idiom.WordID)
	assert.Equal(t, first, requestDailyIdiomAt(t, west, enums.LanguageEn, now.Add(time.Hour)))
	seen := map[uuid.UUID]bool{first.Idiom.WordID: true, second.Idiom.WordID: true}
	third := requestDailyIdiomAt(t, west, enums.LanguageEn, now.AddDate(0, 0, 10))
	assert.False(t, seen[third.Idiom.WordID])
	assertDailyIdiomCount(t, 3)
	for cycle := range 2 {
		seen = make(map[uuid.UUID]bool)
		for day := range 3 {
			selected := requestDailyIdiomAt(t, west, enums.LanguageEn, now.AddDate(0, 0, 20+cycle*3+day))
			assert.False(t, seen[selected.Idiom.WordID], "each word appears once per cycle")
			seen[selected.Idiom.WordID] = true
		}
	}
	assertDailyIdiomCount(t, 9)
	newcomer := seedDailyIdiomWord(t, "new idiom", enums.LanguageEn)
	selected := requestDailyIdiomAt(t, west, enums.LanguageEn, now.AddDate(0, 0, 40))
	assert.Equal(t, newcomer.ID, selected.Idiom.WordID)
	assert.Equal(t, first, requestDailyIdiomAt(t, west, enums.LanguageEn, now))
}

func TestDailyIdiomBelowMaximumAndIndependentLanguages(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	unused := seedDailyIdiomWord(t, "unused", enums.LanguageEn)
	middle := seedDailyIdiomWord(t, "middle", enums.LanguageEn)
	highest := seedDailyIdiomWord(t, "highest", enums.LanguageEn)
	italian := seedDailyIdiomWord(t, "in bocca al lupo", enums.LanguageIt)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, word := range []models.Word{middle, highest, highest, highest, italian, italian, italian, italian, italian} {
		require.NoError(t, db.DB.Create(&models.DailyIdiom{Date: start.AddDate(0, 0, i), Language: word.Language, WordID: word.ID}).Error)
	}
	// Pin the connection and seed PostgreSQL's random stream to make repeated
	// candidate sampling reproducible without replacing the production query.
	originalDB := db.DB
	require.NoError(t, originalDB.Connection(func(conn *gorm.DB) error {
		db.DB = conn.Session(&gorm.Session{NewDB: true})
		defer func() { db.DB = originalDB }()
		require.NoError(t, db.DB.Exec("SELECT setseed(0.5)").Error)
		seen := make(map[uuid.UUID]bool)
		for range 32 {
			selected := requestDailyIdiomAt(t, user, enums.LanguageEn, start.AddDate(0, 1, 0))
			assert.NotEqual(t, highest.ID, selected.Idiom.WordID, "another language's count must not make the English maximum eligible")
			seen[selected.Idiom.WordID] = true
			require.NoError(t, db.DB.Delete(&models.DailyIdiom{}, "id = ?", selected.Idiom.ID).Error)
		}
		assert.True(t, seen[unused.ID], "zero-use idioms must be eligible")
		assert.True(t, seen[middle.ID], "below-maximum selection must include non-minimum counts")
		return nil
	}))
	selected := requestDailyIdiomAt(t, user, enums.LanguageIt, start.AddDate(0, 1, 0))
	assert.Equal(t, italian.ID, selected.Idiom.WordID, "a single idiom can recur indefinitely")
}

func TestDailyIdiomConcurrentRequests(t *testing.T) {
	for _, differentDates := range []bool{false, true} {
		t.Run(fmt.Sprintf("different_dates=%t", differentDates), func(t *testing.T) {
			testkit.Truncate(t)
			user := testkit.CreateUser(t)
			const requests = 8
			for i := range requests {
				seedDailyIdiomWord(t, fmt.Sprintf("idiom %d", i), enums.LanguageEn)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			results := make([]*services.DailyIdiomResponse, requests)
			errors := make([]error, requests)
			start := make(chan struct{})
			var wg sync.WaitGroup
			for i := range requests {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
					if differentDates {
						now = now.AddDate(0, 0, i)
					}
					results[i], errors[i] = services.GetDailyIdiom(ctx, user.ID, enums.LanguageEn, now)
				}()
			}
			close(start)
			wg.Wait()
			seen := make(map[uuid.UUID]bool)
			for i, result := range results {
				require.NoError(t, errors[i])
				require.NotNil(t, result.Idiom)
				seen[result.Idiom.WordID] = true
				if !differentDates {
					assert.Equal(t, results[0], result)
				}
			}
			if differentDates {
				assert.Len(t, seen, requests, "usage is recalculated after every committed selection")
				assertDailyIdiomCount(t, requests)
			} else {
				assertDailyIdiomCount(t, 1)
			}
		})
	}
}

func seedDailyIdiomWord(t *testing.T, text string, language enums.Language) models.Word {
	t.Helper()
	word := models.Word{Word: text, Language: language, Type: enums.TypeIdiom}
	require.NoError(t, db.DB.Create(&word).Error)
	return word
}

func requestDailyIdiomAt(t *testing.T, user models.User, language enums.Language, now time.Time) *services.DailyIdiomResponse {
	t.Helper()
	result, err := services.GetDailyIdiom(context.Background(), user.ID, language, now)
	require.NoError(t, err)
	require.NotNil(t, result.Idiom)
	return result
}

func assertDailyIdiomCount(t *testing.T, expected int64) {
	t.Helper()
	var count int64
	require.NoError(t, db.DB.Model(&models.DailyIdiom{}).Count(&count).Error)
	assert.Equal(t, expected, count)
}
