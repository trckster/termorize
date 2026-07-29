package tests

import (
	"fmt"
	"net/http"
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

func collectionPracticeSeedVocabulary(
	t *testing.T,
	userID uint,
	translationID uuid.UUID,
	knowledge int,
) models.Vocabulary {
	t.Helper()

	vocabulary := models.Vocabulary{
		UserID:        userID,
		TranslationID: translationID,
		Progress: models.ProgressEntries{{
			Type:      enums.KnowledgeTypeTranslation,
			Knowledge: knowledge,
		}},
	}
	require.NoError(t, db.DB.Create(&vocabulary).Error)
	return vocabulary
}

func collectionPracticeKnowledge(t *testing.T, vocabularyID uuid.UUID) int {
	t.Helper()

	var vocabulary models.Vocabulary
	require.NoError(t, db.DB.Where("id = ?", vocabularyID).First(&vocabulary).Error)
	for _, entry := range vocabulary.Progress {
		if entry.Type == enums.KnowledgeTypeTranslation {
			return entry.Knowledge
		}
	}

	t.Fatalf("translation knowledge not found for vocabulary %s", vocabularyID)
	return 0
}

func TestStartCollectionPracticeRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodPost, "/api/collections/"+uuid.NewString()+"/practice", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestStartCollectionPracticeRejectsEmptyIntersection(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Travel", uintPtr(user.ID), false, true)
	translationID := collectionSeedTranslation(t, "train", "treno", enums.LanguageEn, enums.LanguageIt)
	collectionLink(t, collection.ID, translationID, 0)

	rec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/practice",
		nil,
	)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), services.ErrNoCollectionPracticeVocabulary.Error())
}

func TestStartCollectionPracticeIncludesMasteredAndReportsCounts(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Travel", uintPtr(user.ID), false, true)
	masteredTranslationID := collectionSeedTranslation(t, "train", "treno", enums.LanguageEn, enums.LanguageIt)
	deletedTranslationID := collectionSeedTranslation(t, "plane", "aereo", enums.LanguageEn, enums.LanguageIt)
	missingTranslationID := collectionSeedTranslation(t, "ship", "nave", enums.LanguageEn, enums.LanguageIt)
	collectionLink(t, collection.ID, masteredTranslationID, 0)
	collectionLink(t, collection.ID, deletedTranslationID, 1)
	collectionLink(t, collection.ID, missingTranslationID, 2)

	mastered := collectionPracticeSeedVocabulary(t, user.ID, masteredTranslationID, 100)
	now := time.Now().UTC()
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).
		Where("id = ?", mastered.ID).
		Updates(map[string]any{"mastered_at": &now}).Error)

	deleted := collectionPracticeSeedVocabulary(t, user.ID, deletedTranslationID, 30)
	require.NoError(t, db.DB.Model(&models.Vocabulary{}).
		Where("id = ?", deleted.ID).
		Update("deleted_at", now).Error)

	startRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/practice",
		nil,
	)
	require.Equal(t, http.StatusOK, startRec.Code)

	var round services.CollectionPracticeRound
	testkit.DecodeJSON(t, startRec, &round)
	assert.Equal(t, collection.ID, round.CollectionID)
	assert.Equal(t, collection.Title, round.CollectionTitle)
	assert.Equal(t, []uuid.UUID{mastered.ID}, round.VocabularyIDs)

	detailRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodGet,
		"/api/collections/"+collection.ID.String(),
		nil,
	)
	require.Equal(t, http.StatusOK, detailRec.Code)
	var detail services.CollectionDetail
	testkit.DecodeJSON(t, detailRec, &detail)
	assert.Equal(t, 1, detail.VocabularyCount)

	listRec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections", nil)
	require.Equal(t, http.StatusOK, listRec.Code)
	var list services.CollectionListResponse
	testkit.DecodeJSON(t, listRec, &list)
	require.Len(t, list.Data, 1)
	assert.Equal(t, 1, list.Data[0].VocabularyCount)
}

