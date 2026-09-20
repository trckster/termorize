package tests

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A failed translation must preserve the answer, and a subsequent Telegram
// delivery must retry only translation without applying the score twice.
func TestDescriptionFeedbackRetriesTranslationOnTelegramReplay(t *testing.T) {
	for _, firstFailure := range []string{"error", "blank"} {
		t.Run(firstFailure, func(t *testing.T) {
			testkit.Truncate(t)
			tg := testkit.MockTelegramAPI(t)
			const telegramID, messageID int64 = 555201, 603
			user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID), testkit.WithSettings(models.UserSettings{SystemLanguage: enums.LanguageEn}))
			vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
			exercise := exerciseSeedExercise(t, user.ID, enums.ExerciseTypeDescriptionDirect, enums.ExerciseStatusInProgress, vocabulary.ID)
			require.NoError(t, db.DB.Model(&exercise).Updates(map[string]any{"telegram_message_id": messageID, "description": "Original clue."}).Error)
			calls := 0
			testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
				TranslateFunc: func(text, source, target string) (string, error) {
					calls++
					assert.Equal(t, "Original clue.", text)
					if calls == 1 {
						if firstFailure == "error" {
							return "", errors.New("unavailable")
						}
						return " \n ", nil
					}
					return "Traduzione.", nil
				},
			})
			update := exerciseResultCallback(telegramID, messageID, "retry-translation", "idk:"+exercise.ID.String())
			testkit.RequireStatus(t, telegramUpdate(t, update), http.StatusOK)
			firstLink := exerciseLink(t, exercise.ID, vocabulary.ID)
			require.NotNil(t, firstLink.ProgressDelta)
			assert.Empty(t, exerciseReload(t, exercise.ID).DescriptionTranslation)
			testkit.RequireStatus(t, telegramUpdate(t, update), http.StatusOK)
			assert.Equal(t, 2, calls)
			assert.Equal(t, "Traduzione.", exerciseReload(t, exercise.ID).DescriptionTranslation)
			assert.Equal(t, firstLink.ProgressDelta, exerciseLink(t, exercise.ID, vocabulary.ID).ProgressDelta)
			assert.Equal(t, firstLink.KnowledgeAfter, exerciseLink(t, exercise.ID, vocabulary.ID).KnowledgeAfter)
			require.Len(t, tg.RequestsFor("editMessageText"), 2)
			for i, request := range tg.RequestsFor("editMessageText") {
				var sent struct {
					Text string `json:"text"`
				}
				require.NoError(t, json.Unmarshal(request.Body, &sent))
				assert.Contains(t, sent.Text, "paper")
				assert.Contains(t, sent.Text, "carta")
				assert.Contains(t, sent.Text, "Original description:\n🇬🇧 Original clue.")
				if i == 0 {
					assert.NotContains(t, sent.Text, "Description translation:")
				} else {
					assert.Contains(t, sent.Text, "Description translation:")
					assert.Contains(t, sent.Text, "Traduzione.")
				}
			}
		})
	}
}

func TestDescriptionFeedbackLegacySnapshotFallback(t *testing.T) {
	for _, exerciseType := range []enums.ExerciseType{enums.ExerciseTypeDescriptionDirect, enums.ExerciseTypeDescriptionReversed} {
		t.Run(string(exerciseType), func(t *testing.T) {
			testkit.Truncate(t)
			user := testkit.CreateUser(t)
			exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
			source := "en"
			if exerciseType == enums.ExerciseTypeDescriptionReversed {
				source = "it"
			}
			testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
				DetectFunc: func(string) (string, error) { return source, nil },
			})
			exercise, err := services.CreateRandomExerciseOfTypes(user.ID, exerciseType)
			require.NoError(t, err)
			require.NoError(t, db.DB.Model(&models.Exercise{}).Where("id = ?", exercise.ExerciseID).Update("description", "").Error)
			calls := 0
			testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
				TranslateFunc: func(text, source, target string) (string, error) {
					calls++
					assert.Equal(t, exercise.Description, text)
					return "Legacy translation.", nil
				},
			})
			result, err := services.VerifyExerciseAnswer(exercise.ExerciseID, user.ID, "unrelated")
			require.NoError(t, err)
			require.NotNil(t, result.DescriptionFeedback)
			assert.Equal(t, exercise.Description, result.DescriptionFeedback.Original)
			assert.Equal(t, "Legacy translation.", result.DescriptionFeedback.Translation)
			assert.Equal(t, 1, calls)
			assert.Equal(t, exercise.Description, exerciseReload(t, exercise.ExerciseID).Description)
		})
	}
}

func TestDescriptionFeedbackOwnershipAndStateGuards(t *testing.T) {
	for _, scenario := range []string{"foreign", "in progress", "completed", "basic", "deleted vocabulary", "missing clue"} {
		t.Run(scenario, func(t *testing.T) {
			testkit.Truncate(t)
			owner := testkit.CreateUser(t)
			other := testkit.CreateUser(t)
			vocabulary := exerciseSeedVocabulary(t, owner.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
			exerciseType, status := enums.ExerciseTypeDescriptionDirect, enums.ExerciseStatusFailed
			if scenario == "basic" {
				exerciseType = enums.ExerciseTypeBasicDirect
			}
			if scenario == "in progress" {
				status = enums.ExerciseStatusInProgress
			}
			if scenario == "completed" {
				status = enums.ExerciseStatusCompleted
			}
			exercise := exerciseSeedExercise(t, owner.ID, exerciseType, status, vocabulary.ID)
			if scenario != "missing clue" {
				require.NoError(t, services.SaveExerciseDescription(exercise.ID, "Private clue."))
			}
			if scenario == "deleted vocabulary" {
				require.NoError(t, db.DB.Delete(&vocabulary).Error)
			}
			calls := 0
			testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
				TranslateFunc: func(string, string, string) (string, error) {
					calls++
					return "Must not translate.", nil
				},
			})
			userID := owner.ID
			if scenario == "foreign" {
				userID = other.ID
				rec := testkit.AuthedRequest(t, other, http.MethodPost, "/api/exercises/"+exercise.ID.String()+"/verify", map[string]any{"answer": "wrong"})
				testkit.RequireStatus(t, rec, http.StatusNotFound)
				assert.NotContains(t, rec.Body.String(), "Private clue")
			}
			result := &services.VerifyAnswerResult{Result: services.ExerciseVocabularyResultWrong}
			services.AddDescriptionFeedback(exercise.ID, userID, result)
			assert.Zero(t, calls)
			if scenario == "missing clue" {
				require.NotNil(t, result.DescriptionFeedback)
				assert.Equal(t, "carta", result.DescriptionFeedback.AnswerTranslation)
				assert.Empty(t, result.DescriptionFeedback.Translation)
			} else {
				assert.Nil(t, result.DescriptionFeedback)
			}
		})
	}
}
