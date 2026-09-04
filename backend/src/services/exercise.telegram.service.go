package services

import (
	"encoding/json"
	"errors"
	"math/rand"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetDuePendingExercises(now time.Time) ([]PendingExercise, error) {
	var exercises []PendingExercise

	err := db.DB.Raw(`
		SELECT
			e.id AS exercise_id,
			e.type AS exercise_type,
			e.is_known_vocabulary_repetition AS is_known_vocabulary_repetition,
			e.user_id AS user_id,
			u.username AS username,
			u.telegram_id AS telegram_id,
			original.word AS original_word,
			original.id AS original_word_id,
			original.language AS original_language,
			translated.word AS translation_word,
			translated.id AS translation_word_id,
			translated.language AS translation_language,
			t.id AS translation_id,
			u.settings->>'system_language' AS system_language
		FROM exercises AS e
		JOIN users AS u ON u.id = e.user_id
		JOIN vocabulary_exercises AS ve ON ve.exercise_id = e.id AND ve.is_correct = true
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE e.deleted_at IS NULL
			AND u.deleted_at IS NULL
			AND e.status = ?
			AND e.type IN (?, ?, ?, ?, ?, ?, ?, ?, ?)
			AND e.scheduled_for <= ?
			AND u.settings->'telegram'->'bot_enabled' = ?
		ORDER BY e.scheduled_for ASC, e.created_at ASC
	`, enums.ExerciseStatusPending, enums.ExerciseTypeBasicDirect, enums.ExerciseTypeBasicReversed, enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed, enums.ExerciseTypeCharactersDirect, enums.ExerciseTypeCharactersReversed, enums.ExerciseTypeAudioDirect, enums.ExerciseTypeAudioReversed, enums.ExerciseTypeDescriptionReversed, now, true).Scan(&exercises).Error

	if err != nil {
		return nil, err
	}

	return exercises, nil
}

func GetDuePendingMatchExercises(now time.Time) ([]PendingMatchExercise, error) {
	var exercises []PendingMatchExercise

	err := db.DB.Raw(`
		SELECT
			e.id AS exercise_id,
			e.user_id AS user_id,
			u.username AS username,
			u.telegram_id AS telegram_id,
			u.settings->>'system_language' AS system_language
		FROM exercises AS e
		JOIN users AS u ON u.id = e.user_id
		WHERE e.deleted_at IS NULL
			AND u.deleted_at IS NULL
			AND e.status = ?
			AND e.type = ?
			AND e.scheduled_for <= ?
			AND u.settings->'telegram'->'bot_enabled' = ?
			AND (
				SELECT COUNT(*)
				FROM vocabulary_exercises AS ve
				JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
				WHERE ve.exercise_id = e.id AND ve.is_correct = true
			) = ?
		ORDER BY e.scheduled_for ASC, e.created_at ASC
	`, enums.ExerciseStatusPending, enums.ExerciseTypeMatchPairs, now, true, matchPairsVocabularyCount).Scan(&exercises).Error

	if err != nil {
		return nil, err
	}

	return exercises, nil
}

func buildCanonicalMatchCards(rows []exerciseVocabularyDetails) []ExerciseMatchCard {
	cards := make([]ExerciseMatchCard, 0, len(rows)*2)
	for _, row := range rows {
		cards = append(cards, ExerciseMatchCard{
			ID:           row.VocabularyID.String() + ":" + matchPairCardSideOriginal,
			VocabularyID: row.VocabularyID,
			Word:         row.OriginalWord,
			Language:     row.OriginalLanguage,
			Side:         matchPairCardSideOriginal,
		})
		cards = append(cards, ExerciseMatchCard{
			ID:           row.VocabularyID.String() + ":" + matchPairCardSideTranslation,
			VocabularyID: row.VocabularyID,
			Word:         row.TranslationWord,
			Language:     row.TranslationLanguage,
			Side:         matchPairCardSideTranslation,
		})
	}

	return cards
}

func BuildMatchBoard(exerciseID uuid.UUID) ([]ExerciseMatchCard, []int, error) {
	rows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, true)
	if err != nil {
		return nil, nil, err
	}
	if len(rows) != matchPairsVocabularyCount {
		return nil, nil, ErrExerciseVocabularyDeleted
	}

	cards := buildCanonicalMatchCards(rows)

	order := make([]int, len(cards))
	for i := range order {
		order[i] = i
	}
	rand.Shuffle(len(order), func(i, j int) {
		order[i], order[j] = order[j], order[i]
	})

	return cards, order, nil
}

