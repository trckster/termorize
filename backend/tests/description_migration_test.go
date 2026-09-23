package tests

import (
	"os"
	"testing"
	"time"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/testkit"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type legacyWordDescription struct {
	ID                uuid.UUID `gorm:"default:gen_random_uuid()"`
	WordID            uuid.UUID
	TranslationWordID *uuid.UUID
	Model             string
	Description       string
	CreatedAt         time.Time
	ApprovedAt        *time.Time
}

func (legacyWordDescription) TableName() string { return "pg_temp.word_descriptions" }

// Recreate the nullable pre-upgrade cache on one connection without altering
// the real schema used by the application tests.
func descriptionMigrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	tx := db.DB.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })
	require.NoError(t, tx.Exec(`
		CREATE TEMP TABLE word_descriptions
		    (LIKE public.word_descriptions INCLUDING DEFAULTS INCLUDING INDEXES) ON COMMIT DROP;
		ALTER TABLE word_descriptions ALTER COLUMN translation_word_id DROP NOT NULL;
		ALTER TABLE word_descriptions ADD COLUMN approved_at TIMESTAMP;
		ALTER TABLE word_descriptions ADD CONSTRAINT uq_word_descriptions_word_translation_model
		    UNIQUE NULLS NOT DISTINCT (word_id, translation_word_id, model);
		CREATE TEMP TABLE word_description_backfill_archive
		    (LIKE public.word_description_backfill_archive INCLUDING DEFAULTS INCLUDING INDEXES) ON COMMIT DROP;
	`).Error)
	return tx
}

func TestDescriptionContextMigrationRequiresFirstSavedCounterpart(t *testing.T) {
	migration, err := os.ReadFile("src/data/migrations/0021_require_description_translation_context.sql")
	require.NoError(t, err)
	for _, test := range []struct {
		name          string
		pairs         [][3]int // original index, translated index, creation offset
		initial       int
		expected      int
		conflict      bool
		archiveReason string
	}{
		{name: "earliest outgoing", pairs: [][3]int{{0, 1, 10}, {0, 2, 0}}, expected: 2},
		{name: "earliest incoming", pairs: [][3]int{{1, 0, 10}, {2, 0, 0}}, expected: 2},
		{name: "earliest across directions", pairs: [][3]int{{0, 1, 10}, {2, 0, 0}}, expected: 2},
		{name: "timestamp tie uses ID", pairs: [][3]int{{0, 1, 0}, {2, 0, 0}}, expected: 2},
		{name: "duplicate pairs", pairs: [][3]int{{0, 1, 0}, {0, 1, 10}, {1, 0, 20}}, expected: 1},
		{name: "existing context", pairs: [][3]int{{0, 1, 0}}, initial: 2, expected: 2},
		{name: "missing counterpart", archiveReason: "no_translation"},
		{name: "existing cache wins", pairs: [][3]int{{0, 1, 0}, {0, 2, 10}}, conflict: true, archiveReason: "existing_context"},
	} {
		t.Run(test.name, func(t *testing.T) {
			testkit.Truncate(t)
			conn := descriptionMigrationDB(t)
			words := []models.Word{
				{Word: "bank", Language: enums.LanguageEn},
				{Word: "la banca", Language: enums.LanguageIt},
				{Word: "la riva", Language: enums.LanguageIt},
			}
			require.NoError(t, conn.Create(&words).Error)
			now := time.Now().UTC().Truncate(time.Microsecond)
			for index, pair := range test.pairs {
				translation := models.Translation{
					ID:         uuid.UUID{15: byte(len(test.pairs) - index)},
					OriginalID: words[pair[0]].ID, TranslationID: words[pair[1]].ID,
					Source: enums.TranslationSourceGoogle, CreatedAt: now.Add(time.Duration(pair[2]) * time.Second),
				}
				if index > 0 {
					user := testkit.CreateUser(t)
					translation.Source, translation.UserID = enums.TranslationSourceUser, &user.ID
				}
				require.NoError(t, conn.Create(&translation).Error)
			}
			legacy := legacyWordDescription{
				WordID: words[0].ID, Model: "model", Description: "Original approved clue.", CreatedAt: now, ApprovedAt: &now,
			}
			if test.initial != 0 {
				legacy.TranslationWordID = &words[test.initial].ID
			}
			require.NoError(t, conn.Create(&legacy).Error)
			contextual := legacyWordDescription{
				WordID: words[0].ID, TranslationWordID: &words[1].ID, Model: "model", Description: "Newer contextual clue.",
			}
			if test.conflict {
				require.NoError(t, conn.Create(&contextual).Error)
			}
			for range 2 {
				require.NoError(t, conn.Exec(string(migration)).Error)
				var stored legacyWordDescription
				if test.archiveReason != "" {
					assert.ErrorIs(t, conn.First(&stored, "id = ?", legacy.ID).Error, gorm.ErrRecordNotFound)
					require.NoError(t, conn.Table("word_description_backfill_archive").First(&stored, "id = ?", legacy.ID).Error)
					var reason string
					require.NoError(t, conn.Table("word_description_backfill_archive").Select("reason").Where("id = ?", legacy.ID).Scan(&reason).Error)
					assert.Equal(t, test.archiveReason, reason)
				} else {
					require.NoError(t, conn.First(&stored, "id = ?", legacy.ID).Error)
					assert.Equal(t, &words[test.expected].ID, stored.TranslationWordID)
				}
				assert.Equal(t, legacy.Description, stored.Description)
				assert.Equal(t, legacy.Model, stored.Model)
				assert.WithinDuration(t, now, stored.CreatedAt, time.Microsecond)
				require.NotNil(t, stored.ApprovedAt)
				assert.WithinDuration(t, now, *stored.ApprovedAt, time.Microsecond)
				if test.conflict {
					var preserved legacyWordDescription
					require.NoError(t, conn.First(&preserved, "id = ?", contextual.ID).Error)
					assert.Equal(t, contextual.Description, preserved.Description)
					assert.Equal(t, contextual.TranslationWordID, preserved.TranslationWordID)
				}
				var missing int64
				require.NoError(t, conn.Model(&legacyWordDescription{}).Where("translation_word_id IS NULL").Count(&missing).Error)
				assert.Zero(t, missing)
			}
			invalid := legacyWordDescription{WordID: words[0].ID, Model: "another-model", Description: "Missing context."}
			var pgError *pgconn.PgError
			require.ErrorAs(t, conn.Create(&invalid).Error, &pgError)
			assert.Equal(t, "23502", pgError.Code)
			assert.Equal(t, "translation_word_id", pgError.ColumnName)
		})
	}
}

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
			conn := descriptionMigrationDB(t)
			words := []models.Word{
				{Word: "bank", Language: enums.LanguageEn},
				{Word: "la banca", Language: enums.LanguageIt},
				{Word: "la riva", Language: enums.LanguageIt},
			}
			require.NoError(t, conn.Create(&words).Error)
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
				require.NoError(t, conn.Create(&translation).Error)
			}
			now := time.Now().UTC().Truncate(time.Microsecond)
			legacy := legacyWordDescription{
				WordID: words[0].ID, Model: "legacy-model", Description: "An approved existing clue.",
				CreatedAt: now, ApprovedAt: &now,
			}
			if test.initial != 0 {
				legacy.TranslationWordID = &words[test.initial].ID
			}
			require.NoError(t, conn.Create(&legacy).Error)
			var contextual legacyWordDescription
			if test.conflictModel != "" {
				contextual = legacyWordDescription{
					WordID: words[0].ID, TranslationWordID: &words[1].ID,
					Model: test.conflictModel, Description: "A newer contextual clue.",
				}
				require.NoError(t, conn.Create(&contextual).Error)
			}

			for range 2 {
				require.NoError(t, conn.Exec(string(migration)).Error, "backfill must also be safe to rerun")
				var stored legacyWordDescription
				require.NoError(t, conn.First(&stored, "id = ?", legacy.ID).Error)
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
				require.NoError(t, conn.Model(&legacyWordDescription{}).Count(&count).Error)
				expectedCount := int64(1)
				if test.conflictModel != "" {
					expectedCount++
					var preserved legacyWordDescription
					require.NoError(t, conn.First(&preserved, "id = ?", contextual.ID).Error)
					assert.Equal(t, contextual.Description, preserved.Description)
					assert.Equal(t, contextual.TranslationWordID, preserved.TranslationWordID)
				}
				assert.Equal(t, expectedCount, count)
			}
		})
	}
}