func TestCollectionPracticeAnswerKeepsKnowledgeAndAppearsInHistory(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Travel basics", uintPtr(user.ID), false, true)
	targetTranslationID := collectionSeedTranslation(t, "train", "treno", enums.LanguageEn, enums.LanguageIt)
	collectionLink(t, collection.ID, targetTranslationID, 0)
	target := collectionPracticeSeedVocabulary(t, user.ID, targetTranslationID, 45)

	for index, pair := range [][2]string{
		{"plane", "aereo"},
		{"ship", "nave"},
		{"car", "auto"},
	} {
		translationID := collectionSeedTranslation(
			t,
			pair[0],
			pair[1],
			enums.LanguageEn,
			enums.LanguageIt,
		)
		collectionPracticeSeedVocabulary(t, user.ID, translationID, 20+index)
	}

	exerciseRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/practice/exercises",
		map[string]any{
			"target_vocabulary_id": target.ID,
			"matching":             false,
		},
	)
	require.Equal(t, http.StatusOK, exerciseRec.Code)

	var exercise struct {
		ExerciseID uuid.UUID          `json:"exercise_id"`
		Type       enums.ExerciseType `json:"type"`
	}
	testkit.DecodeJSON(t, exerciseRec, &exercise)

	answer := "treno"
	if exercise.Type == enums.ExerciseTypeBasicReversed ||
		exercise.Type == enums.ExerciseTypeChoiceReversed ||
		exercise.Type == enums.ExerciseTypeCharactersReversed {
		answer = "train"
	}

	verifyRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/exercises/"+exercise.ExerciseID.String()+"/verify",
		map[string]any{"answer": answer},
	)
	require.Equal(t, http.StatusOK, verifyRec.Code)

	var verify struct {
		Result        string `json:"result"`
		Knowledge     int    `json:"knowledge"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, verifyRec, &verify)
	assert.Equal(t, "correct", verify.Result)
	assert.Equal(t, 0, verify.ProgressDelta)
	assert.Equal(t, 45, verify.Knowledge)
	assert.Equal(t, 45, collectionPracticeKnowledge(t, target.ID))

	historyRec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises", nil)
	require.Equal(t, http.StatusOK, historyRec.Code)
	var history services.ExerciseListResponse
	testkit.DecodeJSON(t, historyRec, &history)
	require.Len(t, history.Data, 1)
	require.NotNil(t, history.Data[0].CollectionPractice)
	assert.Equal(t, collection.ID, history.Data[0].CollectionPractice.ID)
	assert.Equal(t, "Travel basics", history.Data[0].CollectionPractice.Title)
	require.NotEmpty(t, history.Data[0].Vocabulary)

	var correctLink *services.ExerciseListVocabulary
	for index := range history.Data[0].Vocabulary {
		if history.Data[0].Vocabulary[index].IsCorrect {
			correctLink = &history.Data[0].Vocabulary[index]
			break
		}
	}
	require.NotNil(t, correctLink)
	require.NotNil(t, correctLink.ProgressDelta)
	assert.Equal(t, 0, *correctLink.ProgressDelta)
}

func TestCollectionPracticeWrongAnswerDoesNotSubtractKnowledge(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Difficult words", uintPtr(user.ID), false, true)
	translationID := collectionSeedTranslation(t, "train", "treno", enums.LanguageEn, enums.LanguageIt)
	collectionLink(t, collection.ID, translationID, 0)
	vocabulary := collectionPracticeSeedVocabulary(t, user.ID, translationID, 45)

	exerciseRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/practice/exercises",
		map[string]any{
			"target_vocabulary_id": vocabulary.ID,
			"matching":             false,
		},
	)
	require.Equal(t, http.StatusOK, exerciseRec.Code)

	var exercise struct {
		ExerciseID uuid.UUID `json:"exercise_id"`
	}
	testkit.DecodeJSON(t, exerciseRec, &exercise)

	verifyRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/exercises/"+exercise.ExerciseID.String()+"/verify",
		map[string]any{"answer": "intentionally wrong answer"},
	)
	require.Equal(t, http.StatusOK, verifyRec.Code)

	var verify struct {
		Result        string `json:"result"`
		Knowledge     int    `json:"knowledge"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, verifyRec, &verify)
	assert.Equal(t, "wrong", verify.Result)
	assert.Equal(t, 0, verify.ProgressDelta)
	assert.Equal(t, 45, verify.Knowledge)
	assert.Equal(t, 45, collectionPracticeKnowledge(t, vocabulary.ID))
}

