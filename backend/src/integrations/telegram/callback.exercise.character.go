package telegram

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/services"

	"github.com/google/uuid"
)

func parseExerciseCharacterPayload(payload []string) (uuid.UUID, int, bool) {
	if len(payload) != 3 || payload[0] != exerciseActionCharacterTap {
		return uuid.Nil, 0, false
	}

	exerciseID, err := parseCallbackUUID(payload[1])
	if err != nil {
		return uuid.Nil, 0, false
	}

	tappedIndex, err := strconv.Atoi(payload[2])
	if err != nil || tappedIndex < 0 || tappedIndex > 1024 {
		return uuid.Nil, 0, false
	}

	return exerciseID, tappedIndex, true
}

func parseExerciseCharacterBackspacePayload(payload []string) (uuid.UUID, bool) {
	if len(payload) != 2 || payload[0] != exerciseActionCharacterBackspace {
		return uuid.Nil, false
	}

	exerciseID, err := parseCallbackUUID(payload[1])
	if err != nil {
		return uuid.Nil, false
	}
	return exerciseID, true
}

func handleCharacterTap(callback *callbackQuery, payload []string, t BotTexts) error {
	if callback.Message == nil {
		return nil
	}

	exerciseID, tappedIndex, ok := parseExerciseCharacterPayload(payload)
	if !ok {
		return nil
	}

	exercise, err := services.GetExerciseByTelegramMessage(callback.Message.MessageID, callback.From.ID)
	if err != nil {
		return err
	}
	if exercise == nil {
		exercise, err = recoverPendingCharacterExerciseFromCallback(callback, exerciseID)
		if err != nil {
			return err
		}
	}
	if exercise == nil || exercise.ExerciseID != exerciseID {
		return nil
	}

	questionText := BuildBasicExerciseQuestion(
		exercise.OriginalWord,
		exercise.TranslationWord,
		exercise.OriginalLanguage,
		exercise.TranslationLanguage,
		exercise.ExerciseType,
		t,
	)

	switch exercise.Status {
	case enums.ExerciseStatusIgnored:
		return sendIgnoredExerciseMessage(callback.Message.Chat.ID, callback.Message.MessageID, callback.From.ID, exercise, t)
	case enums.ExerciseStatusCompleted, enums.ExerciseStatusFailed:
		board := completedCharacterBoard(exercise)
		return EditCharacterBoardMessage(
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			buildCharacterBoardText(questionText, board),
			[][]inlineKeyboardButton{},
		)
	}

	if len(exercise.Vocabulary) == 0 || exercise.Vocabulary[0].Translation == nil {
		_ = services.MarkExerciseVocabularyResultWithoutProgress(exercise.ExerciseID, services.ExerciseVocabularyResultIgnored, services.ExerciseVocabularyResultReasonDeletedVocabulary)
		_ = services.IgnoreExercise(exercise.ExerciseID)
		return sendDeletedVocabularyMessage(callback.Message.Chat.ID, callback.Message.MessageID, callback.From.ID, t.ExerciseVocabularyDeleted)
	}

	board, finished, err := services.ApplyCharacterTap(exercise.ExerciseID, exercise.UserID, tappedIndex)
	if err != nil {
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil
		}
		if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
			return sendDeletedVocabularyMessage(callback.Message.Chat.ID, callback.Message.MessageID, callback.From.ID, t.ExerciseVocabularyDeleted)
		}
		return err
	}

	if !finished {
		return EditCharacterBoardMessage(
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			buildCharacterBoardText(questionText, board),
			buildCharacterKeyboard(exercise.ExerciseID, board, t),
		)
	}

	result, err := services.VerifyExerciseAnswer(exercise.ExerciseID, exercise.UserID, board.Answer)
	if err != nil {
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil
		}
		if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
			return sendDeletedVocabularyMessage(callback.Message.Chat.ID, callback.Message.MessageID, callback.From.ID, t.ExerciseVocabularyDeleted)
		}
		return err
	}

	if err := EditCharacterBoardMessage(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		buildCharacterBoardText(questionText, board),
		[][]inlineKeyboardButton{},
	); err != nil {
		logger.L().Warnw("failed to finalize character exercise board", "error", err, "exercise_id", exercise.ExerciseID)
	}

	switch result.Result {
	case "correct":
		return SendMessageMarkdown(callback.From.ID, buildExerciseSuccessResultText(result.Knowledge, t))
	case "almost":
		return SendMessageMarkdown(callback.From.ID, buildExerciseAlmostResultText(
			exercise.OriginalWord,
			exercise.TranslationWord,
			exercise.OriginalLanguage,
			exercise.TranslationLanguage,
			result.Knowledge,
			t,
		))
	default:
		return SendMessageMarkdown(callback.From.ID, buildExerciseInvalidResultText(
			exercise.OriginalWord,
			exercise.TranslationWord,
			exercise.OriginalLanguage,
			exercise.TranslationLanguage,
			result.Knowledge,
			t,
		))
	}
}

