package services

import (
	"errors"
	"math/rand"
	"strings"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNoCollectionPracticeVocabulary          = errors.New("no collection words found in vocabulary")
	ErrCollectionPracticeVocabularyUnavailable = errors.New("collection practice word is no longer available")
)

type CollectionPracticeRound struct {
	CollectionID    uuid.UUID   `json:"collection_id"`
	CollectionTitle string      `json:"collection_title"`
	VocabularyIDs   []uuid.UUID `json:"vocabulary_ids"`
}

func collectionPracticeVocabularyIDs(conn *gorm.DB, userID uint, collectionID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := conn.
		Table("collection_translations AS ct").
		Select("v.id").
		Joins("JOIN vocabulary AS v ON v.translation_id = ct.translation_id").
		Where("ct.collection_id = ?", collectionID).
		Where("v.user_id = ? AND v.deleted_at IS NULL", userID).
		Order("ct.position ASC, v.id ASC").
		Pluck("v.id", &ids).Error
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func StartCollectionPracticeRound(userID uint, collectionID uuid.UUID) (*CollectionPracticeRound, error) {
	collection, err := getAccessibleCollection(db.DB, userID, collectionID)
	if err != nil {
		return nil, err
	}

	ids, err := collectionPracticeVocabularyIDs(db.DB, userID, collectionID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrNoCollectionPracticeVocabulary
	}

	rand.Shuffle(len(ids), func(left, right int) {
		ids[left], ids[right] = ids[right], ids[left]
	})

	return &CollectionPracticeRound{
		CollectionID:    collection.ID,
		CollectionTitle: collection.Title,
		VocabularyIDs:   ids,
	}, nil
}

func CreateCollectionPracticeExercise(
	userID uint,
	collectionID uuid.UUID,
	targetVocabularyID uuid.UUID,
	matching bool,
) (*RandomExerciseResult, error) {
	collection, err := getAccessibleCollection(db.DB, userID, collectionID)
	if err != nil {
		return nil, err
	}

	collectionVocabularyIDs, err := collectionPracticeVocabularyIDs(db.DB, userID, collectionID)
	if err != nil {
		return nil, err
	}
	if !containsUUID(collectionVocabularyIDs, targetVocabularyID) {
		return nil, ErrCollectionPracticeVocabularyUnavailable
	}

	targetVocabulary, err := loadExerciseVocabulary(targetVocabularyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCollectionPracticeVocabularyUnavailable
		}
		return nil, err
	}

	if matching {
		result, matchErr := createCollectionPracticeMatchExercise(
			userID,
			collection,
			targetVocabulary,
			collectionVocabularyIDs,
		)
		if matchErr == nil {
			return result, nil
		}
		if !errors.Is(matchErr, errNoExerciseTypeAvailable) {
			return nil, matchErr
		}
	}

	return createCollectionPracticeTargetExercise(
		userID,
		collection,
		targetVocabulary,
		collectionVocabularyIDs,
	)
}

func containsUUID(ids []uuid.UUID, target uuid.UUID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}

	return false
}

func createCollectionPracticeTargetExercise(
	userID uint,
	collection *models.Collection,
	vocabulary *models.Vocabulary,
	collectionVocabularyIDs []uuid.UUID,
) (*RandomExerciseResult, error) {
	optionsByType, err := buildCollectionPracticeOptionsByType(userID, vocabulary, collectionVocabularyIDs)
	if err != nil {
		return nil, err
	}

	exerciseType, options, err := selectCollectionPracticeExerciseType(optionsByType)
	if err != nil {
		return nil, err
	}

	questionWord, language, answerLanguage, err := buildExerciseQuestionData(vocabulary, exerciseType)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	collectionTitle := collection.Title
	exercise := models.Exercise{
		Type:                    exerciseType,
		Status:                  enums.ExerciseStatusInProgress,
		UserID:                  userID,
		StartedAt:               &now,
		PracticeCollectionID:    &collection.ID,
		PracticeCollectionTitle: &collectionTitle,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&exercise).Error; err != nil {
			return err
		}

		return createExerciseVocabularyLinks(tx, exercise.ID, vocabulary.ID, options)
	})
	if err != nil {
		return nil, err
	}

	resultOptions := collectExerciseOptionLabels(options)
	if isCharacterExerciseType(exerciseType) {
		resultOptions = ShuffledAnswerCharacters(options[0].AnswerWord)
	}

	return &RandomExerciseResult{
		ExerciseID:     exercise.ID,
		Type:           exerciseType,
		QuestionWord:   questionWord,
		Language:       language,
		AnswerLanguage: answerLanguage,
		Options:        resultOptions,
	}, nil
}

