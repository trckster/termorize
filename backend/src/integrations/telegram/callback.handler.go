package telegram

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"termorize/src/enums"
	"termorize/src/logger"
	"termorize/src/services"

	"github.com/google/uuid"
)

func handleCallbackQuery(callback *callbackQuery) error {
	if callback == nil {
		return nil
	}

	if callback.ID != "" && !shouldDeferCallbackAnswer(callback) {
		if err := answerTelegramCallbackQuery(callback.ID); err != nil {
			logger.L().Warnw("failed to answer callback query", "error", err, "callback_id", callback.ID)
		}
	}

	if callback.From == nil {
		return nil
	}

	return routeCallbackData(callback)
}

func shouldDeferCallbackAnswer(callback *callbackQuery) bool {
	if callback.From == nil || callback.Message == nil {
		return false
	}

	handlerType, payload, ok := parseCallbackData(callback.Data)
	return ok && handlerType == callbackTypeExercise && len(payload) > 0 && payload[0] == exerciseActionMatchTap
}

func parseCallbackData(data string) (string, []string, bool) {
	parts := strings.Split(data, ":")
	if len(parts) < 2 || parts[0] == "" {
		return "", nil, false
	}

	return parts[0], parts[1:], true
}

func routeCallbackData(callback *callbackQuery) error {
	handlerType, payload, ok := parseCallbackData(callback.Data)
	if !ok {
		return nil
	}

	switch handlerType {
	case callbackTypeExercise:
		return handleExerciseCallback(callback, payload)
	case callbackTypeMenu:
		return handleMenuCallback(callback, payload)
	case callbackTypeVocabulary:
		return handleVocabularyCallback(callback, payload)
	default:
		return nil
	}
}

func parseExerciseIDKPayload(payload []string) (uuid.UUID, bool) {
	if len(payload) != 2 || payload[0] != exerciseActionIDK {
		return uuid.Nil, false
	}

	exerciseID, err := uuid.Parse(payload[1])
	if err != nil {
		return uuid.Nil, false
	}

	return exerciseID, true
}

func parseExerciseAnswerPayload(payload []string) (uuid.UUID, uuid.UUID, bool) {
	if len(payload) != 3 || payload[0] != exerciseActionAnswer {
		return uuid.Nil, uuid.Nil, false
	}

	exerciseID, err := parseCallbackUUID(payload[1])
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}

	selectedVocabularyID, err := parseCallbackUUID(payload[2])
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}

	return exerciseID, selectedVocabularyID, true
}

func parseExerciseMatchPayload(payload []string) (uuid.UUID, int, bool) {
	if len(payload) != 3 || payload[0] != exerciseActionMatchTap {
		return uuid.Nil, 0, false
	}

	exerciseID, err := parseCallbackUUID(payload[1])
	if err != nil {
		return uuid.Nil, 0, false
	}

	tappedIdx, err := strconv.Atoi(payload[2])
	if err != nil || tappedIdx < 0 || tappedIdx > 9 {
		return uuid.Nil, 0, false
	}

	return exerciseID, tappedIdx, true
}

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

func parseCallbackUUID(value string) (uuid.UUID, error) {
	if id, err := uuid.Parse(value); err == nil {
		return id, nil
	}

	bytes, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return uuid.Nil, err
	}

	return uuid.FromBytes(bytes)
}

