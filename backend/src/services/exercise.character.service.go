package services

import (
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"strings"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func AnswerCharacters(answer string) []string {
	trimmed := strings.TrimSpace(answer)
	characters := make([]string, 0, len([]rune(trimmed)))
	for _, character := range []rune(trimmed) {
		characters = append(characters, string(character))
	}

	return characters
}

func ShuffledAnswerCharacters(answer string) []string {
	characters := AnswerCharacters(answer)
	rand.Shuffle(len(characters), func(i, j int) {
		characters[i], characters[j] = characters[j], characters[i]
	})

	return characters
}

func BuildCharacterBoardForAnswer(answer string) *CharacterBoardState {
	characters := AnswerCharacters(answer)
	order := make([]int, characterBoardSlotCount(len(characters)))
	for index := range characters {
		order[index] = index
	}
	for index := len(characters); index < len(order); index++ {
		order[index] = -1
	}
	rand.Shuffle(len(order), func(i, j int) {
		order[i], order[j] = order[j], order[i]
	})

	return buildCharacterBoardState(order, characters, nil)
}

func StartCharacterExercise(exerciseID uuid.UUID, telegramMessageID int64, order []int) error {
	stateBytes, err := json.Marshal(characterStateJSON{
		Order:  append([]int(nil), order...),
		Chosen: []int{},
	})
	if err != nil {
		return err
	}

	result := db.DB.Model(&models.Exercise{}).
		Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusPending).
		Updates(map[string]any{
			"status":              enums.ExerciseStatusInProgress,
			"telegram_message_id": telegramMessageID,
			"started_at":          time.Now().UTC(),
			"character_state":     string(stateBytes),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrExerciseNotInProgress
	}

	return nil
}

func ApplyCharacterTap(exerciseID uuid.UUID, userID uint, tappedIndex int) (*CharacterBoardState, bool, error) {
	var board *CharacterBoardState
	var finished bool
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

		if !isCharacterExerciseType(exercise.Type) {
			return ErrInvalidCharacterResults
		}
		if exercise.Status != enums.ExerciseStatusInProgress {
			return ErrExerciseNotInProgress
		}

		correctVocabulary, detailErr := getCorrectExerciseVocabularyDetails(exercise.ID)
		if detailErr != nil {
			return detailErr
		}
		if correctVocabulary == nil {
			vocabularyDeleted = true
			return ErrExerciseVocabularyDeleted
		}

		answer := correctVocabulary.TranslationWord
		if isReversedExerciseType(exercise.Type) {
			answer = correctVocabulary.OriginalWord
		}
		characters := AnswerCharacters(answer)
		if len(characters) == 0 || tappedIndex < 0 || tappedIndex >= len(characters) {
			return ErrInvalidCharacterResults
		}

		if exercise.CharacterState == nil {
			return ErrInvalidCharacterResults
		}

		var state characterStateJSON
		if unmarshalErr := json.Unmarshal([]byte(*exercise.CharacterState), &state); unmarshalErr != nil {
			return unmarshalErr
		}
		if !validCharacterState(state, len(characters)) {
			return ErrInvalidCharacterResults
		}

		if len(state.Chosen) >= len(characters) || containsInt(state.Chosen, tappedIndex) {
			board = buildCharacterBoardState(state.Order, characters, state.Chosen)
			finished = len(state.Chosen) == len(characters)
			return nil
		}

		state.Chosen = append(state.Chosen, tappedIndex)
		stateBytes, marshalErr := json.Marshal(state)
		if marshalErr != nil {
			return marshalErr
		}

		updateResult := tx.Model(&models.Exercise{}).
			Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
			Update("character_state", string(stateBytes))
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return ErrExerciseNotInProgress
		}

		board = buildCharacterBoardState(state.Order, characters, state.Chosen)
		finished = len(state.Chosen) == len(characters)
		return nil
	})

	if txErr != nil {
		if vocabularyDeleted {
			_ = MarkExerciseVocabularyResultWithoutProgress(exerciseID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
			_ = IgnoreExercise(exerciseID)
		}
		return nil, false, txErr
	}

	return board, finished, nil
}

