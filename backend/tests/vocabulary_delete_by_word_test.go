package tests

import (
	"testing"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedVocabularyForWordDeletion(t *testing.T, userID uint, original, translated string) models.Vocabulary {
	t.Helper()
	originalWord := models.Word{Word: original, Language: enums.LanguageEn}
	require.NoError(t, db.DB.Where("word = ? AND language = ?", original, enums.LanguageEn).FirstOrCreate(&originalWord).Error)
	translatedWord := models.Word{Word: translated, Language: enums.LanguageDe}
	require.NoError(t, db.DB.Where("word = ? AND language = ?", translated, enums.LanguageDe).FirstOrCreate(&translatedWord).Error)
	translation := models.Translation{
		OriginalID:    originalWord.ID,
		TranslationID: translatedWord.ID,
		Source:        enums.TranslationSourceGoogle,
	}
	require.NoError(t, db.DB.Create(&translation).Error)
	vocabulary := models.Vocabulary{UserID: userID, TranslationID: translation.ID}
	require.NoError(t, db.DB.Create(&vocabulary).Error)
	return vocabulary
}

func TestDeleteVocabularyByWordIgnoresDeletedMatches(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	active := seedVocabularyForWordDeletion(t, user.ID, "bank", "Ufer")
	deleted := seedVocabularyForWordDeletion(t, user.ID, "bank", "Bank")
	require.NoError(t, services.DeleteVocabulary(user.ID, deleted.ID))

	result, err := services.DeleteVocabularyByWord(user.ID, "  BANK  ")
	require.NoError(t, err)
	assert.True(t, result.Deleted)
	assert.Equal(t, "bank", result.Original)
	assert.Equal(t, "Ufer", result.Translation)
	assert.NotNil(t, vocabFindByID(t, active.ID).DeletedAt)

	result, err = services.DeleteVocabularyByWord(user.ID, "bank")
	require.NoError(t, err)
	assert.False(t, result.Deleted)
	assert.False(t, result.Ambiguous)
}

func TestDeleteVocabularyByWordDoesNotChooseAmbiguousMatch(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	seedVocabularyForWordDeletion(t, user.ID, "bank", "Ufer")
	seedVocabularyForWordDeletion(t, user.ID, "bank", "Bank")

	result, err := services.DeleteVocabularyByWord(user.ID, "bank")
	require.NoError(t, err)
	assert.False(t, result.Deleted)
	assert.True(t, result.Ambiguous)
	assert.Equal(t, int64(2), vocabCountForUser(t, user.ID))

	result, err = services.DeleteVocabularyByWord(user.ID, " BANK : bank ")
	require.NoError(t, err)
	assert.True(t, result.Deleted)
	assert.Equal(t, "bank", result.Original)
	assert.Equal(t, "Bank", result.Translation)
	assert.Equal(t, int64(1), vocabCountForUser(t, user.ID))
}

func TestDeleteVocabularyByWordMatchesTranslatedSide(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	seedVocabularyForWordDeletion(t, user.ID, "river", "Fluss")

	result, err := services.DeleteVocabularyByWord(user.ID, " flUSS ")
	require.NoError(t, err)
	assert.True(t, result.Deleted)
	assert.Equal(t, "river", result.Original)
	assert.Equal(t, "Fluss", result.Translation)
}