func handleExerciseCallback(callback *callbackQuery, payload []string) error {
	if callback.Message == nil {
		return nil
	}

	t := getBotTextsForTelegramID(callback.From.ID)

	if len(payload) > 0 && payload[0] == exerciseActionMatchNoop {
		return nil
	}
	if len(payload) > 0 && payload[0] == exerciseActionCharacterNoop {
		return nil
	}
	if len(payload) >= 2 && payload[0] == exerciseActionMatchTap {
		return handleMatchTap(callback, payload, t)
	}
	if len(payload) >= 2 && payload[0] == exerciseActionCharacterTap {
		return handleCharacterTap(callback, payload, t)
	}

	exerciseID, selectedVocabularyID, hasAnswer := parseExerciseAnswerPayload(payload)
	if !hasAnswer {
		var ok bool
		exerciseID, ok = parseExerciseIDKPayload(payload)
		if !ok {
			return nil
		}
	}

	exercise, err := services.GetExerciseByTelegramMessage(callback.Message.MessageID, callback.From.ID)
	if err != nil {
		return err
	}

	if exercise == nil || exercise.ExerciseID != exerciseID {
		return nil
	}

	switch exercise.Status {
	case enums.ExerciseStatusIgnored:
		return SendMessage(callback.From.ID, t.ExerciseOutdated)
	case enums.ExerciseStatusCompleted:
		return SendMessage(callback.From.ID, t.ExerciseCompleted)
	case enums.ExerciseStatusFailed:
		return SendMessage(callback.From.ID, t.ExerciseFailed)
	}

	if len(exercise.Vocabulary) == 0 || exercise.Vocabulary[0].Translation == nil {
		_ = services.MarkExerciseVocabularyResultWithoutProgress(exercise.ExerciseID, services.ExerciseVocabularyResultIgnored, services.ExerciseVocabularyResultReasonDeletedVocabulary)
		_ = services.IgnoreExercise(exercise.ExerciseID)
		return SendMessage(callback.From.ID, t.ExerciseVocabularyDeleted)
	}

	if (exercise.ExerciseType == enums.ExerciseTypeChoiceDirect || exercise.ExerciseType == enums.ExerciseTypeChoiceReversed) &&
		len(exercise.Options) != services.ChoiceExerciseVocabularyCount {
		_ = services.MarkExerciseVocabularyResultWithoutProgress(exercise.ExerciseID, services.ExerciseVocabularyResultIgnored, services.ExerciseVocabularyResultReasonInvalidOptions)
		_ = services.IgnoreExercise(exercise.ExerciseID)
		return SendMessage(callback.From.ID, t.ExerciseVocabularyDeleted)
	}

	if !hasAnswer && exercise.ExerciseType != enums.ExerciseTypeBasicDirect && exercise.ExerciseType != enums.ExerciseTypeBasicReversed {
		return nil
	}

	if err := removeMessageInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID); err != nil {
		logger.L().Warnw("failed to remove inline keyboard", "error", err, "chat_id", callback.Message.Chat.ID, "message_id", callback.Message.MessageID)
	}

	if hasAnswer {
		result, err := services.VerifyExerciseChoice(exerciseID, exercise.UserID, selectedVocabularyID)
		if err != nil {
			if errors.Is(err, services.ErrExerciseNotInProgress) {
				return nil
			}

			if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
				return SendMessage(callback.From.ID, t.ExerciseVocabularyDeleted)
			}

			return err
		}

		switch result.Result {
		case "correct":
			return SendMessageMarkdown(callback.From.ID, buildExerciseSuccessResultText(result.Knowledge, t))
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

	updated, translationKnowledge, err := services.FinishExercise(
		exerciseID,
		enums.ExerciseStatusFailed,
		services.ExerciseVocabularyResultIgnored,
		services.ExerciseVocabularyResultReasonSkipped,
		services.ExerciseFailProgressDelta,
	)
	if err != nil {
		return err
	}

	if !updated {
		return nil
	}

	words, err := services.GetExerciseWordsByTelegram(exerciseID, callback.From.ID)
	if err != nil {
		return err
	}

	if words == nil {
		return nil
	}

	answerText := buildExerciseIDKResultText(
		words.OriginalWord,
		words.TranslationWord,
		words.OriginalLanguage,
		words.TranslationLanguage,
		translationKnowledge,
		t,
	)
	return SendMessageMarkdown(callback.From.ID, answerText)
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
		return SendMessage(callback.From.ID, t.ExerciseOutdated)
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
		return SendMessage(callback.From.ID, t.ExerciseVocabularyDeleted)
	}

	board, finished, err := services.ApplyCharacterTap(exercise.ExerciseID, exercise.UserID, tappedIndex)
	if err != nil {
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil
		}
		if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
			if removeErr := removeMessageInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID); removeErr != nil {
				logger.L().Warnw("failed to remove inline keyboard", "error", removeErr, "chat_id", callback.Message.Chat.ID, "message_id", callback.Message.MessageID)
			}
			return SendMessage(callback.From.ID, t.ExerciseVocabularyDeleted)
		}
		return err
	}

	if !finished {
		return EditCharacterBoardMessage(
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			buildCharacterBoardText(questionText, board),
			buildCharacterKeyboard(exercise.ExerciseID, board),
		)
	}

	result, err := services.VerifyExerciseAnswer(exercise.ExerciseID, exercise.UserID, board.Answer)
	if err != nil {
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil
		}
		if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
			return SendMessage(callback.From.ID, t.ExerciseVocabularyDeleted)
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

	order := make([]int, 0, characterCount)
	seen := make(map[int]bool, characterCount)
	for _, row := range markup.InlineKeyboard {
		for _, button := range row {
			handlerType, payload, ok := parseCallbackData(button.CallbackData)
			if !ok || handlerType != callbackTypeExercise || len(payload) == 0 {
				return nil, false
			}
			if payload[0] == exerciseActionCharacterNoop {
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

	if len(order) != characterCount {
		return nil, false
	}
	return order, true
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

func handleMatchTap(callback *callbackQuery, payload []string, t BotTexts) error {
	if callback.Message == nil {
		return nil
	}

	callbackAnswered := false
	defer func() {
		if callback.ID == "" || callbackAnswered {
			return
		}
		if answerErr := answerTelegramCallbackQuery(callback.ID); answerErr != nil {
			logger.L().Warnw("failed to answer match callback", "error", answerErr, "callback_id", callback.ID)
		}
	}()

	exerciseID, tappedIdx, ok := parseExerciseMatchPayload(payload)
	if !ok {
		return nil
	}

	exercise, err := services.GetExerciseByTelegramMessage(callback.Message.MessageID, callback.From.ID)
	if err != nil {
		return err
	}

	if exercise == nil {
		exercise, err = recoverPendingMatchExerciseFromCallback(callback, exerciseID)
		if err != nil {
			return err
		}
	}

	if exercise == nil || exercise.ExerciseID != exerciseID {
		return nil
	}

	switch exercise.Status {
	case enums.ExerciseStatusIgnored:
		return SendMessage(callback.From.ID, t.ExerciseOutdated)
	case enums.ExerciseStatusCompleted, enums.ExerciseStatusFailed:
		result, resultErr := services.GetCompletedMatchPairsResult(exercise.ExerciseID, exercise.UserID)
		if resultErr != nil {
			return resultErr
		}
		return EditMatchBoardMessage(callback.Message.Chat.ID, callback.Message.MessageID, buildMatchResultSummaryText(result, t), [][]inlineKeyboardButton{})
	}

	board, wasWrong, finished, finalizeAttempts, err := services.ApplyMatchTap(exercise.ExerciseID, exercise.UserID, tappedIdx)
	if err != nil {
		if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
			if removeErr := removeMessageInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID); removeErr != nil {
				logger.L().Warnw("failed to remove inline keyboard", "error", removeErr, "chat_id", callback.Message.Chat.ID, "message_id", callback.Message.MessageID)
			}
			return SendMessage(callback.From.ID, t.ExerciseVocabularyDeleted)
		}
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil
		}

		return err
	}

	if !finished {
		if wasWrong {
			if answerErr := answerTelegramCallbackQueryWithText(callback.ID, t.MatchNotAMatchToast); answerErr != nil {
				logger.L().Warnw("failed to answer match callback toast", "error", answerErr, "callback_id", callback.ID)
			} else {
				callbackAnswered = true
			}
		}

		return EditMatchBoardMessage(callback.Message.Chat.ID, callback.Message.MessageID, buildMatchBoardText(board, t), buildMatchKeyboard(exercise.ExerciseID, board))
	}

	if len(finalizeAttempts) == 0 {
		return nil
	}

	result, err := services.CompleteMatchPairsExercise(exercise.ExerciseID, exercise.UserID, finalizeAttempts)
	if err != nil {
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil
		}
		if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
			if removeErr := removeMessageInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID); removeErr != nil {
				logger.L().Warnw("failed to remove inline keyboard", "error", removeErr, "chat_id", callback.Message.Chat.ID, "message_id", callback.Message.MessageID)
			}
			return SendMessage(callback.From.ID, t.ExerciseVocabularyDeleted)
		}

		return err
	}

	return EditMatchBoardMessage(callback.Message.Chat.ID, callback.Message.MessageID, buildMatchResultSummaryText(result, t), [][]inlineKeyboardButton{})
}