func RemoveLastCharacterSelection(exerciseID uuid.UUID, userID uint) (*CharacterBoardState, error) {
	var board *CharacterBoardState
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

		if !isCharacterExerciseType(exercise.Type) {
			return ErrInvalidCharacterResults
		}
		if exercise.Status != enums.ExerciseStatusInProgress {
			return ErrExerciseNotInProgress
		}

		correctVocabulary, detailErr := getCorrectExerciseVocabularyDetails(exercise.ID)
		if detailErr != nil {
			return detailErr
		}
		if correctVocabulary == nil {
			vocabularyDeleted = true
			return ErrExerciseVocabularyDeleted
		}

		answer := correctVocabulary.TranslationWord
		if isReversedExerciseType(exercise.Type) {
			answer = correctVocabulary.OriginalWord
		}
		characters := AnswerCharacters(answer)

		if exercise.CharacterState == nil {
			return ErrInvalidCharacterResults
		}
		var state characterStateJSON
		if unmarshalErr := json.Unmarshal([]byte(*exercise.CharacterState), &state); unmarshalErr != nil {
			return unmarshalErr
		}
		if !validCharacterState(state, len(characters)) {
			return ErrInvalidCharacterResults
		}

		if len(state.Chosen) > 0 {
			state.Chosen = state.Chosen[:len(state.Chosen)-1]
		}
		stateBytes, marshalErr := json.Marshal(state)
		if marshalErr != nil {
			return marshalErr
		}
		updateResult := tx.Model(&models.Exercise{}).
			Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusInProgress).
			Update("character_state", string(stateBytes))
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return ErrExerciseNotInProgress
		}

		board = buildCharacterBoardState(state.Order, characters, state.Chosen)
		return nil
	})

	if txErr != nil {
		if vocabularyDeleted {
			_ = MarkExerciseVocabularyResultWithoutProgress(exerciseID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
			_ = IgnoreExercise(exerciseID)
		}
		return nil, txErr
	}

	return board, nil
}

func characterBoardSide(characterCount int) int {
	if characterCount <= 0 {
		return 0
	}
	return int(math.Ceil(math.Sqrt(float64(characterCount))))
}

func characterBoardSlotCount(characterCount int) int {
	side := characterBoardSide(characterCount)
	if side == 0 {
		return 0
	}
	return side * side
}

func validCharacterState(state characterStateJSON, characterCount int) bool {
	currentSlotCount := characterBoardSlotCount(characterCount)
	legacySide := int(math.Ceil(math.Sqrt(float64(characterCount + 1))))
	legacySlotCount := legacySide*legacySide - 1
	if (len(state.Order) != currentSlotCount && len(state.Order) != legacySlotCount) || len(state.Chosen) > characterCount {
		return false
	}

	seenOrder := make(map[int]bool, characterCount)
	for _, index := range state.Order {
		if index == -1 {
			continue
		}
		if index < 0 || index >= characterCount || seenOrder[index] {
			return false
		}
		seenOrder[index] = true
	}
	if len(seenOrder) != characterCount {
		return false
	}

	seenChosen := make(map[int]bool, len(state.Chosen))
	for _, index := range state.Chosen {
		if index < 0 || index >= characterCount || seenChosen[index] {
			return false
		}
		seenChosen[index] = true
	}

	return true
}

func containsInt(values []int, expected int) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func buildCharacterBoardState(order []int, characters []string, chosen []int) *CharacterBoardState {
	var answer strings.Builder
	for _, index := range chosen {
		if index >= 0 && index < len(characters) {
			answer.WriteString(characters[index])
		}
	}

	return &CharacterBoardState{
		Order:      append([]int(nil), order...),
		Characters: append([]string(nil), characters...),
		Chosen:     append([]int(nil), chosen...),
		Answer:     answer.String(),
	}
}
