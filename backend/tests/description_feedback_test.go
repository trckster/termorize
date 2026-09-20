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

// Contract: both description directions reveal the original clue's translation
// and the word translation after a wrong/skipped answer, never before or for
// correct/almost answers. Translation failure must preserve scoring and answers.
func TestDescriptionFeedbackVerify(t *testing.T) {
	for _, exerciseType := range []enums.ExerciseType{enums.ExerciseTypeDescriptionDirect, enums.ExerciseTypeDescriptionReversed} {
		for _, verdict := range []string{"wrong", "skipped", "correct", "almost", "translation failure"} {
			t.Run(string(exerciseType)+"/"+verdict, func(t *testing.T) {
				testkit.Truncate(t)
				user := testkit.CreateUser(t)
				exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
				source, target, expected, wordTranslation := "en", "it", "paper", "carta"
				if exerciseType == enums.ExerciseTypeDescriptionReversed {
					source, target, expected, wordTranslation = "it", "en", "carta", "paper"
				}
				calls := 0
				var originalClue string
				testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
					DetectFunc: func(string) (string, error) { return source, nil },
					TranslateFunc: func(text, from, to string) (string, error) {
						calls++
						assert.Equal(t, originalClue, text)
						assert.Equal(t, source, from)
						assert.Equal(t, target, to)
						if verdict == "translation failure" {
							return "", errors.New("translation unavailable")
						}
						return "Translated clue.", nil
					},
				})
				exercise, err := services.CreateRandomExerciseOfTypes(user.ID, exerciseType)
				require.NoError(t, err)
				originalClue = exercise.Description
				assert.Equal(t, originalClue, exerciseReload(t, exercise.ExerciseID).Description)
				assert.Zero(t, calls, "translations must not be requested before an answer")
				require.NoError(t, db.DB.Model(&models.WordDescription{}).Where("1 = 1").Update("description", "A replacement clue.").Error)
				answer, expectedResult := "unrelated", "wrong"
				switch verdict {
				case "skipped":
					answer = "termorize skipped answer intentionally incorrect"
				case "correct", "almost":
					answer, expectedResult = expected, verdict
					if verdict == "almost" {
						answer = expected[:len(expected)-1]
					}
				}
				rec := testkit.AuthedRequest(t, user, http.MethodPost, "/api/exercises/"+exercise.ExerciseID.String()+"/verify", map[string]any{"answer": answer})
				testkit.RequireStatus(t, rec, http.StatusOK)
				var response struct {
					Result        string                        `json:"result"`
					CorrectAnswer string                        `json:"correct_answer"`
					Feedback      *services.DescriptionFeedback `json:"description_feedback"`
				}
				testkit.DecodeJSON(t, rec, &response)
				assert.Equal(t, expectedResult, response.Result)
				assert.Equal(t, expected, response.CorrectAnswer)
				if expectedResult != "wrong" {
					assert.Nil(t, response.Feedback)
					assert.Zero(t, calls)
					return
				}
				require.NotNil(t, response.Feedback)
				assert.Equal(t, originalClue, response.Feedback.Original)
				assert.Equal(t, enums.Language(target), response.Feedback.Language)
				assert.Equal(t, wordTranslation, response.Feedback.AnswerTranslation)
				assert.Equal(t, 1, calls)
				stored := exerciseReload(t, exercise.ExerciseID)
				assert.Equal(t, enums.ExerciseStatusFailed, stored.Status)
				if verdict == "translation failure" {
					assert.Empty(t, response.Feedback.Translation)
					assert.Empty(t, stored.DescriptionTranslation)
				} else {
					assert.Equal(t, "Translated clue.", response.Feedback.Translation)
					assert.Equal(t, response.Feedback.Translation, stored.DescriptionTranslation)
				}
			})
		}
	}
}

func TestTelegramDescriptionFeedbackAndReplay(t *testing.T) {
	for _, exerciseType := range []enums.ExerciseType{enums.ExerciseTypeDescriptionDirect, enums.ExerciseTypeDescriptionReversed} {
		for _, interaction := range []string{"wrong", "idk"} {
			t.Run(string(exerciseType)+"/"+interaction, func(t *testing.T) {
				testkit.Truncate(t)
				tg := testkit.MockTelegramAPI(t)
				const telegramID, messageID int64 = 555101, 503
				user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID))
				vocabulary := exerciseSeedVocabulary(t, user.ID, "paper", "carta", enums.LanguageEn, enums.LanguageIt)
				exercise := exerciseSeedExercise(t, user.ID, exerciseType, enums.ExerciseStatusInProgress, vocabulary.ID)
				require.NoError(t, db.DB.Model(&exercise).Update("telegram_message_id", messageID).Error)
				require.NoError(t, services.SaveExerciseDescription(exercise.ID, "Original *clue* [daily]."))
				calls := 0
				testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
					TranslateFunc: func(text, source, target string) (string, error) {
						calls++
						assert.Equal(t, "Original *clue* [daily].", text)
						if exerciseType == enums.ExerciseTypeDescriptionDirect {
							assert.Equal(t, "en", source)
							assert.Equal(t, "it", target)
						} else {
							assert.Equal(t, "it", source)
							assert.Equal(t, "en", target)
						}
						return "Translated *clue* [daily].", nil
					},
				})
				update := exerciseResultCallback(telegramID, messageID, "description-idk", "idk:"+exercise.ID.String())
				method := "editMessageText"
				if interaction == "wrong" {
					update = telegramPrivateMessage(telegramID, "unrelated")
					update["message"].(map[string]any)["reply_to_message"] = map[string]any{"message_id": messageID}
					method = "sendMessage"
				}
				for attempt := 0; attempt < 2; attempt++ {
					testkit.RequireStatus(t, telegramUpdate(t, update), http.StatusOK)
					require.Len(t, tg.RequestsFor(method), attempt+1)
					var sent struct {
						Text string `json:"text"`
					}
					require.NoError(t, json.Unmarshal(tg.RequestsFor(method)[attempt].Body, &sent))
					assert.Contains(t, sent.Text, "paper")
					assert.Contains(t, sent.Text, "carta")
					originalFlag := enums.LanguageEn.Flag()
					if exerciseType == enums.ExerciseTypeDescriptionReversed {
						originalFlag = enums.LanguageIt.Flag()
					}
					assert.Contains(t, sent.Text, originalFlag+` Original \*clue\* \[daily].`)
					assert.Contains(t, sent.Text, `Translated \*clue\* \[daily].`)
				}
				assert.Equal(t, 1, calls, "replay reuses the saved translation")
				assert.JSONEq(t, string(tg.RequestsFor(method)[0].Body), string(tg.RequestsFor(method)[1].Body))
				assert.Equal(t, enums.ExerciseStatusFailed, exerciseReload(t, exercise.ID).Status)
				link := exerciseLink(t, exercise.ID, vocabulary.ID)
				require.NotNil(t, link.ProgressDelta)
				assert.Equal(t, services.ExerciseBasicWrongProgressDelta, *link.ProgressDelta)
				if interaction == "idk" {
					assert.Len(t, tg.RequestsFor("answerCallbackQuery"), 2)
					assert.Empty(t, tg.RequestsFor("sendMessage"))
				} else {
					assert.Empty(t, tg.RequestsFor("editMessageText"))
				}
			})
		}
	}
}
