package tests

import (
	"testing"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWordPronunciationMigrationAndConstraints(t *testing.T) {
	testkit.Truncate(t)

	var migrationCount int64
	require.NoError(t, db.DB.Table("migrations").Where("name = ?", "0012_create_word_pronunciations").Count(&migrationCount).Error)
	assert.EqualValues(t, 1, migrationCount)

	word := models.Word{Word: "ciao", Language: enums.LanguageIt}
	require.NoError(t, db.DB.Create(&word).Error)
	audio := []byte{0x00, 0xff, 0x10, 0x80, 0x01}
	pronunciation := models.WordPronunciation{
		WordID:   word.ID,
		Model:    "provider/model",
		Voice:    "voice",
		Audio:    audio,
		MIMEType: models.PronunciationMIMETypeMP3,
	}
	require.NoError(t, db.DB.Create(&pronunciation).Error)

	var stored models.WordPronunciation
	require.NoError(t, db.DB.Where("id = ?", pronunciation.ID).First(&stored).Error)
	assert.Equal(t, audio, stored.Audio)
	assert.Nil(t, stored.TelegramFileID)
	assert.False(t, stored.CreatedAt.IsZero())

	duplicate := models.WordPronunciation{
		WordID: word.ID, Model: pronunciation.Model, Voice: pronunciation.Voice,
		Audio: []byte("different"), MIMEType: models.PronunciationMIMETypeMP3,
	}
	require.Error(t, db.DB.Create(&duplicate).Error)
}
