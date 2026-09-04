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
	testkit.RequireStatus(t, rec, http.StatusUnauthorized)
}

func TestStartCollectionPracticeRejectsInvalidCollectionID(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections/not-a-uuid/practice",
		nil,
	)
	testkit.RequireStatus(t, rec, http.StatusBadRequest)

	var body map[string]string
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "invalid collection ID", body["error"])
}

func TestStartCollectionPracticeReturnsNotFoundForMissingCollection(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	rec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections/"+uuid.NewString()+"/practice",
		nil,
	)
	testkit.RequireStatus(t, rec, http.StatusNotFound)

	var body map[string]string
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "collection not found", body["error"])
}

func TestStartCollectionPracticeHidesInaccessibleCollection(t *testing.T) {
	testkit.Truncate(t)

	owner := testkit.CreateUser(t, testkit.WithName("Owner"))
	viewer := testkit.CreateUser(t, testkit.WithName("Viewer"))
	collection := collectionSeed(t, "Private", uintPtr(owner.ID), false, false)

	rec := testkit.AuthedRequest(
		t,
		viewer,
		http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/practice",
		nil,
	)
	testkit.RequireStatus(t, rec, http.StatusNotFound)

	var body map[string]string
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "collection not found", body["error"])
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
	testkit.RequireStatus(t, rec, http.StatusUnprocessableEntity)
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
	testkit.RequireStatus(t, startRec, http.StatusOK)

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
	testkit.RequireStatus(t, detailRec, http.StatusOK)
	var detail services.CollectionDetail
	testkit.DecodeJSON(t, detailRec, &detail)
	assert.Equal(t, 1, detail.VocabularyCount)

	listRec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/collections", nil)
	testkit.RequireStatus(t, listRec, http.StatusOK)
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
	testkit.RequireStatus(t, exerciseRec, http.StatusOK)

	var exercise struct {
		ExerciseID uuid.UUID          `json:"exercise_id"`
		Type       enums.ExerciseType `json:"type"`
	}
	testkit.DecodeJSON(t, exerciseRec, &exercise)

	answer := "treno"
	if exercise.Type == enums.ExerciseTypeBasicReversed ||
		exercise.Type == enums.ExerciseTypeChoiceReversed ||
		exercise.Type == enums.ExerciseTypeCharactersReversed ||
		exercise.Type == enums.ExerciseTypeAudioReversed ||
		exercise.Type == enums.ExerciseTypeDescriptionReversed {
		answer = "train"
	}

	verifyRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/exercises/"+exercise.ExerciseID.String()+"/verify",
		map[string]any{"answer": answer},
	)
	testkit.RequireStatus(t, verifyRec, http.StatusOK)

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
	testkit.RequireStatus(t, historyRec, http.StatusOK)
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
	testkit.RequireStatus(t, exerciseRec, http.StatusOK)

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
	testkit.RequireStatus(t, verifyRec, http.StatusOK)

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

func TestCollectionPracticeCanExcludeAudioForImmediateReplacement(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)
	collection := collectionSeed(t, "Listening replacement", uintPtr(user.ID), false, true)
	translationID := collectionSeedTranslation(t, "train", "treno", enums.LanguageEn, enums.LanguageIt)
	collectionLink(t, collection.ID, translationID, 0)
	vocabulary := collectionPracticeSeedVocabulary(t, user.ID, translationID, 45)

	for range 20 {
		rec := testkit.AuthedRequest(
			t,
			user,
			http.MethodPost,
			"/api/collections/"+collection.ID.String()+"/practice/exercises",
			map[string]any{
				"target_vocabulary_id": vocabulary.ID,
				"matching":             false,
				"exclude_audio":        true,
			},
		)
		testkit.RequireStatus(t, rec, http.StatusOK)
		var exercise struct {
			Type        enums.ExerciseType `json:"type"`
			AudioWordID *uuid.UUID         `json:"audio_word_id"`
		}
		testkit.DecodeJSON(t, rec, &exercise)
		assert.NotEqual(t, enums.ExerciseTypeAudioDirect, exercise.Type)
		assert.NotEqual(t, enums.ExerciseTypeAudioReversed, exercise.Type)
		assert.Nil(t, exercise.AudioWordID)
	}
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
	testkit.RequireStatus(t, exerciseRec, http.StatusOK)

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
	testkit.RequireStatus(t, completeRec, http.StatusOK)

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
				testkit.RequireStatus(t, rec, http.StatusOK)

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
	testkit.RequireStatus(t, rec, http.StatusConflict)
	assert.Contains(t, rec.Body.String(), services.ErrCollectionPracticeVocabularyUnavailable.Error())
}