func TestRemoveDescriptionApprovalPreservesAllRows(t *testing.T) {
	testkit.Truncate(t)
	conn := descriptionMigrationDB(t)
	wordID, translationID := uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	rows := []legacyWordDescription{
		{WordID: wordID, TranslationWordID: &translationID, Model: "old-model", Description: "Approved clue.", CreatedAt: now.Add(-time.Hour), ApprovedAt: &now},
		{WordID: wordID, TranslationWordID: &translationID, Model: "new-model", Description: "Latest clue.", CreatedAt: now},
	}
	require.NoError(t, conn.Create(&rows).Error)
	migration, err := os.ReadFile("src/data/migrations/0024_remove_description_approval.sql")
	require.NoError(t, err)
	require.NoError(t, conn.Exec(string(migration)).Error)
	var stored []models.WordDescription
	require.NoError(t, conn.Order("created_at").Find(&stored).Error)
	require.Len(t, stored, len(rows))
	for i, row := range rows {
		assert.Equal(t, row.ID, stored[i].ID)
		assert.Equal(t, row.WordID, stored[i].WordID)
		assert.Equal(t, row.TranslationWordID, stored[i].TranslationWordID)
		assert.Equal(t, row.Model, stored[i].Model)
		assert.Equal(t, row.Description, stored[i].Description)
		assert.WithinDuration(t, row.CreatedAt, stored[i].CreatedAt, time.Microsecond)
	}
	var approvalColumns int64
	require.NoError(t, conn.Raw(`SELECT count(*) FROM pg_attribute
		WHERE attrelid = 'pg_temp.word_descriptions'::regclass
		AND attname = 'approved_at' AND NOT attisdropped`).Scan(&approvalColumns).Error)
	assert.Zero(t, approvalColumns)
	require.NoError(t, conn.Model(&models.WordDescription{}).Where("id = ?", rows[0].ID).Update("model", rows[1].Model).Error)
}
