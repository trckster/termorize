package tests

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/testkit"
	"testing"
	"time"
)

func runIdiomMigration(t *testing.T, name string) {
	t.Helper()
	sql, err := os.ReadFile("src/data/migrations/" + name + ".sql")
	require.NoError(t, err)
	require.NoError(t, db.DB.Exec(string(sql)).Error)
}

func TestRetireStarterClassificationsPreservesWordsAndRelations(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	vocabulary := exerciseSeedVocabulary(t, user.ID, "BREAK THE ICE", "rompere il ghiaccio", enums.LanguageEn, enums.LanguageIt)
	var translation models.Translation
	require.NoError(t, db.DB.First(&translation, "id = ?", vocabulary.TranslationID).Error)
	unrelated := models.Word{Word: "another idiom", Language: enums.LanguageEn, Type: enums.TypeIdiom}
	require.NoError(t, db.DB.Create(&unrelated).Error)
	runIdiomMigration(t, "0028_seed_daily_idioms_all_languages")
	daily := models.DailyIdiom{Date: time.Now(), Language: enums.LanguageEn, WordID: translation.OriginalID}
	require.NoError(t, db.DB.Create(&daily).Error)
	require.NoError(t, db.DB.Exec(`INSERT INTO daily_idiom_deliveries (user_id,date,daily_idiom_id,sent_at) VALUES (?,CURRENT_DATE,?,NOW())`, user.ID, daily.ID).Error)
	require.NoError(t, db.DB.Exec(`INSERT INTO word_descriptions (word_id,translation_word_id,model,description) VALUES (?,?,'fixture','a meaning')`, translation.OriginalID, translation.TranslationID).Error)
	snapshots := map[string]string{}
	for _, table := range []string{"words", "translations", "vocabulary", "daily_idioms", "daily_idiom_deliveries", "word_descriptions"} {
		expression := "to_jsonb(t)"
		if table == "words" {
			expression += " - 'type'"
		}
		var before string
		require.NoError(t, db.DB.Raw("SELECT jsonb_agg("+expression+" ORDER BY "+expression+")::text FROM "+table+" t").Scan(&before).Error)
		snapshots[table] = before
	}
	for range 2 {
		runIdiomMigration(t, "0031_clear_manual_idiom_starters")
	}
	var idioms []models.Word
	require.NoError(t, db.DB.Where("type = ?", enums.TypeIdiom).Find(&idioms).Error)
	require.Len(t, idioms, 1)
	assert.Equal(t, unrelated.ID, idioms[0].ID)
	for table, before := range snapshots {
		expression := "to_jsonb(t)"
		if table == "words" {
			expression += " - 'type'"
		}
		var after string
		require.NoError(t, db.DB.Raw("SELECT jsonb_agg("+expression+" ORDER BY "+expression+")::text FROM "+table+" t").Scan(&after).Error)
		assert.Equal(t, before, after, table)
	}
}

func TestRetireStartersPreservesAmbiguousImportedMatches(t *testing.T) {
	for _, tc := range []struct {
		edition, status string
		processed       int64
		preserved       int
	}{
		{"plwiktionary", "succeeded", 3, 3},
		{"frwiktionary", "failed", 500, 3},
		{"trwiktionary", "interrupted", 500, 3},
		{"enwiktionary", "succeeded", 3, 30},
		{"ruwiktionary", "failed", 500, 30},
		{"itwiktionary", "interrupted", 500, 30},
		{"unknown-edition", "succeeded", 3, 30},
		{"dewiktionary", "failed", 0, 0},
		{"enwiktionary-es", "queued", 0, 0},
	} {
		t.Run(tc.edition+tc.status, func(t *testing.T) {
			testkit.Truncate(t)
			runIdiomMigration(t, "0028_seed_daily_idioms_all_languages")
			source := seedDictionary(t, tc.edition, "https://example.invalid")
			// Skipped records can confirm seeds that already existed.
			job := models.DictionaryImportJob{DictionaryID: source.ID, Edition: tc.edition, Status: tc.status, Processed: tc.processed, Skipped: tc.processed, RecordErrors: []string{}}
			require.NoError(t, db.DB.Create(&job).Error)
			runIdiomMigration(t, "0031_clear_manual_idiom_starters")
			var count int64
			require.NoError(t, db.DB.Model(&models.Word{}).Where("type = ?", enums.TypeIdiom).Count(&count).Error)
			assert.EqualValues(t, tc.preserved, count)
		})
	}
}

func TestRemoveDictionaryLanguageColumnsPreservesExistingJobs(t *testing.T) {
	testkit.Truncate(t)
	tx := db.DB.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	require.NoError(t, tx.Exec(`CREATE TEMP TABLE dictionaries (LIKE public.dictionaries INCLUDING ALL) ON COMMIT DROP;
 CREATE TEMP TABLE dictionary_import_jobs (LIKE public.dictionary_import_jobs INCLUDING ALL) ON COMMIT DROP;`).Error)
	migration, err := os.ReadFile("src/data/migrations/0029_add_language_idiom_imports.sql")
	require.NoError(t, err)
	require.NoError(t, tx.Exec(string(migration)).Error)
	require.NoError(t, tx.Exec(`INSERT INTO dictionary_import_jobs (dictionary_id,source_name,edition,download_url,target_language,status,processed,skipped)
 SELECT id,name,edition,download_url,target_language,'queued',7,7 FROM dictionaries`).Error)
	var before string
	require.NoError(t, tx.Raw(`SELECT jsonb_agg(to_jsonb(j) - 'target_language' ORDER BY id)::text FROM dictionary_import_jobs j`).Scan(&before).Error)
	migration, err = os.ReadFile("src/data/migrations/0030_remove_dictionary_target_language.sql")
	require.NoError(t, err)
	for range 2 {
		require.NoError(t, tx.Exec(string(migration)).Error)
	}
	var after string
	require.NoError(t, tx.Raw(`SELECT jsonb_agg(to_jsonb(j) ORDER BY id)::text FROM dictionary_import_jobs j`).Scan(&after).Error)
	assert.Equal(t, before, after)
	var columns int64
	require.NoError(t, tx.Raw(`SELECT count(*) FROM pg_attribute WHERE attrelid IN ('pg_temp.dictionaries'::regclass,'pg_temp.dictionary_import_jobs'::regclass) AND attname='target_language' AND NOT attisdropped`).Scan(&columns).Error)
	assert.Zero(t, columns)
}