func handleCharacterBackspace(callback *callbackQuery, payload []string, t BotTexts) error {
	if callback.Message == nil {
		return nil
	}

	exerciseID, ok := parseExerciseCharacterBackspacePayload(payload)
	if !ok {
		return nil
	}

	exercise, err := services.GetExerciseByTelegramMessage(callback.Message.MessageID, callback.From.ID)
	if err != nil {
		return err
	}
	if exercise == nil {
		exercise, err = recoverPendingCharacterExerciseFromCallback(callback, exerciseID)
		if err != nil {
			return err
		}
	}
	if exercise == nil || exercise.ExerciseID != exerciseID {
		return nil
	}

	switch exercise.Status {
	case enums.ExerciseStatusIgnored:
		return sendIgnoredExerciseMessage(callback.Message.Chat.ID, callback.Message.MessageID, callback.From.ID, exercise, t)
	case enums.ExerciseStatusCompleted, enums.ExerciseStatusFailed:
		questionText := BuildBasicExerciseQuestion(
			exercise.OriginalWord,
			exercise.TranslationWord,
			exercise.OriginalLanguage,
			exercise.TranslationLanguage,
			exercise.ExerciseType,
			t,
		)
		board := completedCharacterBoard(exercise)
		return EditCharacterBoardMessage(
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			buildCharacterBoardText(questionText, board),
			[][]inlineKeyboardButton{},
		)
	}

	board, err := services.RemoveLastCharacterSelection(exercise.ExerciseID, exercise.UserID)
	if err != nil {
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil
		}
		if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
			return sendDeletedVocabularyMessage(callback.Message.Chat.ID, callback.Message.MessageID, callback.From.ID, t.ExerciseVocabularyDeleted)
		}
		return err
	}

	questionText := BuildBasicExerciseQuestion(
		exercise.OriginalWord,
		exercise.TranslationWord,
		exercise.OriginalLanguage,
		exercise.TranslationLanguage,
		exercise.ExerciseType,
		t,
	)
	return EditCharacterBoardMessage(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		buildCharacterBoardText(questionText, board),
		buildCharacterKeyboard(exercise.ExerciseID, board, t),
	)
}

func recoverPendingCharacterExerciseFromCallback(callback *callbackQuery, exerciseID uuid.UUID) (*services.TelegramMessageExercise, error) {
	exercise, err := services.GetExerciseByTelegramExerciseID(exerciseID, callback.From.ID)
	if err != nil {
		return nil, err
	}
	if exercise == nil ||
		exercise.Status != enums.ExerciseStatusPending ||
		(exercise.ExerciseType != enums.ExerciseTypeCharactersDirect && exercise.ExerciseType != enums.ExerciseTypeCharactersReversed) {
		return nil, nil
	}

	answer := characterExerciseAnswer(exercise)
	order, ok := extractCharacterOrderFromReplyMarkup(callback.Message.ReplyMarkup, exerciseID, len(services.AnswerCharacters(answer)))
	if !ok {
		return nil, nil
	}

	if err := services.StartCharacterExercise(exerciseID, callback.Message.MessageID, order); err != nil {
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil, nil
		}
		return nil, err
	}

	return services.GetExerciseByTelegramMessage(callback.Message.MessageID, callback.From.ID)
}

func extractCharacterOrderFromReplyMarkup(markup *inlineKeyboardMarkup, exerciseID uuid.UUID, characterCount int) ([]int, bool) {
	if markup == nil || characterCount == 0 {
		return nil, false
	}

	if order, ok := extractCurrentCharacterOrder(markup.InlineKeyboard, exerciseID, characterCount); ok {
		return order, true
	}

	return extractLegacyCharacterOrder(markup.InlineKeyboard, exerciseID, characterCount)
}