func TestCollectionPracticeUserPathCompletesThroughPublicEndpoints(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t)

	createRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections",
		map[string]any{"title": "Travel essentials"},
	)
	testkit.RequireStatus(t, createRec, http.StatusCreated)
	var collection services.CollectionDetail
	testkit.DecodeJSON(t, createRec, &collection)
	require.NotEqual(t, uuid.Nil, collection.ID)

	pairs := [][2]string{
		{"train", "treno"},
		{"plane", "aereo"},
		{"ship", "nave"},
		{"car", "auto"},
	}
	for _, pair := range pairs {
		addRec := testkit.AuthedRequest(
			t,
			user,
			http.MethodPost,
			"/api/collections/"+collection.ID.String()+"/translations",
			map[string]any{
				"original":             pair[0],
				"translation":          pair[1],
				"original_language":    "en",
				"translation_language": "it",
			},
		)
		testkit.RequireStatus(t, addRec, http.StatusCreated)
	}

	addVocabularyRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/add-to-vocabulary",
		map[string]any{},
	)
	testkit.RequireStatus(t, addVocabularyRec, http.StatusOK)
	var added services.AddCollectionToVocabularyResult
	testkit.DecodeJSON(t, addVocabularyRec, &added)
	assert.Equal(t, len(pairs), added.Total)
	assert.Equal(t, len(pairs), added.Added)
	assert.Equal(t, 0, added.Skipped)

	startRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/practice",
		nil,
	)
	testkit.RequireStatus(t, startRec, http.StatusOK)
	var round services.CollectionPracticeRound
	testkit.DecodeJSON(t, startRec, &round)
	assert.Equal(t, collection.ID, round.CollectionID)
	assert.Equal(t, "Travel essentials", round.CollectionTitle)
	require.Len(t, round.VocabularyIDs, len(pairs))

	targetID := round.VocabularyIDs[0]
	exerciseRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/collections/"+collection.ID.String()+"/practice/exercises",
		map[string]any{
			"target_vocabulary_id": targetID,
			"matching":             false,
		},
	)
	testkit.RequireStatus(t, exerciseRec, http.StatusOK)
	var exercise struct {
		ExerciseID     uuid.UUID          `json:"exercise_id"`
		Type           enums.ExerciseType `json:"type"`
		QuestionWord   string             `json:"question_word"`
		AnswerLanguage enums.Language     `json:"answer_language"`
	}
	testkit.DecodeJSON(t, exerciseRec, &exercise)
	require.NotEqual(t, uuid.Nil, exercise.ExerciseID)
	assert.NotEmpty(t, exercise.QuestionWord)

	var target models.Vocabulary
	require.NoError(t, db.DB.
		Preload("Translation.Original").
		Preload("Translation.Translation").
		Where("id = ? AND user_id = ?", targetID, user.ID).
		First(&target).Error)
	require.NotNil(t, target.Translation)
	require.NotNil(t, target.Translation.Original)
	require.NotNil(t, target.Translation.Translation)

	answer := target.Translation.Translation.Word
	if exercise.AnswerLanguage == target.Translation.Original.Language {
		answer = target.Translation.Original.Word
	}

	verifyRec := testkit.AuthedRequest(
		t,
		user,
		http.MethodPost,
		"/api/exercises/"+exercise.ExerciseID.String()+"/verify",
		map[string]any{"answer": answer},
	)
	testkit.RequireStatus(t, verifyRec, http.StatusOK)
	var verified struct {
		Result        string `json:"result"`
		Knowledge     int    `json:"knowledge"`
		ProgressDelta int    `json:"progress_delta"`
	}
	testkit.DecodeJSON(t, verifyRec, &verified)
	assert.Equal(t, services.ExerciseVocabularyResultCorrect, verified.Result)
	assert.Equal(t, 0, verified.Knowledge)
	assert.Equal(t, 0, verified.ProgressDelta)

	var storedExercise models.Exercise
	require.NoError(t, db.DB.
		Where("id = ? AND user_id = ?", exercise.ExerciseID, user.ID).
		First(&storedExercise).Error)
	assert.Equal(t, enums.ExerciseStatusCompleted, storedExercise.Status)
	require.NotNil(t, storedExercise.PracticeCollectionID)
	assert.Equal(t, collection.ID, *storedExercise.PracticeCollectionID)

	historyRec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/exercises", nil)
	testkit.RequireStatus(t, historyRec, http.StatusOK)
	var history services.ExerciseListResponse
	testkit.DecodeJSON(t, historyRec, &history)
	require.Len(t, history.Data, 1)
	require.NotNil(t, history.Data[0].CollectionPractice)
	assert.Equal(t, collection.ID, history.Data[0].CollectionPractice.ID)
	assert.Equal(t, "Travel essentials", history.Data[0].CollectionPractice.Title)
}
