package services

import (
	"strings"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"termorize/src/utils"

	"github.com/google/uuid"
)

var webRussianYoReplacer = strings.NewReplacer("ё", "е", "Ё", "Е")

func normalizeAnswer(value string) string {
	return strings.ToLower(webRussianYoReplacer.Replace(strings.TrimSpace(value)))
}

func almostCorrectThreshold(expected string) int {
	if len([]rune(expected)) > 10 {
		return 2
	}
	return 1
}

func VerifyExerciseAnswer(exerciseID uuid.UUID, userID uint, answer string) (*VerifyAnswerResult, error) {
	exercise, correctVocabulary, err := getExerciseWithCorrectVocabulary(exerciseID, userID)
	if err != nil {
		return nil, err
	}

	if exercise.Status != enums.ExerciseStatusInProgress {
		return nil, exerciseNotInProgressError(db.DB, exercise.ID)
	}

	if isMatchPairsExerciseType(exercise.Type) {
		return nil, ErrInvalidMatchPairResults
	}

	if correctVocabulary == nil {
		_ = MarkExerciseVocabularyResultWithoutProgress(exercise.ID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
		_ = IgnoreExercise(exercise.ID)
		return nil, ErrExerciseVocabularyDeleted
	}

	expectedAnswer := correctVocabulary.TranslationWord
	if isReversedExerciseType(exercise.Type) {
		expectedAnswer = correctVocabulary.OriginalWord
	}

	normalizedAnswer := normalizeAnswer(answer)
	normalizedExpectedAnswer := normalizeAnswer(expectedAnswer)

	var updated bool
	var knowledge int
	var progressDelta int
	var resultType string
	deltas := exerciseProgressDeltasForType(exercise.Type)

	if isChoiceExerciseType(exercise.Type) {
		if normalizedAnswer == normalizedExpectedAnswer {
			progressDelta = exerciseProgressDelta(exercise, deltas.Correct)
			updated, knowledge, progressDelta, err = FinishExerciseWithProgressDelta(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultCorrect, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
			resultType = "correct"
		} else {
			progressDelta = exerciseProgressDelta(exercise, deltas.Wrong)
			updated, knowledge, progressDelta, err = FinishExerciseWithProgressDelta(exerciseID, enums.ExerciseStatusFailed, ExerciseVocabularyResultWrong, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
			resultType = "wrong"
		}
	} else {
		answerReason := ExerciseVocabularyResultReasonTypedAnswer
		if isCharacterExerciseType(exercise.Type) {
			answerReason = ExerciseVocabularyResultReasonCharacterAnswer
		}

		if normalizedAnswer == normalizedExpectedAnswer {
			progressDelta = exerciseProgressDelta(exercise, deltas.Correct)
			updated, knowledge, progressDelta, err = FinishExerciseWithProgressDelta(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultCorrect, answerReason, progressDelta)
			resultType = "correct"
		} else {
			distance := utils.DamerauLevenshteinDistance(normalizedAnswer, normalizedExpectedAnswer)
			threshold := almostCorrectThreshold(normalizedExpectedAnswer)
			if distance <= threshold {
				progressDelta = exerciseProgressDelta(exercise, deltas.Almost)
				updated, knowledge, progressDelta, err = FinishExerciseWithProgressDelta(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultAlmost, answerReason, progressDelta)
				resultType = "almost"
			} else {
				progressDelta = exerciseProgressDelta(exercise, deltas.Wrong)
				updated, knowledge, progressDelta, err = FinishExerciseWithProgressDelta(exerciseID, enums.ExerciseStatusFailed, ExerciseVocabularyResultWrong, answerReason, progressDelta)
				resultType = "wrong"
			}
		}
	}

	if err != nil {
		return nil, err
	}

	if !updated {
		return nil, ErrExerciseNotInProgress
	}

	return &VerifyAnswerResult{
		Result:        resultType,
		CorrectAnswer: expectedAnswer,
		Knowledge:     knowledge,
		ProgressDelta: progressDelta,
	}, nil
}

func VerifyExerciseChoice(exerciseID uuid.UUID, userID uint, selectedVocabularyID uuid.UUID) (*VerifyAnswerResult, error) {
	exercise, correctVocabulary, err := getExerciseWithCorrectVocabulary(exerciseID, userID)
	if err != nil {
		return nil, err
	}

	if exercise.Status != enums.ExerciseStatusInProgress {
		return nil, exerciseNotInProgressError(db.DB, exercise.ID)
	}

	if isMatchPairsExerciseType(exercise.Type) {
		return nil, ErrInvalidMatchPairResults
	}

	if correctVocabulary == nil {
		_ = MarkExerciseVocabularyResultWithoutProgress(exercise.ID, ExerciseVocabularyResultIgnored, ExerciseVocabularyResultReasonDeletedVocabulary)
		_ = IgnoreExercise(exercise.ID)
		return nil, ErrExerciseVocabularyDeleted
	}

	correctAnswer := correctVocabulary.TranslationWord
	if isReversedExerciseType(exercise.Type) {
		correctAnswer = correctVocabulary.OriginalWord
	}

	var updated bool
	var knowledge int
	var progressDelta int
	var resultType string
	deltas := exerciseProgressDeltasForType(exercise.Type)

	if selectedVocabularyID == correctVocabulary.VocabularyID {
		progressDelta = exerciseProgressDelta(exercise, deltas.Correct)
		updated, knowledge, progressDelta, err = FinishExerciseWithProgressDelta(exerciseID, enums.ExerciseStatusCompleted, ExerciseVocabularyResultCorrect, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
		resultType = "correct"
	} else {
		progressDelta = exerciseProgressDelta(exercise, deltas.Wrong)
		updated, knowledge, progressDelta, err = FinishExerciseWithProgressDelta(exerciseID, enums.ExerciseStatusFailed, ExerciseVocabularyResultWrong, ExerciseVocabularyResultReasonChoiceAnswer, progressDelta)
		resultType = "wrong"
	}

	if err != nil {
		return nil, err
	}

	if !updated {
		return nil, ErrExerciseNotInProgress
	}

	return &VerifyAnswerResult{
		Result:        resultType,
		CorrectAnswer: correctAnswer,
		Knowledge:     knowledge,
		ProgressDelta: progressDelta,
	}, nil
}

func exerciseProgressDelta(exercise *models.Exercise, regularDelta int) int {
	if exercise != nil && exercise.PracticeCollectionTitle != nil {
		return 0
	}

	return regularDelta
}

func exerciseProgressDeltasForType(exerciseType enums.ExerciseType) exerciseProgressDeltas {
	switch exerciseType {
	case enums.ExerciseTypeBasicDirect, enums.ExerciseTypeBasicReversed,
		enums.ExerciseTypeAudioDirect, enums.ExerciseTypeAudioReversed:
		return exerciseProgressDeltas{
			Correct: ExerciseBasicCorrectProgressDelta,
			Almost:  ExerciseBasicAlmostProgressDelta,
			Wrong:   ExerciseBasicWrongProgressDelta,
		}
	case enums.ExerciseTypeCharactersDirect, enums.ExerciseTypeCharactersReversed:
		return exerciseProgressDeltas{
			Correct: ExerciseCharacterCorrectProgressDelta,
			Almost:  ExerciseCharacterAlmostProgressDelta,
			Wrong:   ExerciseCharacterWrongProgressDelta,
		}
	case enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed:
		return exerciseProgressDeltas{
			Correct: ExerciseChoiceCorrectProgressDelta,
			Wrong:   ExerciseChoiceWrongProgressDelta,
		}
	case enums.ExerciseTypeMatchPairs:
		return exerciseProgressDeltas{
			Correct: ExerciseMatchCorrectProgressDelta,
			Almost:  ExerciseMatchAlmostProgressDelta,
			Wrong:   ExerciseMatchWrongProgressDelta,
		}
	default:
		return exerciseProgressDeltas{}
	}
}

func isReversedExerciseType(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeBasicReversed, enums.ExerciseTypeChoiceReversed, enums.ExerciseTypeCharactersReversed,
		enums.ExerciseTypeAudioReversed:
		return true
	default:
		return false
	}
}

func isCharacterExerciseType(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeCharactersDirect, enums.ExerciseTypeCharactersReversed:
		return true
	default:
		return false
	}
}

func isChoiceExerciseType(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed:
		return true
	default:
		return false
	}
}

func isMatchPairsExerciseType(exerciseType enums.ExerciseType) bool {
	return exerciseType == enums.ExerciseTypeMatchPairs
}

func isAudioExerciseType(exerciseType enums.ExerciseType) bool {
	return exerciseType == enums.ExerciseTypeAudioDirect || exerciseType == enums.ExerciseTypeAudioReversed
}
