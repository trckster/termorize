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

type RandomExerciseResult struct {
	ExerciseID     uuid.UUID
	Type           enums.ExerciseType
	QuestionWord   string
	Language       enums.Language
	AnswerLanguage enums.Language
	AudioWordID    *uuid.UUID
	Options        []string
	Cards          []ExerciseMatchCard
	Description    string
}

func CreateRandomExercise(userID uint) (*RandomExerciseResult, error) {
	return createRandomExerciseOfTypes(userID, nil, false, false)
}

func CreateRandomExerciseExcludingAudio(userID uint) (*RandomExerciseResult, error) {
	return createRandomExerciseOfTypes(userID, nil, true, false)
}

// CreateRandomExerciseOfTypes creates an exercise using only the requested
// concrete types. Selection between the available requested types is random.
func CreateRandomExerciseOfTypes(userID uint, exerciseTypes ...enums.ExerciseType) (*RandomExerciseResult, error) {
	if len(exerciseTypes) == 0 {
		return nil, errNoExerciseTypeAvailable
	}

	return createRandomExerciseOfTypes(userID, exerciseTypes, false, false)
}

func createRandomExerciseOfTypes(userID uint, exerciseTypes []enums.ExerciseType, excludeAudio, excludeDescription bool) (*RandomExerciseResult, error) {
	ids, err := getEligibleVocabularyIDs(userID, 64)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		hasVocabulary, err := userHasVocabulary(userID)
		if err != nil {
			return nil, err
		}

		if !hasVocabulary {
			return nil, ErrNoVocabularyForExercise
		}

		return nil, ErrAllVocabularyMastered
	}

	for _, vocabularyID := range ids {
		result, createErr := createRandomExerciseForVocabulary(userID, vocabularyID, exerciseTypes, excludeAudio, excludeDescription)
		if errors.Is(createErr, errNoExerciseTypeAvailable) {
			continue
		}

		if createErr != nil {
			return nil, createErr
		}

		return result, nil
	}

	return nil, ErrNoVocabularyForExercise
}

