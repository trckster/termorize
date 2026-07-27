package services

import (
	"encoding/json"
	"errors"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MatchPairAttempt struct {
	FirstCardID  string
	SecondCardID string
}

type MatchPairsCompleteResult struct {
	Status  enums.ExerciseStatus     `json:"status"`
	Results []ExerciseListVocabulary `json:"results"`
}

type matchStateJSON struct {
	Order    []int    `json:"order"`
	Pending  int      `json:"pending"`
	Attempts [][2]int `json:"attempts"`
}

type MatchBoardState struct {
	Order        []int                // display permutation of canonical card indices
	Cards        []ExerciseMatchCard  // canonical order; index == canonical card index
	Pending      int                  // canonical index of held first pick, or -1
	Resolved     map[uuid.UUID]string // vocabulary -> result (correct/almost/wrong)
	CardWrong    map[string]int       // card.ID -> wrong attempts
	MatchedCount int                  // number of resolved vocabularies
}

func replayMatchAttempts(
	cardByID map[string]ExerciseMatchCard,
	expectedVocabularyIDs []uuid.UUID,
	attempts []MatchPairAttempt,
) (states map[uuid.UUID]string, cardWrong map[string]int, err error) {
	states = make(map[uuid.UUID]string, len(expectedVocabularyIDs))
	cardWrong = make(map[string]int, len(expectedVocabularyIDs)*2)
	for _, vocabularyID := range expectedVocabularyIDs {
		states[vocabularyID] = ""
	}

	for _, attempt := range attempts {
		if attempt.FirstCardID == attempt.SecondCardID {
			return nil, nil, ErrInvalidMatchPairResults
		}

		firstCard, ok := cardByID[attempt.FirstCardID]
		if !ok {
			return nil, nil, ErrInvalidMatchPairResults
		}
		secondCard, ok := cardByID[attempt.SecondCardID]
		if !ok {
			return nil, nil, ErrInvalidMatchPairResults
		}

		if states[firstCard.VocabularyID] != "" || states[secondCard.VocabularyID] != "" {
			return nil, nil, ErrInvalidMatchPairResults
		}

		isCorrectPair := firstCard.VocabularyID == secondCard.VocabularyID && firstCard.Side != secondCard.Side
		if isCorrectPair {
			result := ExerciseVocabularyResultCorrect
			if cardWrong[firstCard.ID] > 0 || cardWrong[secondCard.ID] > 0 {
				result = ExerciseVocabularyResultAlmost
			}
			states[firstCard.VocabularyID] = result
			continue
		}

		for _, card := range []ExerciseMatchCard{firstCard, secondCard} {
			cardWrong[card.ID]++
			if cardWrong[card.ID] >= 2 {
				states[card.VocabularyID] = ExerciseVocabularyResultWrong
			}
		}
	}

	return states, cardWrong, nil
}

func CompleteMatchPairsExercise(exerciseID uuid.UUID, userID uint, attempts []MatchPairAttempt) (*MatchPairsCompleteResult, error) {
	if len(attempts) == 0 {
		return nil, ErrInvalidMatchPairResults
	}

	var exercise models.Exercise
	if err := db.DB.
		Where("id = ? AND user_id = ?", exerciseID, userID).
		Take(&exercise).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExerciseNotFound
		}

		return nil, err
	}

	if exercise.Type != enums.ExerciseTypeMatchPairs {
		return nil, ErrInvalidMatchPairResults
	}

	if exercise.Status != enums.ExerciseStatusInProgress {
		return nil, ErrExerciseNotInProgress
	}

	rows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, true)
	if err != nil {
		return nil, err
	}
	if len(rows) != matchPairsVocabularyCount {
		_ = MarkExerciseVocabularyResultWithoutProgress(exercise.ID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
		_ = IgnoreExercise(exercise.ID)
		return nil, ErrExerciseVocabularyDeleted
	}

	expected := make(map[uuid.UUID]exerciseVocabularyDetails, len(rows))
	for _, row := range rows {
		expected[row.VocabularyID] = row
	}

	cardByID := make(map[string]ExerciseMatchCard, len(rows)*2)
	for _, row := range rows {
		originalCard := ExerciseMatchCard{
			ID:           row.VocabularyID.String() + ":" + matchPairCardSideOriginal,
			VocabularyID: row.VocabularyID,
			Word:         row.OriginalWord,
			Language:     row.OriginalLanguage,
			Side:         matchPairCardSideOriginal,
		}
		translationCard := ExerciseMatchCard{
			ID:           row.VocabularyID.String() + ":" + matchPairCardSideTranslation,
			VocabularyID: row.VocabularyID,
			Word:         row.TranslationWord,
			Language:     row.TranslationLanguage,
			Side:         matchPairCardSideTranslation,
		}
		cardByID[originalCard.ID] = originalCard
		cardByID[translationCard.ID] = translationCard
	}

	expectedVocabularyIDs := make([]uuid.UUID, 0, len(expected))
	for vocabularyID := range expected {
		expectedVocabularyIDs = append(expectedVocabularyIDs, vocabularyID)
	}

	states, _, err := replayMatchAttempts(cardByID, expectedVocabularyIDs, attempts)
	if err != nil {
		return nil, err
	}

	submitted := make(map[uuid.UUID]string, len(expected))
	hasWrong := false
	for vocabularyID, result := range states {
		if result == "" {
			return nil, ErrInvalidMatchPairResults
		}

		switch result {
		case ExerciseVocabularyResultCorrect, ExerciseVocabularyResultAlmost:
		case ExerciseVocabularyResultWrong:
			hasWrong = true
		default:
			return nil, ErrInvalidMatchPairResults
		}

		submitted[vocabularyID] = result
	}

	status := enums.ExerciseStatusCompleted
	if hasWrong {
		status = enums.ExerciseStatusFailed
	}

	var completedRows []ExerciseListVocabulary
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		dbResult := tx.Model(&models.Exercise{}).
			Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
			Updates(map[string]any{
				"status":      status,
				"finished_at": time.Now().UTC(),
			})
		if dbResult.Error != nil {
			return dbResult.Error
		}
		if dbResult.RowsAffected == 0 {
			return ErrExerciseNotInProgress
		}

		for vocabularyID, result := range submitted {
			delta := ExerciseMatchCorrectProgressDelta
			if result == ExerciseVocabularyResultAlmost {
				delta = ExerciseMatchAlmostProgressDelta
			}
			if result == ExerciseVocabularyResultWrong {
				delta = ExerciseMatchFailProgressDelta
			}

			if _, updateErr := updateVocabularyProgressByID(tx, exerciseID, vocabularyID, result, ExerciseVocabularyResultReasonMatchPairs, delta); updateErr != nil {
				return updateErr
			}
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, ErrExerciseNotInProgress) {
			return nil, ErrExerciseNotInProgress
		}

		return nil, err
	}

	updatedRows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, false)
	if err != nil {
		return nil, err
	}
	completedRows = make([]ExerciseListVocabulary, 0, len(updatedRows))
	for _, row := range updatedRows {
		completedRows = append(completedRows, buildListVocabularyFromExerciseDetails(row))
	}

	return &MatchPairsCompleteResult{
		Status:  status,
		Results: completedRows,
	}, nil
}