func buildCollectionPracticeOptionsByType(
	userID uint,
	vocabulary *models.Vocabulary,
	collectionVocabularyIDs []uuid.UUID,
) (map[enums.ExerciseType][]exerciseChoiceCandidate, error) {
	optionsByType := make(map[enums.ExerciseType][]exerciseChoiceCandidate, 6)

	for _, exerciseType := range []enums.ExerciseType{
		enums.ExerciseTypeBasicDirect,
		enums.ExerciseTypeBasicReversed,
	} {
		options, err := buildCollectionPracticeOptionsForType(
			userID,
			vocabulary,
			exerciseType,
			collectionVocabularyIDs,
		)
		if err != nil {
			return nil, err
		}
		if len(options) == 0 {
			continue
		}

		optionsByType[exerciseType] = append([]exerciseChoiceCandidate(nil), options[:1]...)

		characterType := enums.ExerciseTypeCharactersDirect
		choiceType := enums.ExerciseTypeChoiceDirect
		if isReversedExerciseType(exerciseType) {
			characterType = enums.ExerciseTypeCharactersReversed
			choiceType = enums.ExerciseTypeChoiceReversed
		}

		if strings.TrimSpace(options[0].AnswerWord) != "" {
			optionsByType[characterType] = append([]exerciseChoiceCandidate(nil), options[:1]...)
		}
		if len(options) == choiceExerciseVocabularyCount {
			optionsByType[choiceType] = shuffledExerciseOptions(options)
		}
	}

	return optionsByType, nil
}

func buildCollectionPracticeOptionsForType(
	userID uint,
	vocabulary *models.Vocabulary,
	exerciseType enums.ExerciseType,
	collectionVocabularyIDs []uuid.UUID,
) ([]exerciseChoiceCandidate, error) {
	if vocabulary == nil || vocabulary.Translation == nil ||
		vocabulary.Translation.Original == nil || vocabulary.Translation.Translation == nil {
		return nil, errors.New("vocabulary has no translation")
	}

	translation := vocabulary.Translation
	answerOption := exerciseChoiceCandidate{
		VocabularyID: vocabulary.ID,
		AnswerWord:   translation.Translation.Word,
	}
	answerColumn := "translated.word"
	if isReversedExerciseType(exerciseType) {
		answerOption.AnswerWord = translation.Original.Word
		answerColumn = "original.word"
	}

	distractors, err := getCollectionPracticeDistractors(
		userID,
		translation.Original.Language,
		translation.Translation.Language,
		answerColumn,
		vocabulary.ID,
		answerOption.AnswerWord,
		collectionVocabularyIDs,
	)
	if err != nil {
		return nil, err
	}

	options := []exerciseChoiceCandidate{answerOption}
	if len(distractors) >= choiceExerciseVocabularyCount-1 {
		options = append(options, distractors[:choiceExerciseVocabularyCount-1]...)
	}

	return options, nil
}

func getCollectionPracticeDistractors(
	userID uint,
	originalLanguage enums.Language,
	translationLanguage enums.Language,
	answerColumn string,
	correctVocabularyID uuid.UUID,
	correctAnswer string,
	collectionVocabularyIDs []uuid.UUID,
) ([]exerciseChoiceCandidate, error) {
	query := `
		SELECT
			v.id AS vocabulary_id,
			` + answerColumn + ` AS answer_word
		FROM vocabulary AS v
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE v.user_id = ?
			AND v.deleted_at IS NULL
			AND original.language = ?
			AND translated.language = ?
			AND v.id <> ?
		ORDER BY CASE WHEN v.id IN ? THEN 0 ELSE 1 END, RANDOM()
	`

	var rows []exerciseChoiceCandidate
	if err := db.DB.Raw(
		query,
		userID,
		originalLanguage,
		translationLanguage,
		correctVocabularyID,
		collectionVocabularyIDs,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}

	seen := map[string]struct{}{normalizeAnswer(correctAnswer): {}}
	options := make([]exerciseChoiceCandidate, 0, len(rows))
	for _, row := range rows {
		normalized := normalizeAnswer(row.AnswerWord)
		if strings.TrimSpace(normalized) == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		options = append(options, row)
	}

	return options, nil
}

