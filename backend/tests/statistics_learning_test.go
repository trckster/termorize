package tests

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExerciseStatisticsLearningDistribution(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	other := testkit.CreateUser(t)

	read := func() services.VocabularyLearningDistribution {
		rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises/statistics", nil)
		testkit.RequireStatus(t, rec, http.StatusOK)
		var body services.ExerciseStatistics
		testkit.DecodeJSON(t, rec, &body)
		return body.VocabularyLearning
	}
	assert.Equal(t, services.VocabularyLearningDistribution{}, read())

	for i, knowledge := range []int{0, 1, 34, 35, 69, 70, 99, 100} {
		vocab := exerciseSeedVocabulary(t, user.ID, fmt.Sprintf("word%d", i), fmt.Sprintf("wort%d", i), enums.LanguageEn, enums.LanguageDe)
		progress := models.ProgressEntries{{Type: enums.KnowledgeTypeTranslation, Knowledge: knowledge}}
		require.NoError(t, db.DB.Model(&vocab).UpdateColumn("progress", progress).Error)
	}
	missing := exerciseSeedVocabulary(t, user.ID, "missing", "fehlt", enums.LanguageEn, enums.LanguageDe)
	require.NoError(t, db.DB.Model(&missing).UpdateColumn("progress", models.ProgressEntries{}).Error)
	deleted := exerciseSeedVocabulary(t, user.ID, "deleted", "gelöscht", enums.LanguageEn, enums.LanguageDe)
	require.NoError(t, db.DB.Model(&deleted).UpdateColumn("deleted_at", time.Now()).Error)
	exerciseSeedVocabulary(t, other.ID, "other", "andere", enums.LanguageEn, enums.LanguageDe)

	assert.Equal(t, services.VocabularyLearningDistribution{
		NotStarted: 2, Beginning: 2, Developing: 2, Confident: 2, Mastered: 1,
	}, read())
}