func userHasVocabulary(userID uint) (bool, error) {
	var count int64

	err := db.DB.
		Model(&models.Vocabulary{}).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func createRandomExerciseForVocabulary(userID uint, vocabularyID uuid.UUID, requestedTypes []enums.ExerciseType, excludeAudio, excludeDescription bool) (*RandomExerciseResult, error) {
	vocabulary, err := loadExerciseVocabulary(vocabularyID)
	if err != nil {
		return nil, err
	}

	var exerciseType enums.ExerciseType
	var options []exerciseChoiceCandidate
	if len(requestedTypes) == 0 {
		exerciseType, options, err = selectExerciseTypeAndOptionsWithConfig(db.DB, userID, vocabulary, true, excludeAudio, excludeDescription)
	} else {
		exerciseType, options, err = selectRequestedExerciseTypeAndOptions(userID, vocabulary, requestedTypes)
	}
	if err != nil {
		return nil, err
	}
	if exerciseType == enums.ExerciseTypeMatchPairs {
		return createRandomMatchPairsExercise(userID, options)
	}

	questionWord, language, answerLanguage, err := buildExerciseQuestionData(vocabulary, exerciseType)
	if err != nil {
		return nil, err
	}
	var description string
	if isDescriptionExerciseType(exerciseType) {
		generated, generationErr := GetOrCreateTranslationDescription(vocabulary.Translation.ID)
		if generationErr != nil {
			if len(requestedTypes) > 0 {
				return nil, generationErr
			}
			exerciseType, options, err = selectExerciseTypeAndOptionsWithConfig(db.DB, userID, vocabulary, true, excludeAudio, true)
			if err != nil {
				return nil, err
			}
			if exerciseType == enums.ExerciseTypeMatchPairs {
				return createRandomMatchPairsExercise(userID, options)
			}
			questionWord, language, answerLanguage, err = buildExerciseQuestionData(vocabulary, exerciseType)
			if err != nil {
				return nil, err
			}
		} else {
			description = generated.Description
			questionWord = ""
			language = vocabulary.Translation.Original.Language
			answerLanguage = vocabulary.Translation.Original.Language
		}
	}

	now := time.Now().UTC()
	exercise := models.Exercise{
		Type:      exerciseType,
		Status:    enums.ExerciseStatusInProgress,
		UserID:    userID,
		StartedAt: &now,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if isDescriptionExerciseType(exerciseType) {
			eligible, eligibilityErr := lockDescriptionLanguageEligibility(
				tx,
				userID,
				vocabulary.Translation.Original.Language,
			)
			if eligibilityErr != nil {
				return eligibilityErr
			}
			if !eligible {
				if len(requestedTypes) > 0 {
					remainingTypes := make([]enums.ExerciseType, 0, len(requestedTypes))
					for _, requestedType := range requestedTypes {
						if !isDescriptionExerciseType(requestedType) {
							remainingTypes = append(remainingTypes, requestedType)
						}
					}
					if len(remainingTypes) == 0 {
						return ErrNoVocabularyForExercise
					}
					exerciseType, options, err = selectRequestedExerciseTypeAndOptions(userID, vocabulary, remainingTypes)
				} else {
					exerciseType, options, err = selectExerciseTypeAndOptionsWithConfig(tx, userID, vocabulary, false, excludeAudio, true)
				}
				if err != nil {
					return err
				}
				questionWord, language, answerLanguage, err = buildExerciseQuestionData(vocabulary, exerciseType)
				if err != nil {
					return err
				}
				description = ""
			}
		}

		exercise.Type = exerciseType
		if err := tx.Create(&exercise).Error; err != nil {
			return err
		}

		return createExerciseVocabularyLinks(tx, exercise.ID, vocabularyID, options)
	})
	if err != nil {
		return nil, err
	}

	resultOptions := collectExerciseOptionLabels(options)
	if isCharacterExerciseType(exerciseType) && len(options) > 0 {
		resultOptions = ShuffledAnswerCharacters(options[0].AnswerWord)
	}

	var audioWordID *uuid.UUID
	if isAudioExerciseType(exerciseType) {
		wordID := vocabulary.Translation.Original.ID
		if isReversedExerciseType(exerciseType) {
			wordID = vocabulary.Translation.Translation.ID
		}
		audioWordID = &wordID
	}

	return &RandomExerciseResult{
		ExerciseID:     exercise.ID,
		Type:           exerciseType,
		QuestionWord:   questionWord,
		Language:       language,
		AnswerLanguage: answerLanguage,
		AudioWordID:    audioWordID,
		Options:        resultOptions,
		Description:    description,
	}, nil
}

func selectRequestedExerciseTypeAndOptions(userID uint, vocabulary *models.Vocabulary, requestedTypes []enums.ExerciseType) (enums.ExerciseType, []exerciseChoiceCandidate, error) {
	includeMatchPairs := false
	for _, exerciseType := range requestedTypes {
		if isMatchPairsExerciseType(exerciseType) {
			includeMatchPairs = true
			break
		}
	}

	availableTypes, err := buildExerciseOptionsByType(userID, vocabulary, includeMatchPairs)
	if err != nil {
		return "", nil, err
	}

	candidates := make([]enums.ExerciseType, 0, len(requestedTypes))
	for _, exerciseType := range requestedTypes {
		if isExerciseTypeAvailable(exerciseType, availableTypes[exerciseType]) {
			candidates = append(candidates, exerciseType)
		}
	}
	if len(candidates) == 0 {
		return "", nil, errNoExerciseTypeAvailable
	}

	exerciseType := candidates[rand.Intn(len(candidates))]
	return exerciseType, append([]exerciseChoiceCandidate(nil), availableTypes[exerciseType]...), nil
}

