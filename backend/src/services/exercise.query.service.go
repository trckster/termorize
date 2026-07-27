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

func GetExerciseWordsByTelegram(exerciseID uuid.UUID, telegramID int64) (*ExerciseWords, error) {
	var words ExerciseWords

	err := db.DB.Raw(`
		SELECT
			e.type AS exercise_type,
			original.word AS original_word,
			original.language AS original_language,
			translated.word AS translation_word,
			translated.language AS translation_language
		FROM exercises AS e
		JOIN users AS u ON u.id = e.user_id
		JOIN vocabulary_exercises AS ve ON ve.exercise_id = e.id AND ve.is_correct = true
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE e.id = ?
			AND u.telegram_id = ?
		LIMIT 1
	`, exerciseID, telegramID).Scan(&words).Error

	if err != nil {
		return nil, err
	}

	if words.OriginalWord == "" && words.TranslationWord == "" {
		return nil, nil
	}

	return &words, nil
}

func GetExerciseStatistics(userID uint) (*ExerciseStatistics, error) {
	var statistics ExerciseStatistics

	err := db.DB.Raw(`
		SELECT
			COUNT(*) FILTER (WHERE status = ?) AS in_progress,
			COUNT(*) FILTER (WHERE status = ?) AS done,
			COUNT(*) FILTER (WHERE status = ?) AS failed,
			COUNT(*) FILTER (WHERE status = ?) AS ignored
		FROM exercises
		WHERE user_id = ?
	`, enums.ExerciseStatusInProgress, enums.ExerciseStatusCompleted, enums.ExerciseStatusFailed, enums.ExerciseStatusIgnored, userID).Scan(&statistics).Error
	if err != nil {
		return nil, err
	}

	location := time.UTC
	var user models.User
	if err := db.DB.Select("settings").First(&user, userID).Error; err != nil {
		return nil, err
	}
	if user.Settings.TimeZone != "" {
		if configuredLocation, locationErr := time.LoadLocation(user.Settings.TimeZone); locationErr == nil {
			location = configuredLocation
		}
	}

	now := time.Now().In(location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	weekStart := today.AddDate(0, 0, -7)
	activityStart := time.Date(today.Year(), today.Month()-5, 1, 0, 0, 0, 0, location)
	rangeEnd := today.AddDate(0, 0, 1)

	type exerciseActivityRow struct {
		Status     enums.ExerciseStatus
		FinishedAt time.Time
	}
	var exerciseRows []exerciseActivityRow
	if err := db.DB.Model(&models.Exercise{}).
		Select("status, finished_at").
		Where("user_id = ?", userID).
		Where("status IN ?", []enums.ExerciseStatus{enums.ExerciseStatusCompleted, enums.ExerciseStatusFailed}).
		Where("finished_at >= ? AND finished_at < ?", weekStart.UTC(), rangeEnd.UTC()).
		Find(&exerciseRows).Error; err != nil {
		return nil, err
	}

	exerciseByDate := make(map[string]int, 8)
	statistics.ExerciseActivity = make([]ExerciseDailyActivity, 0, 8)
	for day := weekStart; !day.After(today); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		statistics.ExerciseActivity = append(statistics.ExerciseActivity, ExerciseDailyActivity{Date: date})
		exerciseByDate[date] = len(statistics.ExerciseActivity) - 1
	}
	for _, row := range exerciseRows {
		index, ok := exerciseByDate[row.FinishedAt.In(location).Format("2006-01-02")]
		if !ok {
			continue
		}
		if row.Status == enums.ExerciseStatusCompleted {
			statistics.ExerciseActivity[index].Completed++
		} else {
			statistics.ExerciseActivity[index].Failed++
		}
	}

	type vocabularyActivityRow struct {
		CreatedAt time.Time
	}
	var vocabularyRows []vocabularyActivityRow
	if err := db.DB.Raw(`
		SELECT created_at
		FROM vocabulary
		WHERE user_id = ? AND created_at >= ? AND created_at < ?
	`, userID, activityStart.UTC(), rangeEnd.UTC()).Scan(&vocabularyRows).Error; err != nil {
		return nil, err
	}

	vocabularyByDate := make(map[string]int)
	statistics.VocabularyActivity = make([]VocabularyDailyActivity, 0, 186)
	for day := activityStart; !day.After(today); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		statistics.VocabularyActivity = append(statistics.VocabularyActivity, VocabularyDailyActivity{Date: date})
		vocabularyByDate[date] = len(statistics.VocabularyActivity) - 1
	}
	for _, row := range vocabularyRows {
		index, ok := vocabularyByDate[row.CreatedAt.In(location).Format("2006-01-02")]
		if ok {
			statistics.VocabularyActivity[index].Count++
		}
	}

	return &statistics, nil
}

