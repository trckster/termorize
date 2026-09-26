package tests

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"
	"testing"
)

func TestAdminIdiomCoverageIncludesEveryLanguageAndActualCounts(t *testing.T) {
	testkit.Truncate(t)
	const path = "/api/admin/dictionaries/coverage"
	testkit.RequireStatus(t, testkit.Request(t, http.MethodGet, path, nil), http.StatusUnauthorized)
	user := testkit.CreateUser(t)
	testkit.RequireStatus(t, testkit.AuthedRequest(t, user, http.MethodGet, path, nil), http.StatusForbidden)
	admin := testkit.CreateUser(t, testkit.WithAdmin())
	seedDailyIdiomWord(t, "break the ice", enums.LanguageEn)
	seedDailyIdiomWord(t, "piece of cake", enums.LanguageEn)
	seedDailyIdiomWord(t, "briser la glace", enums.LanguageFr)
	require.NoError(t, db.DB.Create(&models.Word{Word: "bonjour", Language: enums.LanguageFr, Type: enums.TypeUnknown}).Error)
	check := func() []services.IdiomLanguageCoverage {
		rec := testkit.AuthedRequest(t, admin, http.MethodGet, path, nil)
		testkit.RequireStatus(t, rec, http.StatusOK)
		assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
		var rows []services.IdiomLanguageCoverage
		testkit.DecodeJSON(t, rec, &rows)
		require.Len(t, rows, len(enums.AllLanguageValues()))
		for i, language := range enums.AllLanguageValues() {
			assert.Equal(t, language, rows[i].Language)
			assert.Equal(t, language.DisplayName(), rows[i].Name)
		}
		return rows
	}
	for _, row := range check() {
		expected := int64(0)
		if row.Language == enums.LanguageEn {
			expected = 2
		}
		if row.Language == enums.LanguageFr {
			expected = 1
		}
		assert.Equal(t, expected, row.IdiomCount, string(row.Language))
	}
	require.NoError(t, db.DB.Model(&models.Word{}).Where("word = ? AND language = ?", "bonjour", enums.LanguageFr).Update("type", enums.TypeIdiom).Error)
	for _, row := range check() {
		if row.Language == enums.LanguageFr {
			assert.EqualValues(t, 2, row.IdiomCount)
		}
	}
}
