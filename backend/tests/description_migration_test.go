package tests

import (
	"os"
	"testing"
	"time"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescriptionTranslationBackfill(t *testing.T) {
	migration, err := os.ReadFile("src/data/migrations/0020_backfill_description_translation_context.sql")
	require.NoError(t, err)

	for _, test := range []struct {
		name          string
		pairs         [][2]int
		initial       int
		expected      int
		conflictModel string
	}{
		{name: "original side", pairs: [][2]int{{0, 1}}, expected: 1},
		{name: "translated side", pairs: [][2]int{{1, 0}}, expected: 1},
		{name: "duplicate users sources and directions", pairs: [][2]int{{0, 1}, {0, 1}, {1, 0}}, expected: 1},
		{name: "multiple outgoing meanings", pairs: [][2]int{{0, 1}, {0, 2}}},
		{name: "multiple incoming meanings", pairs: [][2]int{{1, 0}, {2, 0}}},
		{name: "ambiguity across directions", pairs: [][2]int{{0, 1}, {2, 0}}},
		{name: "no translation"},
		{name: "existing context is untouched", pairs: [][2]int{{0, 1}}, initial: 2, expected: 2},
		{name: "existing cache entry wins", pairs: [][2]int{{0, 1}}, conflictModel: "legacy-model"},
		{name: "other model does not conflict", pairs: [][2]int{{0, 1}}, conflictModel: "other-model", expected: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			testkit.Truncate(t)
			words := []models.Word{
				{Word: "bank", Language: enums.LanguageEn},
				{Word: "la banca", Language: enums.LanguageIt},
				{Word: "la riva", Language: enums.LanguageIt},
			}
			require.NoError(t, db.DB.Create(&words).Error)
			for index, pair := range test.pairs {
				translation := models.Translation{
					OriginalID: words[pair[0]].ID, TranslationID: words[pair[1]].ID,
					Source: enums.TranslationSourceGoogle,
				}
				if index > 0 {
					user := testkit.CreateUser(t)
					translation.Source = enums.TranslationSourceUser
					translation.UserID = &user.ID
				}
				require.NoError(t, db.DB.Create(&translation).Error)
			}
			now := time.Now().UTC().Truncate(time.Microsecond)
			legacy := models.WordDescription{
				WordID: words[0].ID, Model: "legacy-model", Description: "An approved existing clue.",
				CreatedAt: now, ApprovedAt: &now,
			}
			if test.initial != 0 {
				legacy.TranslationWordID = &words[test.initial].ID
			}
			require.NoError(t, db.DB.Create(&legacy).Error)
			var contextual models.WordDescription
			if test.conflictModel != "" {
				contextual = models.WordDescription{
					WordID: words[0].ID, TranslationWordID: &words[1].ID,
					Model: test.conflictModel, Description: "A newer contextual clue.",
				}
				require.NoError(t, db.DB.Create(&contextual).Error)
			}

			for range 2 {
				require.NoError(t, db.DB.Exec(string(migration)).Error, "backfill must also be safe to rerun")
				var stored models.WordDescription
				require.NoError(t, db.DB.First(&stored, "id = ?", legacy.ID).Error)
				if test.expected == 0 {
					assert.Nil(t, stored.TranslationWordID)
				} else {
					assert.Equal(t, &words[test.expected].ID, stored.TranslationWordID)
				}
				assert.Equal(t, legacy.Description, stored.Description)
				assert.Equal(t, legacy.Model, stored.Model)
				assert.WithinDuration(t, now, stored.CreatedAt, time.Microsecond)
				require.NotNil(t, stored.ApprovedAt)
				assert.WithinDuration(t, now, *stored.ApprovedAt, time.Microsecond)
				var count int64
				require.NoError(t, db.DB.Model(&models.WordDescription{}).Count(&count).Error)
				expectedCount := int64(1)
				if test.conflictModel != "" {
					expectedCount++
					var preserved models.WordDescription
					require.NoError(t, db.DB.First(&preserved, "id = ?", contextual.ID).Error)
					assert.Equal(t, contextual.Description, preserved.Description)
					assert.Equal(t, contextual.TranslationWordID, preserved.TranslationWordID)
				}
				assert.Equal(t, expectedCount, count)
			}
		})
	}
}
