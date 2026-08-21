package tests

import (
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

// An inverted schedule interval used to drive rand.Intn negative and crash the
// daily cron for every user processed after the offending one. Generation must
// skip such users instead of panicking.
func TestGenerateExercisesSkipsInvalidScheduleWithoutPanic(t *testing.T) {
	testkit.Truncate(t)

	settings := models.UserSettings{
		TimeZone: "UTC",
		Telegram: models.UserTelegramSettings{
			BotEnabled:            true,
			DailyQuestionsEnabled: true,
			DailyQuestionsCount:   3,
			DailyQuestionsSchedule: []models.UserTelegramQuestionsScheduleItem{
				{From: "22:00", To: "06:00"},
			},
		},
	}
	user := testkit.CreateUser(t, testkit.WithSettings(settings))
	exerciseSeedVocabulary(t, user.ID, "carta", "letter", enums.LanguageIt, enums.LanguageEn)

	target := time.Now().UTC().AddDate(0, 0, 1)
	require.NotPanics(t, func() {
		generated := services.GenerateExercises(user, target)
		assert.Zero(t, generated, "an invalid schedule must not generate exercises")
	})

	var count int64
	require.NoError(t, db.DB.Model(&models.Exercise{}).Where("user_id = ?", user.ID).Count(&count).Error)
	assert.Zero(t, count, "no exercise rows should be created from an invalid schedule")
}