func GetExercises(userID uint, page, pageSize int) (*ExerciseListResponse, error) {
	if page <= 0 {
		return nil, ErrInvalidPage
	}

	if pageSize < 1 || pageSize > 1000 {
		return nil, ErrInvalidPageSize
	}

	totalQuery := db.DB.Model(&models.Exercise{}).
		Where("user_id = ?", userID).
		Where("started_at IS NOT NULL")

	var total int64
	if err := totalQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	exercises := make([]models.Exercise, 0, pageSize)

	if err := db.DB.
		Model(&models.Exercise{}).
		Where("user_id = ?", userID).
		Where("started_at IS NOT NULL").
		Order("started_at DESC, id DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&exercises).Error; err != nil {
		return nil, err
	}

	vocabularyByExerciseID, err := loadListVocabularyByExerciseIDs(collectExerciseIDs(exercises))
	if err != nil {
		return nil, err
	}

	data := make([]ExerciseListExercise, 0, len(exercises))
	for _, exerciseModel := range exercises {
		exercise := ExerciseListExercise{
			ID:                exerciseModel.ID,
			Type:              exerciseModel.Type,
			Status:            exerciseModel.Status,
			StartedAt:         exerciseModel.StartedAt,
			FinishedAt:        exerciseModel.FinishedAt,
			TelegramMessageID: exerciseModel.TelegramMessageID,
		}

		if vocabulary, ok := vocabularyByExerciseID[exerciseModel.ID]; ok {
			exercise.Vocabulary = vocabulary
			if len(vocabulary) > 0 {
				legacyVocabulary := vocabulary[0]
				exercise.LegacyVocabulary = &legacyVocabulary
			}
		}

		data = append(data, exercise)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return &ExerciseListResponse{
		Data: data,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func GetExercisesByIDs(userID uint, ids []uuid.UUID) ([]ExerciseListExercise, error) {
	if len(ids) == 0 {
		return []ExerciseListExercise{}, nil
	}

	exercises := make([]models.Exercise, 0, len(ids))

	if err := db.DB.
		Where("user_id = ? AND id IN ?", userID, ids).
		Find(&exercises).Error; err != nil {
		return nil, err
	}

	vocabularyByExerciseID, err := loadListVocabularyByExerciseIDs(collectExerciseIDs(exercises))
	if err != nil {
		return nil, err
	}

	data := make([]ExerciseListExercise, 0, len(exercises))
	for _, exercise := range exercises {
		response := ExerciseListExercise{
			ID:                exercise.ID,
			Type:              exercise.Type,
			Status:            exercise.Status,
			StartedAt:         exercise.StartedAt,
			FinishedAt:        exercise.FinishedAt,
			TelegramMessageID: exercise.TelegramMessageID,
		}

		if vocabulary, ok := vocabularyByExerciseID[exercise.ID]; ok {
			response.Vocabulary = vocabulary
			if len(vocabulary) > 0 {
				legacyVocabulary := vocabulary[0]
				response.LegacyVocabulary = &legacyVocabulary
			}
		}

		data = append(data, response)
	}

	return data, nil
}

func createExerciseVocabularyLinks(tx *gorm.DB, exerciseID uuid.UUID, correctVocabularyID uuid.UUID, options []exerciseChoiceCandidate) error {
	links := make([]map[string]any, 0, len(options))
	for index, option := range options {
		links = append(links, map[string]any{
			"exercise_id":   exerciseID,
			"vocabulary_id": option.VocabularyID,
			"is_correct":    correctVocabularyID == uuid.Nil || option.VocabularyID == correctVocabularyID,
			"position":      index,
		})
	}

	return tx.Table("vocabulary_exercises").Create(&links).Error
}

func GetExerciseMatchCards(exerciseID uuid.UUID) ([]ExerciseMatchCard, error) {
	rows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, true)
	if err != nil {
		return nil, err
	}

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

	rand.Shuffle(len(cards), func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})

	return cards, nil
}

func GetExerciseAnswerOptions(exerciseID uuid.UUID, exerciseType enums.ExerciseType) ([]ExerciseOption, error) {
	answerColumn := "translated.word"
	if isReversedExerciseType(exerciseType) {
		answerColumn = "original.word"
	}

	query := `
		SELECT
			v.id AS vocabulary_id,
			` + answerColumn + ` AS answer_word
		FROM vocabulary_exercises AS ve
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id AND v.deleted_at IS NULL
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE ve.exercise_id = ?
	`

	var rows []exerciseChoiceCandidate
	if err := db.DB.Raw(query, exerciseID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	options := make([]ExerciseOption, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.AnswerWord) == "" {
			continue
		}

		options = append(options, ExerciseOption{
			VocabularyID: row.VocabularyID,
			Label:        row.AnswerWord,
		})
	}

	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	return options, nil
}

func getExerciseWithCorrectVocabulary(exerciseID uuid.UUID, userID uint) (*models.Exercise, *exerciseVocabularyDetails, error) {
	var exercise models.Exercise

	err := db.DB.
		Where("id = ? AND user_id = ?", exerciseID, userID).
		Take(&exercise).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrExerciseNotFound
		}

		return nil, nil, err
	}

	correctVocabulary, err := getCorrectExerciseVocabularyDetails(exercise.ID)
	if err != nil {
		return nil, nil, err
	}

	return &exercise, correctVocabulary, nil
}

