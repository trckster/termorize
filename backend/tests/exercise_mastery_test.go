package tests

import (
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
)

func TestPendingExercisesRecheckMasteryBeforeDelivery(t *testing.T) {
	for _, exerciseType := range []enums.ExerciseType{
		enums.ExerciseTypeBasicDirect, enums.ExerciseTypeBasicReversed,
		enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed,
		enums.ExerciseTypeCharactersDirect, enums.ExerciseTypeCharactersReversed,
		enums.ExerciseTypeAudioDirect, enums.ExerciseTypeAudioReversed,
		enums.ExerciseTypeDescriptionDirect, enums.ExerciseTypeDescriptionReversed,
		enums.ExerciseTypeMatchPairs,
	} {
		t.Run(string(exerciseType), func(t *testing.T) {
			testkit.Truncate(t)
			user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
				Telegram: models.UserTelegramSettings{BotEnabled: true},
			}))
			vocabulary := exerciseSeedVocabulary(t, user.ID, "deduce", "делать вывод", enums.LanguageEn, enums.LanguageRu)
			exerciseSetTranslationKnowledge(t, vocabulary.ID, 85)
			learning := exerciseSeedVocabulary(t, user.ID, "infer", "заключать", enums.LanguageEn, enums.LanguageRu)
			exerciseSetTranslationKnowledge(t, learning.ID, 99)
			now := time.Now().UTC()
			pending := exerciseSeedExercise(t, user.ID, exerciseType, enums.ExerciseStatusPending, vocabulary.ID)
			if exerciseType == enums.ExerciseTypeMatchPairs {
				for i := 1; i < services.MatchPairsVocabularyCount; i++ {
					word := uuid.NewString()
					other := exerciseSeedVocabulary(t, user.ID, word, word+"-ru", enums.LanguageEn, enums.LanguageRu)
					require.NoError(t, db.DB.Create(&models.ExerciseVocabulary{
						ExerciseID: pending.ID, VocabularyID: other.ID, IsCorrect: true, Position: i,
					}).Error)
				}
			}
			require.NoError(t, db.DB.Model(&pending).Update("scheduled_for", now).Error)
			active := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusInProgress, vocabulary.ID)

			// Another answer masters the word after tomorrow's exercise was queued.
			completed, knowledge, err := services.CompleteExercise(active.ID)
			require.NoError(t, err)
			require.True(t, completed)
			require.Equal(t, 100, knowledge)
			repetitionID, err := services.CreatePendingKnownVocabularyRepetition(user.ID, now)
			require.NoError(t, err)
			// A mastered choice distractor must not invalidate a learning target.
			choiceIDs := []uuid.UUID{learning.ID, vocabulary.ID}
			for len(choiceIDs) < services.ChoiceExerciseVocabularyCount {
				word := uuid.NewString()
				distractor := exerciseSeedVocabulary(t, user.ID, word, word+"-ru", enums.LanguageEn, enums.LanguageRu)
				choiceIDs = append(choiceIDs, distractor.ID)
			}
			learningExercise := exerciseSeedChoiceExercise(t, user.ID, enums.ExerciseTypeChoiceDirect, enums.ExerciseStatusPending, choiceIDs)
			require.NoError(t, db.DB.Model(&learningExercise).Update("scheduled_for", now).Error)
			future := exerciseSeedExercise(t, user.ID, exerciseType, enums.ExerciseStatusPending, vocabulary.ID)
			require.NoError(t, db.DB.Model(&future).Update("scheduled_for", now.Add(time.Hour)).Error)

			due, err := services.GetDuePendingExercises(now)
			require.NoError(t, err)
			ids := make([]uuid.UUID, 0, len(due))
			for _, exercise := range due {
				ids = append(ids, exercise.ExerciseID)
			}
			assert.ElementsMatch(t, []uuid.UUID{repetitionID, learningExercise.ID}, ids)
			matches, err := services.GetDuePendingMatchExercises(now)
			require.NoError(t, err)
			assert.Empty(t, matches)

			require.NoError(t, services.IgnoreDuePendingExercisesWithoutActiveVocabulary(now))
			require.NoError(t, services.IgnoreDuePendingExercisesWithoutActiveVocabulary(now))
			stored := exerciseReload(t, pending.ID)
			assert.Equal(t, enums.ExerciseStatusIgnored, stored.Status)
			assert.NotNil(t, stored.FinishedAt)
			var links []models.ExerciseVocabulary
			require.NoError(t, db.DB.Where("exercise_id = ?", pending.ID).Find(&links).Error)
			for _, link := range links {
				require.NotNil(t, link.Result)
				assert.Equal(t, services.ExerciseVocabularyResultIgnored, *link.Result)
				require.NotNil(t, link.ResultReason)
				assert.Equal(t, services.ExerciseVocabularyResultReasonMasteredVocabulary, *link.ResultReason)
				assert.NotNil(t, link.AnsweredAt)
				assert.Nil(t, link.ProgressDelta)
			}
			for _, id := range []uuid.UUID{repetitionID, learningExercise.ID, future.ID} {
				assert.Equal(t, enums.ExerciseStatusPending, exerciseReload(t, id).Status)
			}
			assert.Equal(t, 100, exerciseTranslationKnowledge(t, exerciseReloadVocabulary(t, vocabulary.ID).Progress))
			assert.Equal(t, 99, exerciseTranslationKnowledge(t, exerciseReloadVocabulary(t, learning.ID).Progress))
		})
	}
}

func TestPendingExerciseMasteryWithoutMasteredAt(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t, testkit.WithSettings(models.UserSettings{
		Telegram: models.UserTelegramSettings{BotEnabled: true},
	}))
	vocabulary := exerciseSeedVocabulary(t, user.ID, "deduce", "делать вывод", enums.LanguageEn, enums.LanguageRu)
	exerciseSetTranslationKnowledge(t, vocabulary.ID, 100)
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).
		Where("id = ?", vocabulary.ID).Update("mastered_at", nil).Error)

	now := time.Now().UTC()
	pending := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicDirect, enums.ExerciseStatusPending, vocabulary.ID)
	require.NoError(t, db.DB.Model(&pending).Update("scheduled_for", now).Error)
	inProgress := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeBasicReversed, enums.ExerciseStatusInProgress, vocabulary.ID)

	due, err := services.GetDuePendingExercises(now)
	require.NoError(t, err)
	assert.Empty(t, due)
	require.NoError(t, services.IgnoreDuePendingExercisesWithoutActiveVocabulary(now))
	assert.Equal(t, enums.ExerciseStatusIgnored, exerciseReload(t, pending.ID).Status)
	assert.Equal(t, enums.ExerciseStatusInProgress, exerciseReload(t, inProgress.ID).Status)
	assert.Nil(t, exerciseLink(t, inProgress.ID, vocabulary.ID).Result)
	assert.Equal(t, 100, exerciseTranslationKnowledge(t, exerciseReloadVocabulary(t, vocabulary.ID).Progress))
}