func createRandomMatchPairsExercise(userID uint, options []exerciseChoiceCandidate) (*RandomExerciseResult, error) {
	if len(options) != matchPairsVocabularyCount {
		return nil, errNoExerciseTypeAvailable
	}

	now := time.Now().UTC()
	exercise := models.Exercise{
		Type:      enums.ExerciseTypeMatchPairs,
		Status:    enums.ExerciseStatusInProgress,
		UserID:    userID,
		StartedAt: &now,
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

func loadExerciseVocabulary(vocabularyID uuid.UUID) (*models.Vocabulary, error) {
	return loadExerciseVocabularyWithDB(db.DB, vocabularyID)
}

func loadExerciseVocabularyWithDB(conn *gorm.DB, vocabularyID uuid.UUID) (*models.Vocabulary, error) {
	var vocabulary models.Vocabulary

	err := conn.
		Where("id = ?", vocabularyID).
		Where("deleted_at IS NULL").
		Preload("Translation").
		Preload("Translation.Original").
		Preload("Translation.Translation").
		Take(&vocabulary).Error
	if err != nil {
		return nil, err
	}

	return &vocabulary, nil
}

func selectExerciseTypeAndOptions(userID uint, vocabulary *models.Vocabulary, includeMatchPairs bool) (enums.ExerciseType, []exerciseChoiceCandidate, error) {
	return selectExerciseTypeAndOptionsWithDB(db.DB, userID, vocabulary, includeMatchPairs)
}

func selectExerciseTypeAndOptionsWithDB(conn *gorm.DB, userID uint, vocabulary *models.Vocabulary, includeMatchPairs bool) (enums.ExerciseType, []exerciseChoiceCandidate, error) {
	return selectExerciseTypeAndOptionsWithConfig(conn, userID, vocabulary, includeMatchPairs, false, false)
}

func selectExerciseTypeAndOptionsWithConfig(conn *gorm.DB, userID uint, vocabulary *models.Vocabulary, includeMatchPairs bool, excludeAudio, excludeDescription bool) (enums.ExerciseType, []exerciseChoiceCandidate, error) {
	availableTypes, err := buildExerciseOptionsByTypeWithDB(conn, userID, vocabulary, includeMatchPairs)
	if err != nil {
		return "", nil, err
	}

	type exerciseTypeGroup struct {
		weight int
		types  []enums.ExerciseType
	}

	groups := []exerciseTypeGroup{
		{
			weight: basicExerciseWeight,
			types: []enums.ExerciseType{
				enums.ExerciseTypeBasicDirect,
				enums.ExerciseTypeBasicReversed,
			},
		},
		{
			weight: choiceExerciseWeight,
			types: []enums.ExerciseType{
				enums.ExerciseTypeChoiceDirect,
				enums.ExerciseTypeChoiceReversed,
			},
		},
		{
			weight: characterExerciseWeight,
			types: []enums.ExerciseType{
				enums.ExerciseTypeCharactersDirect,
				enums.ExerciseTypeCharactersReversed,
			},
		},
	}
	if !excludeAudio {
		groups = append(groups, exerciseTypeGroup{
			weight: audioExerciseWeight,
			types: []enums.ExerciseType{
				enums.ExerciseTypeAudioDirect,
				enums.ExerciseTypeAudioReversed,
			},
		})
	}
	if !excludeDescription {
		groups = append(groups, exerciseTypeGroup{
			weight: descriptionExerciseWeight,
			types:  []enums.ExerciseType{enums.ExerciseTypeDescriptionReversed},
		})
	}
	if includeMatchPairs {
		groups = append(groups, exerciseTypeGroup{
			weight: matchPairsExerciseWeight,
			types:  []enums.ExerciseType{enums.ExerciseTypeMatchPairs},
		})
	}

	availableGroups := make([]exerciseTypeGroup, 0, len(groups))
	totalWeight := 0
	for _, group := range groups {
		availableTypesInGroup := make([]enums.ExerciseType, 0, len(group.types))
		for _, exerciseType := range group.types {
			options := availableTypes[exerciseType]
			if isExerciseTypeAvailable(exerciseType, options) {
				availableTypesInGroup = append(availableTypesInGroup, exerciseType)
			}
		}
		if len(availableTypesInGroup) > 0 {
			group.types = availableTypesInGroup
			availableGroups = append(availableGroups, group)
			totalWeight += group.weight
		}
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
	return exerciseType, append([]exerciseChoiceCandidate(nil), availableTypes[exerciseType]...), nil
}

func isExerciseTypeAvailable(exerciseType enums.ExerciseType, options []exerciseChoiceCandidate) bool {
	return (isChoiceExerciseType(exerciseType) && len(options) == choiceExerciseVocabularyCount) ||
		(isMatchPairsExerciseType(exerciseType) && len(options) == matchPairsVocabularyCount) ||
		(!isChoiceExerciseType(exerciseType) && !isMatchPairsExerciseType(exerciseType) && len(options) > 0)
}

func buildExerciseOptionsByType(userID uint, vocabulary *models.Vocabulary, includeMatchPairs bool) (map[enums.ExerciseType][]exerciseChoiceCandidate, error) {
	return buildExerciseOptionsByTypeWithDB(db.DB, userID, vocabulary, includeMatchPairs)
}

func buildExerciseOptionsByTypeWithDB(conn *gorm.DB, userID uint, vocabulary *models.Vocabulary, includeMatchPairs bool) (map[enums.ExerciseType][]exerciseChoiceCandidate, error) {
	optionsByType := make(map[enums.ExerciseType][]exerciseChoiceCandidate, 10)

	directOptions, err := buildExerciseOptionsForTypeWithDB(conn, userID, vocabulary, enums.ExerciseTypeBasicDirect)
	if err != nil {
		return nil, err
	}
	if len(directOptions) > 0 {
		optionsByType[enums.ExerciseTypeBasicDirect] = append([]exerciseChoiceCandidate(nil), directOptions[:1]...)
		if !ignoredAudioLanguageWithDB(conn, userID, vocabulary.Translation.Original.Language) {
			optionsByType[enums.ExerciseTypeAudioDirect] = append([]exerciseChoiceCandidate(nil), directOptions[:1]...)
		}
		if strings.TrimSpace(directOptions[0].AnswerWord) != "" {
			optionsByType[enums.ExerciseTypeCharactersDirect] = append([]exerciseChoiceCandidate(nil), directOptions[:1]...)
		}
	}
	if len(directOptions) == choiceExerciseVocabularyCount {
		optionsByType[enums.ExerciseTypeChoiceDirect] = shuffledExerciseOptions(directOptions)
	}

	reversedOptions, err := buildExerciseOptionsForTypeWithDB(conn, userID, vocabulary, enums.ExerciseTypeBasicReversed)
	if err != nil {
		return nil, err
	}
	if len(reversedOptions) > 0 {
		optionsByType[enums.ExerciseTypeBasicReversed] = append([]exerciseChoiceCandidate(nil), reversedOptions[:1]...)
		if !ignoredAudioLanguageWithDB(conn, userID, vocabulary.Translation.Translation.Language) {
			optionsByType[enums.ExerciseTypeAudioReversed] = append([]exerciseChoiceCandidate(nil), reversedOptions[:1]...)
		}
		if strings.TrimSpace(reversedOptions[0].AnswerWord) != "" {
			optionsByType[enums.ExerciseTypeCharactersReversed] = append([]exerciseChoiceCandidate(nil), reversedOptions[:1]...)
		}
		if descriptionLanguageEligibleWithDB(conn, userID, vocabulary.Translation.Original.Language) {
			optionsByType[enums.ExerciseTypeDescriptionReversed] = append([]exerciseChoiceCandidate(nil), reversedOptions[:1]...)
		}
	}
	if len(reversedOptions) == choiceExerciseVocabularyCount {
		optionsByType[enums.ExerciseTypeChoiceReversed] = shuffledExerciseOptions(reversedOptions)
	}

	if includeMatchPairs {
		matchOptions, err := buildMatchPairOptionsWithDB(conn, userID, vocabulary)
		if err != nil {
			return nil, err
		}
		if len(matchOptions) == matchPairsVocabularyCount {
			optionsByType[enums.ExerciseTypeMatchPairs] = shuffledExerciseOptions(matchOptions)
		}
	}

	return optionsByType, nil
}

func descriptionLanguageEligibleWithDB(conn *gorm.DB, userID uint, language enums.Language) bool {
	var user models.User
	if err := conn.Select("settings").Where("id = ?", userID).Take(&user).Error; err != nil {
		return false
	}
	return user.Settings.MainLearningLanguage == language &&
		!containsLanguage(user.Settings.IgnoredDescriptionLanguages, language)
}

func ignoredAudioLanguageWithDB(conn *gorm.DB, userID uint, language enums.Language) bool {
	var user models.User
	if err := conn.Select("settings").Where("id = ?", userID).Take(&user).Error; err != nil {
		return true
	}
	for _, ignored := range user.Settings.IgnoredAudioLanguages {
		if ignored == language {
			return true
		}
	}
	return false
}

func buildMatchPairOptions(userID uint, vocabulary *models.Vocabulary) ([]exerciseChoiceCandidate, error) {
	return buildMatchPairOptionsWithDB(db.DB, userID, vocabulary)
}

func buildMatchPairOptionsWithDB(conn *gorm.DB, userID uint, vocabulary *models.Vocabulary) ([]exerciseChoiceCandidate, error) {
	if vocabulary == nil || vocabulary.Translation == nil {
		return nil, errors.New("vocabulary has no translation")
	}

	translation := vocabulary.Translation
	if translation.Original == nil {
		return nil, errors.New("vocabulary has no original word")
	}
	if translation.Translation == nil {
		return nil, errors.New("vocabulary has no translation word")
	}

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
			AND v.mastered_at IS NULL
			AND original.language = ?
			AND translated.language = ?
			AND EXISTS (
				SELECT 1
				FROM jsonb_array_elements(v.progress) AS p
				WHERE p->>'type' = ? AND (p->>'knowledge')::int < ?
			)
		ORDER BY RANDOM()
	`

	var rows []exerciseMatchPairCandidate
	if err := conn.Raw(
		query,
		userID,
		translation.Original.Language,
		translation.Translation.Language,
		enums.KnowledgeTypeTranslation,
		100,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]exerciseMatchPairCandidate, len(rows))
	for _, row := range rows {
		byID[row.VocabularyID] = row
	}

	selected, ok := byID[vocabulary.ID]
	if !ok {
		return []exerciseChoiceCandidate{}, nil
	}

	seenOriginal := map[string]struct{}{
		normalizeAnswer(selected.OriginalWord): {},
	}
	seenTranslation := map[string]struct{}{
		normalizeAnswer(selected.TranslationWord): {},
	}
	selectedRows := []exerciseMatchPairCandidate{selected}

	for _, row := range rows {
		if row.VocabularyID == vocabulary.ID {
			continue
		}

		normalizedOriginal := normalizeAnswer(row.OriginalWord)
		normalizedTranslation := normalizeAnswer(row.TranslationWord)
		if strings.TrimSpace(normalizedOriginal) == "" || strings.TrimSpace(normalizedTranslation) == "" {
			continue
		}
		if _, exists := seenOriginal[normalizedOriginal]; exists {
			continue
		}
		if _, exists := seenTranslation[normalizedTranslation]; exists {
			continue
		}

		seenOriginal[normalizedOriginal] = struct{}{}
		seenTranslation[normalizedTranslation] = struct{}{}
		selectedRows = append(selectedRows, row)
		if len(selectedRows) == matchPairsVocabularyCount {
			break
		}
	}

	options := make([]exerciseChoiceCandidate, 0, len(selectedRows))
	for _, row := range selectedRows {
		options = append(options, exerciseChoiceCandidate{VocabularyID: row.VocabularyID})
	}

	return options, nil
}

func buildExerciseOptionsForType(userID uint, vocabulary *models.Vocabulary, exerciseType enums.ExerciseType) ([]exerciseChoiceCandidate, error) {
	return buildExerciseOptionsForTypeWithDB(db.DB, userID, vocabulary, exerciseType)
}

func buildExerciseOptionsForTypeWithDB(conn *gorm.DB, userID uint, vocabulary *models.Vocabulary, exerciseType enums.ExerciseType) ([]exerciseChoiceCandidate, error) {
	if vocabulary == nil || vocabulary.Translation == nil {
		return nil, errors.New("vocabulary has no translation")
	}

	translation := vocabulary.Translation
	if translation.Original == nil {
		return nil, errors.New("vocabulary has no original word")
	}
	if translation.Translation == nil {
		return nil, errors.New("vocabulary has no translation word")
	}

	answerOption := exerciseChoiceCandidate{
		VocabularyID: vocabulary.ID,
		AnswerWord:   translation.Translation.Word,
	}
	queryColumn := "translated.word"

	if isReversedExerciseType(exerciseType) {
		answerOption.AnswerWord = translation.Original.Word
		queryColumn = "original.word"
	}

	distractors, err := getExerciseDistractorWordsWithDB(
		conn,
		userID,
		translation.Original.Language,
		translation.Translation.Language,
		queryColumn,
		vocabulary.ID,
		answerOption.AnswerWord,
	)
	if err != nil {
		return nil, err
	}

	requiredDistractors := choiceExerciseVocabularyCount - 1
	if len(distractors) < requiredDistractors {
		return []exerciseChoiceCandidate{answerOption}, nil
	}

	options := make([]exerciseChoiceCandidate, 0, choiceExerciseVocabularyCount)
	options = append(options, answerOption)
	options = append(options, distractors[:requiredDistractors]...)

	return options, nil
}

func shuffledExerciseOptions(options []exerciseChoiceCandidate) []exerciseChoiceCandidate {
	shuffled := append([]exerciseChoiceCandidate(nil), options...)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled
}

func getExerciseDistractorWords(
	userID uint,
	originalLanguage enums.Language,
	translationLanguage enums.Language,
	answerColumn string,
	correctVocabularyID uuid.UUID,
	correctAnswer string,
) ([]exerciseChoiceCandidate, error) {
	return getExerciseDistractorWordsWithDB(db.DB, userID, originalLanguage, translationLanguage, answerColumn, correctVocabularyID, correctAnswer)
}

func getExerciseDistractorWordsWithDB(
	conn *gorm.DB,
	userID uint,
	originalLanguage enums.Language,
	translationLanguage enums.Language,
	answerColumn string,
	correctVocabularyID uuid.UUID,
	correctAnswer string,
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
	`

	var rows []exerciseChoiceCandidate
	if err := conn.Raw(query, userID, originalLanguage, translationLanguage, correctVocabularyID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	seen := map[string]struct{}{
		normalizeAnswer(correctAnswer): {},
	}
	options := make([]exerciseChoiceCandidate, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.AnswerWord) == "" {
			continue
		}

		normalized := normalizeAnswer(row.AnswerWord)
		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		options = append(options, row)
	}

	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	return options, nil
}

func buildExerciseQuestionData(vocabulary *models.Vocabulary, exerciseType enums.ExerciseType) (string, enums.Language, enums.Language, error) {
	if vocabulary == nil || vocabulary.Translation == nil {
		return "", "", "", errors.New("vocabulary has no translation")
	}

	translation := vocabulary.Translation
	if translation.Original == nil {
		return "", "", "", errors.New("vocabulary has no original word")
	}
	if translation.Translation == nil {
		return "", "", "", errors.New("vocabulary has no translation word")
	}

	if isReversedExerciseType(exerciseType) {
		return translation.Translation.Word, translation.Translation.Language, translation.Original.Language, nil
	}

	return translation.Original.Word, translation.Original.Language, translation.Translation.Language, nil
}