func recoverPendingMatchExerciseFromCallback(callback *callbackQuery, exerciseID uuid.UUID) (*services.TelegramMessageExercise, error) {
	exercise, err := services.GetExerciseByTelegramExerciseID(exerciseID, callback.From.ID)
	if err != nil {
		return nil, err
	}
	if exercise == nil || exercise.Status != enums.ExerciseStatusPending || exercise.ExerciseType != enums.ExerciseTypeMatchPairs {
		return nil, nil
	}

	order, ok := extractMatchOrderFromReplyMarkup(callback.Message.ReplyMarkup, exerciseID)
	if !ok {
		return nil, nil
	}

	if err := services.StartMatchExercise(exerciseID, callback.Message.MessageID, order); err != nil {
		if errors.Is(err, services.ErrExerciseNotInProgress) {
			return nil, nil
		}
		return nil, err
	}

	return services.GetExerciseByTelegramMessage(callback.Message.MessageID, callback.From.ID)
}

func extractMatchOrderFromReplyMarkup(markup *inlineKeyboardMarkup, exerciseID uuid.UUID) ([]int, bool) {
	if markup == nil {
		return nil, false
	}

	expectedCards := services.MatchPairsVocabularyCount * 2
	order := make([]int, 0, expectedCards)
	seen := make(map[int]bool, expectedCards)

	for _, row := range markup.InlineKeyboard {
		for _, button := range row {
			handlerType, payload, ok := parseCallbackData(button.CallbackData)
			if !ok || handlerType != callbackTypeExercise {
				return nil, false
			}

			buttonExerciseID, canonical, ok := parseExerciseMatchPayload(payload)
			if !ok || buttonExerciseID != exerciseID || seen[canonical] {
				return nil, false
			}

			seen[canonical] = true
			order = append(order, canonical)
		}
	}

	if len(order) != expectedCards {
		return nil, false
	}

	for i := 0; i < expectedCards; i++ {
		if !seen[i] {
			return nil, false
		}
	}

	return order, true
}