func extractCurrentCharacterOrder(rows [][]inlineKeyboardButton, exerciseID uuid.UUID, characterCount int) ([]int, bool) {
	expectedSide := int(math.Ceil(math.Sqrt(float64(characterCount))))
	if len(rows) != expectedSide+1 {
		return nil, false
	}
	actionRow := rows[expectedSide]
	if len(actionRow) != 2 {
		return nil, false
	}

	order, ok := extractCharacterGridOrder(rows[:expectedSide], expectedSide, false, exerciseID, characterCount)
	if !ok {
		return nil, false
	}

	backspaceExerciseID, ok := parseExerciseCharacterBackspacePayloadFromData(actionRow[0].CallbackData)
	if !ok || backspaceExerciseID != exerciseID {
		return nil, false
	}
	idkExerciseID, ok := parseExerciseIDKPayloadFromData(actionRow[1].CallbackData)
	if !ok || idkExerciseID != exerciseID {
		return nil, false
	}

	return order, true
}

func extractLegacyCharacterOrder(rows [][]inlineKeyboardButton, exerciseID uuid.UUID, characterCount int) ([]int, bool) {
	expectedSide := int(math.Ceil(math.Sqrt(float64(characterCount + 1))))
	if len(rows) != expectedSide {
		return nil, false
	}

	gridRows := make([][]inlineKeyboardButton, len(rows))
	for rowIndex, row := range rows {
		gridRows[rowIndex] = append([]inlineKeyboardButton(nil), row...)
	}
	if len(gridRows[expectedSide-1]) != expectedSide {
		return nil, false
	}
	lastColumn := expectedSide - 1
	backspaceExerciseID, ok := parseExerciseCharacterBackspacePayloadFromData(gridRows[expectedSide-1][lastColumn].CallbackData)
	if !ok || backspaceExerciseID != exerciseID {
		return nil, false
	}
	gridRows[expectedSide-1] = gridRows[expectedSide-1][:lastColumn]

	order, ok := extractCharacterGridOrder(gridRows, expectedSide, true, exerciseID, characterCount)
	if !ok || len(order) != expectedSide*expectedSide-1 {
		return nil, false
	}
	return order, true
}

func extractCharacterGridOrder(rows [][]inlineKeyboardButton, side int, shortLastRow bool, exerciseID uuid.UUID, characterCount int) ([]int, bool) {
	order := make([]int, 0, side*side)
	seen := make(map[int]bool, characterCount)
	for rowIndex, row := range rows {
		expectedWidth := side
		if shortLastRow && rowIndex == side-1 {
			expectedWidth--
		}
		if len(row) != expectedWidth {
			return nil, false
		}
		for _, button := range row {
			handlerType, payload, ok := parseCallbackData(button.CallbackData)
			if !ok || handlerType != callbackTypeExercise || len(payload) == 0 {
				return nil, false
			}
			if payload[0] == exerciseActionCharacterNoop {
				order = append(order, -1)
				continue
			}
			buttonExerciseID, canonical, ok := parseExerciseCharacterPayload(payload)
			if !ok || buttonExerciseID != exerciseID || canonical >= characterCount || seen[canonical] {
				return nil, false
			}
			seen[canonical] = true
			order = append(order, canonical)
		}
	}

	if len(seen) != characterCount {
		return nil, false
	}
	return order, true
}

func parseExerciseCharacterBackspacePayloadFromData(data string) (uuid.UUID, bool) {
	handlerType, payload, ok := parseCallbackData(data)
	if !ok || handlerType != callbackTypeExercise {
		return uuid.Nil, false
	}
	return parseExerciseCharacterBackspacePayload(payload)
}

func parseExerciseIDKPayloadFromData(data string) (uuid.UUID, bool) {
	handlerType, payload, ok := parseCallbackData(data)
	if !ok || handlerType != callbackTypeExercise {
		return uuid.Nil, false
	}
	return parseExerciseIDKPayload(payload)
}

func characterExerciseAnswer(exercise *services.TelegramMessageExercise) string {
	if exercise.ExerciseType == enums.ExerciseTypeCharactersReversed {
		return exercise.OriginalWord
	}
	return exercise.TranslationWord
}

func completedCharacterBoard(exercise *services.TelegramMessageExercise) *services.CharacterBoardState {
	if exercise.CharacterBoard != nil {
		return exercise.CharacterBoard
	}

	characters := services.AnswerCharacters(characterExerciseAnswer(exercise))
	chosen := make([]int, len(characters))
	for index := range chosen {
		chosen[index] = index
	}
	return &services.CharacterBoardState{
		Characters: characters,
		Chosen:     chosen,
		Answer:     strings.Join(characters, ""),
	}
}