func getCorrectExerciseVocabularyDetails(exerciseID uuid.UUID) (*exerciseVocabularyDetails, error) {
	rows, err := getExerciseVocabularyDetails([]uuid.UUID{exerciseID}, true, true)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		return &row, nil
	}

	return nil, nil
}

func loadListVocabularyByExerciseIDs(exerciseIDs []uuid.UUID) (map[uuid.UUID][]ExerciseListVocabulary, error) {
	rows, err := getExerciseVocabularyDetails(exerciseIDs, false, false)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]ExerciseListVocabulary, len(rows))
	for _, row := range rows {
		result[row.ExerciseID] = append(result[row.ExerciseID], buildListVocabularyFromExerciseDetails(row))
	}

	return result, nil
}

func getExerciseVocabularyDetails(exerciseIDs []uuid.UUID, onlyCorrect bool, onlyActive bool) ([]exerciseVocabularyDetails, error) {
	if len(exerciseIDs) == 0 {
		return []exerciseVocabularyDetails{}, nil
	}

	query := `
		SELECT
			ve.exercise_id AS exercise_id,
			ve.vocabulary_id AS vocabulary_id,
			ve.is_correct AS is_correct,
			ve.position AS position,
			ve.result AS result,
			ve.result_reason AS result_reason,
			ve.progress_delta AS progress_delta,
			ve.knowledge_after AS knowledge_after,
			ve.answered_at AS answered_at,
			v.deleted_at AS vocabulary_deleted_at,
			original.word AS original_word,
			original.language AS original_language,
			translated.word AS translation_word,
			translated.language AS translation_language
		FROM vocabulary_exercises AS ve
		JOIN vocabulary AS v ON v.id = ve.vocabulary_id
		JOIN translations AS t ON t.id = v.translation_id
		JOIN words AS original ON original.id = t.original_id
		JOIN words AS translated ON translated.id = t.translation_id
		WHERE ve.exercise_id IN ?
	`
	args := []any{exerciseIDs}
	if onlyCorrect {
		query += ` AND ve.is_correct = ?`
		args = append(args, true)
	}
	if onlyActive {
		query += ` AND v.deleted_at IS NULL`
	} else {
		query += ` AND (ve.is_correct = true OR ve.result IS NOT NULL)`
	}

	query += ` ORDER BY ve.exercise_id, ve.position ASC, ve.id ASC`

	var rows []exerciseVocabularyDetails
	if err := db.DB.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

func buildListVocabularyFromExerciseDetails(details exerciseVocabularyDetails) ExerciseListVocabulary {
	vocabulary := buildVocabularyFromExerciseDetails(details)
	translation := vocabulary.Translation
	if details.VocabularyDeletedAt != nil {
		translation = nil
	}

	return ExerciseListVocabulary{
		ID:             details.VocabularyID,
		Translation:    translation,
		ExerciseResult: details.Result,
		ResultReason:   details.ResultReason,
		ProgressDelta:  details.ProgressDelta,
		KnowledgeAfter: details.KnowledgeAfter,
		AnsweredAt:     details.AnsweredAt,
		IsCorrect:      details.IsCorrect,
		Position:       details.Position,
	}
}

func buildVocabularyFromExerciseDetails(details exerciseVocabularyDetails) models.Vocabulary {
	return models.Vocabulary{
		ID: details.VocabularyID,
		Translation: &models.Translation{
			Original: &models.Word{
				Word:     details.OriginalWord,
				Language: details.OriginalLanguage,
			},
			Translation: &models.Word{
				Word:     details.TranslationWord,
				Language: details.TranslationLanguage,
			},
		},
	}
}

func collectExerciseIDs(exercises []models.Exercise) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(exercises))
	for _, exercise := range exercises {
		ids = append(ids, exercise.ID)
	}

	return ids
}

func collectExerciseOptionLabels(options []exerciseChoiceCandidate) []string {
	labels := make([]string, 0, len(options))
	for _, option := range options {
		labels = append(labels, option.AnswerWord)
	}

	return labels
}

func exerciseOptionsContainAnswer(options []ExerciseOption, normalizedAnswer string) bool {
	for _, option := range options {
		if normalizeAnswer(option.Label) == normalizedAnswer {
			return true
		}
	}

	return false
}