func TestCollectionPracticeMatchingUsesOnlyCollectionWordsAndKeepsKnowledge(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Travel match", uintPtr(user.ID), false, true)
	collectionVocabulary := make([]models.Vocabulary, 0, 5)
	for index, pair := range [][2]string{
		{"train", "treno"},
		{"plane", "aereo"},
		{"ship", "nave"},
		{"car", "auto"},
		{"road", "strada"},
	} {
		translationID := collectionSeedTranslation(
			t,
			pair[0],
			pair[1],
			enums.LanguageEn,
			enums.LanguageIt,
		)
		collectionLink(t, collection.ID, translationID, index)
		collectionVocabulary = append(
			collectionVocabulary,
			collectionPracticeSeedVocabulary(t, user.ID, translationID, 50+index),
		)
	}

	outsideTranslationID := collectionSeedTranslation(t, "house", "casa", enums.LanguageEn, enums.LanguageIt)
	outsideVocabulary := collectionPracticeSeedVocabulary(t, user.ID, outsideTranslationID, 80)

	exerciseRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/practice/exercises",
		map[string]any{
			"target_vocabulary_id": collectionVocabulary[0].ID,
			"matching":             true,
		},
	)
	require.Equal(t, http.StatusOK, exerciseRec.Code)

	var exercise struct {
		ExerciseID uuid.UUID                    `json:"exercise_id"`
		Type       enums.ExerciseType           `json:"type"`
		Cards      []services.ExerciseMatchCard `json:"cards"`
	}
	testkit.DecodeJSON(t, exerciseRec, &exercise)
	require.Equal(t, enums.ExerciseTypeMatchPairs, exercise.Type)
	require.Len(t, exercise.Cards, 10)

	collectionIDs := make(map[uuid.UUID]bool, len(collectionVocabulary))
	for _, vocabulary := range collectionVocabulary {
		collectionIDs[vocabulary.ID] = true
	}

	cardsByVocabulary := make(map[uuid.UUID][]services.ExerciseMatchCard)
	for _, card := range exercise.Cards {
		assert.True(t, collectionIDs[card.VocabularyID])
		assert.NotEqual(t, outsideVocabulary.ID, card.VocabularyID)
		cardsByVocabulary[card.VocabularyID] = append(cardsByVocabulary[card.VocabularyID], card)
	}

	attempts := make([]map[string]string, 0, 5)
	for vocabularyID, cards := range cardsByVocabulary {
		require.Len(t, cards, 2, "vocabulary %s should have two cards", vocabularyID)
		attempts = append(attempts, map[string]string{
			"first_card_id":  cards[0].ID,
			"second_card_id": cards[1].ID,
		})
	}

	completeRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/exercises/"+exercise.ExerciseID.String()+"/match-pairs/complete",
		map[string]any{"attempts": attempts},
	)
	require.Equal(t, http.StatusOK, completeRec.Code, completeRec.Body.String())

	var result services.MatchPairsCompleteResult
	testkit.DecodeJSON(t, completeRec, &result)
	require.Len(t, result.Results, 5)
	expectedKnowledge := make(map[uuid.UUID]int, len(collectionVocabulary))
	for index, vocabulary := range collectionVocabulary {
		expectedKnowledge[vocabulary.ID] = 50 + index
	}
	for _, row := range result.Results {
		require.NotNil(t, row.ProgressDelta)
		assert.Equal(t, 0, *row.ProgressDelta)
		assert.Equal(t, expectedKnowledge[row.ID], collectionPracticeKnowledge(t, row.ID))
	}
	assert.Equal(t, 80, collectionPracticeKnowledge(t, outsideVocabulary.ID))
}

