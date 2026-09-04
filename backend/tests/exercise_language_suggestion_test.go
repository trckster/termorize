package tests

import (
	"sync"
	"sync/atomic"
	"testing"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExerciseLanguageSuggestionsAreCappedPerUserLanguageAndFamily(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)
	exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)

	for range 5 {
		result, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeAudioDirect)
		require.NoError(t, err)
		assert.True(t, result.ShowIgnoreLanguageSuggestion)
	}

	sixth, err := services.CreateRandomExerciseOfTypes(user.ID, enums.ExerciseTypeAudioDirect)
	require.NoError(t, err)
	assert.False(t, sixth.ShowIgnoreLanguageSuggestion)

	descriptionSuggestion, err := services.ReserveExerciseLanguageSuggestion(
		user.ID,
		services.ExerciseLanguageSuggestionFamilyDescription,
		enums.LanguageEn,
	)
	require.NoError(t, err)
	assert.True(t, descriptionSuggestion)

	italianAudioSuggestion, err := services.ReserveExerciseLanguageSuggestion(
		user.ID,
		services.ExerciseLanguageSuggestionFamilyAudio,
		enums.LanguageIt,
	)
	require.NoError(t, err)
	assert.True(t, italianAudioSuggestion)
}

func TestExerciseLanguageSuggestionReservationIsConcurrencySafe(t *testing.T) {
	testkit.Truncate(t)
	user := testkit.CreateUser(t)

	var shown atomic.Int32
	var wg sync.WaitGroup
	errorsByWorker := make(chan error, 20)
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reserved, err := services.ReserveExerciseLanguageSuggestion(
				user.ID,
				services.ExerciseLanguageSuggestionFamilyDescription,
				enums.LanguageEn,
			)
			if err != nil {
				errorsByWorker <- err
				return
			}
			if reserved {
				shown.Add(1)
			}
		}()
	}
	wg.Wait()
	close(errorsByWorker)
	for err := range errorsByWorker {
		require.NoError(t, err)
	}

	assert.EqualValues(t, 5, shown.Load())
	var count models.ExerciseLanguageSuggestionCount
	require.NoError(t, db.DB.Where(
		"user_id = ? AND family = ? AND language = ?",
		user.ID,
		services.ExerciseLanguageSuggestionFamilyDescription,
		enums.LanguageEn,
	).Take(&count).Error)
	assert.EqualValues(t, 5, count.ShownCount)
}