func StartMatchExercise(exerciseID uuid.UUID, telegramMessageID int64, order []int) error {
	stateBytes, err := json.Marshal(matchStateJSON{Order: order, Pending: -1, Attempts: [][2]int{}})
	if err != nil {
		return err
	}

	result := db.DB.Model(&models.Exercise{}).
		Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusPending).
		Updates(map[string]any{
			"status":              enums.ExerciseStatusInProgress,
			"telegram_message_id": telegramMessageID,
			"started_at":          time.Now().UTC(),
			"match_state":         string(stateBytes),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrExerciseNotInProgress
	}

	return nil
}

func GetExerciseByTelegramMessage(telegramMessageID int64, telegramID int64) (*TelegramMessageExercise, error) {
	var exercise models.Exercise

	err := db.DB.Unscoped().
		Model(&models.Exercise{}).
		Joins("JOIN users AS u ON u.id = exercises.user_id").
		Where("exercises.telegram_message_id = ?", telegramMessageID).
		Where("u.telegram_id = ?", telegramID).
		Where("u.deleted_at IS NULL").
		First(&exercise).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return buildTelegramMessageExercise(exercise)
}

func GetExerciseByTelegramExerciseID(exerciseID uuid.UUID, telegramID int64) (*TelegramMessageExercise, error) {
	var exercise models.Exercise

	err := db.DB.Unscoped().
		Model(&models.Exercise{}).
		Joins("JOIN users AS u ON u.id = exercises.user_id").
		Where("exercises.id = ?", exerciseID).
		Where("u.telegram_id = ?", telegramID).
		Where("u.deleted_at IS NULL").
		First(&exercise).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return buildTelegramMessageExercise(exercise)
}

func buildTelegramMessageExercise(exercise models.Exercise) (*TelegramMessageExercise, error) {
	var resultReason string
	if err := db.DB.Model(&models.ExerciseVocabulary{}).
		Select("result_reason").
		Where("exercise_id = ? AND is_correct = ? AND result_reason IS NOT NULL", exercise.ID, true).
		Order("position ASC").
		Limit(1).
		Scan(&resultReason).Error; err != nil {
		return nil, err
	}

	correctVocabulary, err := getCorrectExerciseVocabularyDetails(exercise.ID)
	if err != nil {
		return nil, err
	}

	options, err := GetExerciseAnswerOptions(exercise.ID, exercise.Type)
	if err != nil {
		return nil, err
	}

	telegramExercise := TelegramMessageExercise{
		ExerciseID:   exercise.ID,
		ExerciseType: exercise.Type,
		Status:       exercise.Status,
		Deleted:      exercise.DeletedAt.Valid,
		ResultReason: resultReason,
		UserID:       exercise.UserID,
		Options:      options,
	}

	if correctVocabulary != nil {
		telegramExercise.OriginalWord = correctVocabulary.OriginalWord
		telegramExercise.OriginalLanguage = correctVocabulary.OriginalLanguage
		telegramExercise.TranslationWord = correctVocabulary.TranslationWord
		telegramExercise.TranslationLanguage = correctVocabulary.TranslationLanguage
		telegramExercise.Vocabulary = []models.Vocabulary{buildVocabularyFromExerciseDetails(*correctVocabulary)}

		if isCharacterExerciseType(exercise.Type) && exercise.CharacterState != nil {
			answer := correctVocabulary.TranslationWord
			if isReversedExerciseType(exercise.Type) {
				answer = correctVocabulary.OriginalWord
			}
			characters := AnswerCharacters(answer)

			var state characterStateJSON
			if unmarshalErr := json.Unmarshal([]byte(*exercise.CharacterState), &state); unmarshalErr != nil {
				return nil, unmarshalErr
			}
			if !validCharacterState(state, len(characters)) {
				return nil, ErrInvalidCharacterResults
			}
			telegramExercise.CharacterBoard = buildCharacterBoardState(state.Order, characters, state.Chosen)
		}
	}

	return &telegramExercise, nil
}

func StartTelegramExercise(exerciseID uuid.UUID, telegramMessageID int64) error {
	result := db.DB.Model(&models.Exercise{}).
		Where("id = ? AND status = ?", exerciseID, enums.ExerciseStatusPending).
		Updates(map[string]any{
			"status":              enums.ExerciseStatusInProgress,
			"telegram_message_id": telegramMessageID,
			"started_at":          time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Delivery happens before this transition. If a concurrent language
		// ignore soft-deleted the row after Telegram accepted the audio, retain
		// the message id so replies resolve to the cancelled exercise.
		cancelled := db.DB.Unscoped().Model(&models.Exercise{}).
			Where("id = ? AND deleted_at IS NOT NULL AND telegram_message_id IS NULL", exerciseID).
			Where("type IN ?", []enums.ExerciseType{enums.ExerciseTypeAudioDirect, enums.ExerciseTypeAudioReversed}).
			Update("telegram_message_id", telegramMessageID)
		if cancelled.Error != nil {
			return cancelled.Error
		}
		return ErrExerciseNotInProgress
	}
	return nil
}