func handleMenuCallback(callback *callbackQuery, payload []string) error {
	if callback.Message == nil {
		return nil
	}

	if len(payload) == 0 {
		return nil
	}

	t := getBotTextsForTelegramID(callback.From.ID)
	action := payload[0]

	if action == menuActionBack || action == menuActionCancel {
		if _, err := services.UpdateUserTelegramState(callback.From.ID, enums.TelegramStateNone); err != nil {
			return err
		}

		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, t.Menu, getMenuKeyboard(t))
	}

	if action == menuActionDeleteTranslation {
		if _, err := services.UpdateUserTelegramState(callback.From.ID, enums.TelegramStateDeletingVocabulary); err != nil {
			return err
		}

		return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, t.MenuDeleteWord, getMenuCancelKeyboard(t))
	}

	if action == menuActionAddTranslation {
		if _, err := services.UpdateUserTelegramState(callback.From.ID, enums.TelegramStateAddingVocabulary); err != nil {
			return err
		}

		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		messageText := buildAddVocabularyFirstText(
			user.Settings.TranslationSourceLanguage.DisplayNameWithFlag(),
			user.Settings.TranslationTargetLanguage.DisplayNameWithFlag(),
			t,
		)
		keyboard := buildAddTranslationKeyboard(user.Settings.TranslationSourceLanguage, user.Settings.TranslationTargetLanguage, t)
		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, keyboard)
	}

	if action == menuActionVocabulary {
		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		messageText, err := buildVocabularyMenuText(user.ID, t)
		if err != nil {
			return err
		}

		return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, messageText, buildVocabularyOverviewKeyboard(t))
	}

	if action == menuActionStatistics {
		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		messageText, err := buildStatisticsMenuText(user.ID, t)
		if err != nil {
			return err
		}

		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, getMenuBackKeyboard(t))
	}

	if action == menuActionChangeSourceLang || action == menuActionChangeTargetLang {
		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		isSource := action == menuActionChangeSourceLang
		keyboard := buildLanguageSelectionKeyboard(user.Settings.TranslationSourceLanguage, user.Settings.TranslationTargetLanguage, isSource, t)
		return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, t.ChooseLanguage, keyboard)
	}

	if action == menuActionChangeSystemLang {
		keyboard := buildSystemLanguageSelectionKeyboard(t)
		return EditMessageTextWithInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID, t.ChooseLanguage, keyboard)
	}

	if action == menuActionSetSourceLang || action == menuActionSetTargetLang {
		if len(payload) != 2 {
			return nil
		}

		langCode := enums.Language(payload[1])
		isSource := action == menuActionSetSourceLang

		user, err := services.UpdateUserTranslationLanguage(callback.From.ID, isSource, langCode)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		messageText := buildAddVocabularyFirstText(
			user.Settings.TranslationSourceLanguage.DisplayNameWithFlag(),
			user.Settings.TranslationTargetLanguage.DisplayNameWithFlag(),
			t,
		)
		keyboard := buildAddTranslationKeyboard(user.Settings.TranslationSourceLanguage, user.Settings.TranslationTargetLanguage, t)
		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, keyboard)
	}

	if action == menuActionSetSystemLang {
		if len(payload) != 2 {
			return nil
		}

		langCode := enums.Language(payload[1])
		isSupported := false
		for _, lang := range getSupportedSystemLanguages() {
			if lang == langCode {
				isSupported = true
				break
			}
		}
		if !isSupported {
			return nil
		}

		user, err := services.UpdateUserSystemLanguage(callback.From.ID, langCode)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		updatedTexts := GetBotTexts(user.Settings.SystemLanguage)
		keyboard := buildSettingsKeyboard(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, updatedTexts)
		messageText := BuildSettingsText(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, updatedTexts)
		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, keyboard)
	}

	if action == menuActionToggleDailyExercises {
		user, err := services.UpdateUserTelegramDailyQuestionsEnabled(callback.From.ID, true)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		updatedTexts := GetBotTexts(user.Settings.SystemLanguage)
		keyboard := buildSettingsKeyboard(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, updatedTexts)
		messageText := BuildSettingsText(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, updatedTexts)
		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, keyboard)
	}

	if action == menuActionSettings {
		user, err := services.GetUserByTelegramID(callback.From.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return nil
		}

		keyboard := buildSettingsKeyboard(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, t)
		messageText := BuildSettingsText(user.Settings.SystemLanguage, user.Settings.Telegram.DailyQuestionsEnabled, t)
		return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, messageText, keyboard)
	}

	selectionText, ok := menuActionToText(action, t)
	if !ok {
		return nil
	}

	return EditMessageTextWithInlineKeyboardMarkdown(callback.Message.Chat.ID, callback.Message.MessageID, selectionText, getMenuBackKeyboard(t))
}