func TestCollectionPracticeChoicePrefersCollectionDistractorsThenFallsBack(t *testing.T) {
	for _, scenario := range []struct {
		name                    string
		collectionDistractors   int
		expectedCollectionLinks int
	}{
		{name: "collection has enough distractors", collectionDistractors: 3, expectedCollectionLinks: 4},
		{name: "collection needs vocabulary fallback", collectionDistractors: 2, expectedCollectionLinks: 3},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			testkit.Truncate(t)

			user := testkit.CreateUser(t)
			collection := collectionSeed(t, "Choice words", uintPtr(user.ID), false, true)
			targetTranslationID := collectionSeedTranslation(t, "train", "treno", enums.LanguageEn, enums.LanguageIt)
			collectionLink(t, collection.ID, targetTranslationID, 0)
			target := collectionPracticeSeedVocabulary(t, user.ID, targetTranslationID, 20)

			collectionIDs := map[uuid.UUID]bool{target.ID: true}
			for index := 0; index < scenario.collectionDistractors; index++ {
				translationID := collectionSeedTranslation(
					t,
					fmt.Sprintf("collection-%d", index),
					fmt.Sprintf("raccolta-%d", index),
					enums.LanguageEn,
					enums.LanguageIt,
				)
				collectionLink(t, collection.ID, translationID, index+1)
				vocabulary := collectionPracticeSeedVocabulary(t, user.ID, translationID, 20)
				collectionIDs[vocabulary.ID] = true
			}

			for index := 0; index < 3; index++ {
				translationID := collectionSeedTranslation(
					t,
					fmt.Sprintf("outside-%d", index),
					fmt.Sprintf("fuori-%d", index),
					enums.LanguageEn,
					enums.LanguageIt,
				)
				collectionPracticeSeedVocabulary(t, user.ID, translationID, 20)
			}

			var choiceExerciseID uuid.UUID
			for attempt := 0; attempt < 40; attempt++ {
				rec := testkit.AuthedRequest(
					t,
					user,
					http.MethodPost,
					"/api/collections/"+collection.ID.String()+"/practice/exercises",
					map[string]any{
						"target_vocabulary_id": target.ID,
						"matching":             false,
					},
				)
				require.Equal(t, http.StatusOK, rec.Code)

				var exercise struct {
					ExerciseID uuid.UUID          `json:"exercise_id"`
					Type       enums.ExerciseType `json:"type"`
				}
				testkit.DecodeJSON(t, rec, &exercise)
				if exercise.Type == enums.ExerciseTypeChoiceDirect ||
					exercise.Type == enums.ExerciseTypeChoiceReversed {
					choiceExerciseID = exercise.ExerciseID
					break
				}
			}
			require.NotEqual(t, uuid.Nil, choiceExerciseID, "expected to generate a choice exercise")

			var linkedIDs []uuid.UUID
			require.NoError(t, db.DB.Table("vocabulary_exercises").
				Where("exercise_id = ?", choiceExerciseID).
				Pluck("vocabulary_id", &linkedIDs).Error)
			require.Len(t, linkedIDs, services.ChoiceExerciseVocabularyCount)

			collectionLinkCount := 0
			for _, vocabularyID := range linkedIDs {
				if collectionIDs[vocabularyID] {
					collectionLinkCount++
				}
			}
			assert.Equal(t, scenario.expectedCollectionLinks, collectionLinkCount)
		})
	}
}

func TestCollectionPracticeRejectsVocabularyOutsideCollection(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Travel", uintPtr(user.ID), false, true)
	collectionTranslationID := collectionSeedTranslation(t, "train", "treno", enums.LanguageEn, enums.LanguageIt)
	collectionLink(t, collection.ID, collectionTranslationID, 0)
	collectionPracticeSeedVocabulary(t, user.ID, collectionTranslationID, 20)

	outsideTranslationID := collectionSeedTranslation(t, "house", "casa", enums.LanguageEn, enums.LanguageIt)
	outside := collectionPracticeSeedVocabulary(t, user.ID, outsideTranslationID, 20)

	rec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/practice/exercises",
		map[string]any{
			"target_vocabulary_id": outside.ID,
			"matching":             false,
		},
	)
	require.Equal(t, http.StatusConflict, rec.Code, fmt.Sprintf("body: %s", rec.Body.String()))
	assert.Contains(t, rec.Body.String(), services.ErrCollectionPracticeVocabularyUnavailable.Error())
}