func selectCollectionPracticeExerciseType(
	optionsByType map[enums.ExerciseType][]exerciseChoiceCandidate,
) (enums.ExerciseType, []exerciseChoiceCandidate, error) {
	type exerciseTypeGroup struct {
		weight int
		types  []enums.ExerciseType
	}

	groups := []exerciseTypeGroup{
		{weight: basicExerciseWeight, types: []enums.ExerciseType{
			enums.ExerciseTypeBasicDirect,
			enums.ExerciseTypeBasicReversed,
		}},
		{weight: choiceExerciseWeight, types: []enums.ExerciseType{
			enums.ExerciseTypeChoiceDirect,
			enums.ExerciseTypeChoiceReversed,
		}},
		{weight: characterExerciseWeight, types: []enums.ExerciseType{
			enums.ExerciseTypeCharactersDirect,
			enums.ExerciseTypeCharactersReversed,
		}},
	}

	availableGroups := make([]exerciseTypeGroup, 0, len(groups))
	totalWeight := 0
	for _, group := range groups {
		availableTypes := make([]enums.ExerciseType, 0, len(group.types))
		for _, exerciseType := range group.types {
			if isExerciseTypeAvailable(exerciseType, optionsByType[exerciseType]) {
				availableTypes = append(availableTypes, exerciseType)
			}
		}
		if len(availableTypes) == 0 {
			continue
		}

		group.types = availableTypes
		availableGroups = append(availableGroups, group)
		totalWeight += group.weight
	}
	if len(availableGroups) == 0 {
		return "", nil, errNoExerciseTypeAvailable
	}

	roll := rand.Intn(totalWeight)
	selectedGroup := availableGroups[len(availableGroups)-1]
	for _, group := range availableGroups {
		if roll < group.weight {
			selectedGroup = group
			break
		}
		roll -= group.weight
	}

	exerciseType := selectedGroup.types[rand.Intn(len(selectedGroup.types))]
	return exerciseType, append([]exerciseChoiceCandidate(nil), optionsByType[exerciseType]...), nil
}

func createCollectionPracticeMatchExercise(
	userID uint,
	collection *models.Collection,
	targetVocabulary *models.Vocabulary,
	collectionVocabularyIDs []uuid.UUID,
) (*RandomExerciseResult, error) {
	if targetVocabulary == nil || targetVocabulary.Translation == nil ||
		targetVocabulary.Translation.Original == nil || targetVocabulary.Translation.Translation == nil {
		return nil, errors.New("vocabulary has no translation")
	}

	translation := targetVocabulary.Translation
	query := `
		SELECT
			v.id AS vocabulary_id,
			original.word AS original_word,
			translated.word AS translation_word
		FROM vocabulary AS v
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE v.user_id = ?
			AND v.deleted_at IS NULL
			AND v.id IN ?
			AND original.language = ?
			AND translated.language = ?
		ORDER BY RANDOM()
	`

	var rows []exerciseMatchPairCandidate
	if err := db.DB.Raw(
		query,
		userID,
		collectionVocabularyIDs,
		translation.Original.Language,
		translation.Translation.Language,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}

	seenOriginal := make(map[string]struct{})
	seenTranslation := make(map[string]struct{})
	options := make([]exerciseChoiceCandidate, 0, matchPairsVocabularyCount)
	for _, row := range rows {
		original := normalizeAnswer(row.OriginalWord)
		translated := normalizeAnswer(row.TranslationWord)
		if strings.TrimSpace(original) == "" || strings.TrimSpace(translated) == "" {
			continue
		}
		if _, exists := seenOriginal[original]; exists {
			continue
		}
		if _, exists := seenTranslation[translated]; exists {
			continue
		}

		seenOriginal[original] = struct{}{}
		seenTranslation[translated] = struct{}{}
		options = append(options, exerciseChoiceCandidate{VocabularyID: row.VocabularyID})
		if len(options) == matchPairsVocabularyCount {
			break
		}
	}
	if len(options) != matchPairsVocabularyCount {
		return nil, errNoExerciseTypeAvailable
	}

	now := time.Now().UTC()
	collectionTitle := collection.Title
	exercise := models.Exercise{
		Type:                    enums.ExerciseTypeMatchPairs,
		Status:                  enums.ExerciseStatusInProgress,
		UserID:                  userID,
		StartedAt:               &now,
		PracticeCollectionID:    &collection.ID,
		PracticeCollectionTitle: &collectionTitle,
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&exercise).Error; err != nil {
			return err
		}

		return createExerciseVocabularyLinks(tx, exercise.ID, uuid.Nil, options)
	})
	if err != nil {
		return nil, err
	}

	cards, err := GetExerciseMatchCards(exercise.ID)
	if err != nil {
		return nil, err
	}

	return &RandomExerciseResult{
		ExerciseID: exercise.ID,
		Type:       enums.ExerciseTypeMatchPairs,
		Cards:      cards,
	}, nil
}