func GetCompletedMatchPairsResult(exerciseID uuid.UUID, userID uint) (*MatchPairsCompleteResult, error) {
	var exercise models.Exercise
	if err := db.DB.
		Where("id = ? AND user_id = ?", exerciseID, userID).
		Take(&exercise).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExerciseNotFound
		}
		return nil, err
	}

	if exercise.Type != enums.ExerciseTypeMatchPairs {
		return nil, ErrInvalidMatchPairResults
	}
	if exercise.Status != enums.ExerciseStatusCompleted && exercise.Status != enums.ExerciseStatusFailed {
		return nil, ErrExerciseNotInProgress
	}

	rows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, false)
	if err != nil {
		return nil, err
	}

	results := make([]ExerciseListVocabulary, 0, len(rows))
	for _, row := range rows {
		results = append(results, buildListVocabularyFromExerciseDetails(row))
	}

	return &MatchPairsCompleteResult{
		Status:  exercise.Status,
		Results: results,
	}, nil
}

func ApplyMatchTap(exerciseID uuid.UUID, userID uint, tappedIdx int) (
	board *MatchBoardState, wasWrong bool, finished bool, finalizeAttempts []MatchPairAttempt, err error,
) {
	var vocabularyDeleted bool

	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		var exercise models.Exercise
		if lockErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", exerciseID, userID).
			Take(&exercise).Error; lockErr != nil {
			if errors.Is(lockErr, gorm.ErrRecordNotFound) {
				return ErrExerciseNotFound
			}
			return lockErr
		}

		if exercise.Type != enums.ExerciseTypeMatchPairs {
			return ErrInvalidMatchPairResults
		}
		if exercise.Status != enums.ExerciseStatusInProgress {
			return ErrExerciseNotInProgress
		}

		rows, detailErr := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, true)
		if detailErr != nil {
			return detailErr
		}
		if len(rows) != matchPairsVocabularyCount {
			vocabularyDeleted = true
			return ErrExerciseVocabularyDeleted
		}

		cards := buildCanonicalMatchCards(rows)
		if tappedIdx < 0 || tappedIdx >= len(cards) {
			return ErrInvalidMatchPairResults
		}

		cardByID := make(map[string]ExerciseMatchCard, len(cards))
		for _, card := range cards {
			cardByID[card.ID] = card
		}

		expectedVocabularyIDs := make([]uuid.UUID, 0, len(rows))
		for _, row := range rows {
			expectedVocabularyIDs = append(expectedVocabularyIDs, row.VocabularyID)
		}

		var state matchStateJSON
		if exercise.MatchState == nil {
			return ErrInvalidMatchPairResults
		}
		if unmarshalErr := json.Unmarshal([]byte(*exercise.MatchState), &state); unmarshalErr != nil {
			return unmarshalErr
		}

		attemptsToPairs := func(attempts [][2]int) []MatchPairAttempt {
			pairs := make([]MatchPairAttempt, 0, len(attempts))
			for _, attempt := range attempts {
				if attempt[0] < 0 || attempt[0] >= len(cards) || attempt[1] < 0 || attempt[1] >= len(cards) {
					continue
				}
				pairs = append(pairs, MatchPairAttempt{
					FirstCardID:  cards[attempt[0]].ID,
					SecondCardID: cards[attempt[1]].ID,
				})
			}
			return pairs
		}

		states, cardWrong, replayErr := replayMatchAttempts(cardByID, expectedVocabularyIDs, attemptsToPairs(state.Attempts))
		if replayErr != nil {
			return replayErr
		}

		tappedCard := cards[tappedIdx]

		wasWrong = false
		if states[tappedCard.VocabularyID] != "" {
			board = buildMatchBoardState(state.Order, cards, state.Pending, states, cardWrong)
			finished = isMatchFinished(states)
			if finished {
				finalizeAttempts = attemptsToPairs(state.Attempts)
			}
			return nil
		}

		switch {
		case state.Pending == -1:
			state.Pending = tappedIdx
		case state.Pending == tappedIdx:
			state.Pending = -1
		default:
			resolvedCorrectBefore := countResolvedNonWrong(states)
			state.Attempts = append(state.Attempts, [2]int{state.Pending, tappedIdx})
			state.Pending = -1

			states, cardWrong, replayErr = replayMatchAttempts(cardByID, expectedVocabularyIDs, attemptsToPairs(state.Attempts))
			if replayErr != nil {
				return replayErr
			}
			wasWrong = countResolvedNonWrong(states) == resolvedCorrectBefore
		}

		stateBytes, marshalErr := json.Marshal(state)
		if marshalErr != nil {
			return marshalErr
		}

		updateResult := tx.Model(&models.Exercise{}).
			Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
			Update("match_state", string(stateBytes))
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return ErrExerciseNotInProgress
		}

		board = buildMatchBoardState(state.Order, cards, state.Pending, states, cardWrong)
		finished = isMatchFinished(states)
		if finished {
			finalizeAttempts = attemptsToPairs(state.Attempts)
		}

		return nil
	})

	if txErr != nil {
		if vocabularyDeleted {
			_ = MarkExerciseVocabularyResultWithoutProgress(exerciseID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
			_ = IgnoreExercise(exerciseID)
		}
		return nil, false, false, nil, txErr
	}

	return board, wasWrong, finished, finalizeAttempts, nil
}

func buildMatchBoardState(order []int, cards []ExerciseMatchCard, pending int, states map[uuid.UUID]string, cardWrong map[string]int) *MatchBoardState {
	resolved := make(map[uuid.UUID]string)
	for vocabularyID, result := range states {
		if result != "" {
			resolved[vocabularyID] = result
		}
	}

	return &MatchBoardState{
		Order:        order,
		Cards:        cards,
		Pending:      pending,
		Resolved:     resolved,
		CardWrong:    cardWrong,
		MatchedCount: len(resolved),
	}
}

func isMatchFinished(states map[uuid.UUID]string) bool {
	if len(states) == 0 {
		return false
	}
	for _, result := range states {
		if result == "" {
			return false
		}
	}
	return true
}

func countResolvedNonWrong(states map[uuid.UUID]string) int {
	count := 0
	for _, result := range states {
		if result == ExerciseVocabularyResultCorrect || result == ExerciseVocabularyResultAlmost {
			count++
		}
	}
	return count
}
