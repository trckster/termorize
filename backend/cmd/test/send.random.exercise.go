package main

// Import first to set UTC timezone before any other package uses invalid timezone
import _ "termorize/src/utils"

import (
	"errors"
	"os"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/telegram"
	"termorize/src/logger"
	"termorize/src/models"
	"termorize/src/monitoring"
	"termorize/src/services"
	"time"

	"gorm.io/gorm"
)

// Test helper: generates an exercise for the first found user and sends it to
// Telegram immediately, mirroring the runner's send flow (see processDueExercises
// and processDueMatchExercises in src/runners/exercise.runner.go).
//
// Usage:
//
//	go run ./cmd/test/            # default: match/pairs exercise
//	go run ./cmd/test/ match      # match/pairs exercise
//	go run ./cmd/test/ basic      # basic exercise, random direction
//	go run ./cmd/test/ choice     # choice exercise, random direction
//	go run ./cmd/test/ characters # character exercise, random direction
//	go run ./cmd/test/ audio      # audio exercise, random direction
//	go run ./cmd/test/ repeat     # known-vocabulary repetition, random direction
const (
	testExerciseRunnerBuffer = time.Hour
)

func main() {
	defer logger.Sync()
	config.LoadEnv()

	monitoring.Init()
	defer monitoring.Flush()

	if err := db.Connect(); err != nil {
		fatal("database connection failed", err)
	}

	mode := "match"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	var user models.User
	if err := db.DB.Order("id ASC").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fatal("no users found", errors.New("users table is empty"))
		}
		fatal("failed to fetch first user", err)
	}

	if user.TelegramID == 0 {
		fatal("first user has no telegram_id", errors.New("cannot send exercise to telegram"))
	}

	logger.L().Infow("sending test exercise", "mode", mode, "user_id", user.ID, "username", user.Username, "telegram_id", user.TelegramID)

	texts := telegram.GetBotTexts(user.Settings.SystemLanguage)

	switch mode {
	case "match":
		sendMatchExercise(user, texts)
	case "basic":
		sendBasicOrChoiceExercise(user, texts,
			enums.ExerciseTypeBasicDirect,
			enums.ExerciseTypeBasicReversed,
		)
	case "choice":
		sendBasicOrChoiceExercise(user, texts,
			enums.ExerciseTypeChoiceDirect,
			enums.ExerciseTypeChoiceReversed,
		)
	case "characters":
		sendCharacterExercise(user, texts)
	case "audio":
		sendAudioExercise(user, texts)
	case "repeat":
		sendRepetitionExercise(user, texts)
	default:
		fatal("unknown mode", errors.New("supported modes: match, basic, choice, characters, audio, repeat"))
	}
}

// sendMatchExercise mirrors processDueMatchExercises: create a pending match
// exercise, build its board, send it, then mark it started.
func sendMatchExercise(user models.User, texts telegram.BotTexts) {
	// Keep the exercise out of the due queue while this helper sends and starts it.
	// Otherwise, a running server can claim the same exercise first.
	scheduledFor := time.Now().UTC().Add(testExerciseRunnerBuffer)
	exerciseID, err := services.CreatePendingMatchExercise(user.ID, scheduledFor)
	if err != nil {
		fatal("failed to create match exercise (needs >=5 non-mastered words in the same language pair)", err)
	}

	cards, order, err := services.BuildMatchBoard(exerciseID)
	if err != nil {
		fatal("failed to build match board", err)
	}

	messageID, err := telegram.SendMatchExerciseMessage(user.TelegramID, exerciseID, cards, order, texts)
	if err != nil {
		fatal("failed to send match exercise to telegram", err)
	}
	if messageID == nil {
		fatal("telegram did not return a message id", errors.New("user may have blocked the bot or disabled it"))
	}

	if err := services.StartMatchExercise(exerciseID, *messageID, order); err != nil {
		fatal("failed to mark match exercise as started", err)
	}

	logger.L().Infow("match exercise sent to telegram",
		"exercise_id", exerciseID,
		"telegram_id", user.TelegramID,
		"message_id", *messageID,
	)
}

func sendCharacterExercise(user models.User, texts telegram.BotTexts) {
	// Keep the exercise out of the due queue while this helper sends and starts it.
	// Otherwise, a running server can claim the same exercise first.
	scheduledFor := time.Now().UTC().Add(testExerciseRunnerBuffer)
	result, err := services.CreatePendingCharacterExercise(user.ID, scheduledFor)
	if err != nil {
		fatal("failed to create character exercise", err)
	}

	options, err := services.GetExerciseAnswerOptions(result.ExerciseID, result.Type)
	if err != nil {
		fatal("failed to load character exercise answer", err)
	}
	if len(options) != 1 {
		fatal("character exercise has an invalid answer", errors.New("expected exactly one answer"))
	}

	board := services.BuildCharacterBoardForAnswer(options[0].Label)
	questionText := buildExerciseText(result, texts)
	messageID, err := telegram.SendCharacterExerciseMessage(
		user.TelegramID,
		questionText,
		result.ExerciseID,
		board,
	)
	if err != nil {
		fatal("failed to send character exercise to telegram", err)
	}
	if messageID == nil {
		fatal("telegram did not return a message id", errors.New("user may have blocked the bot or disabled it"))
	}

	if err := services.StartCharacterExercise(result.ExerciseID, *messageID, board.Order); err != nil {
		fatal("failed to mark character exercise as started", err)
	}

	logger.L().Infow("character exercise sent to telegram",
		"exercise_id", result.ExerciseID,
		"exercise_type", result.Type,
		"telegram_id", user.TelegramID,
		"message_id", *messageID,
	)
}

func sendAudioExercise(user models.User, texts telegram.BotTexts) {
	result, err := services.CreateRandomExerciseOfTypes(
		user.ID,
		enums.ExerciseTypeAudioDirect,
		enums.ExerciseTypeAudioReversed,
	)
	if err != nil {
		fatal("failed to create audio exercise", err)
	}
	if result.AudioWordID == nil {
		fatal("audio exercise has no spoken word", errors.New("audio_word_id is missing"))
	}

	pronunciation, err := services.FindConfiguredWordPronunciationMetadata(
		*result.AudioWordID,
		string(result.Language),
	)
	if err == nil && pronunciation == nil {
		pronunciation, err = services.GetOrCreateWordPronunciation(*result.AudioWordID)
	}
	if err != nil {
		fatal("failed to prepare audio exercise pronunciation", err)
	}

	messageID, err := telegram.SendAudioExerciseMessage(
		user.TelegramID,
		result.ExerciseID,
		pronunciation,
		result.Language,
		result.AnswerLanguage,
		texts,
	)
	if err != nil {
		fatal("failed to send audio exercise to telegram", err)
	}
	if messageID == nil {
		fatal("telegram did not return a message id", errors.New("user may have blocked the bot or disabled it"))
	}

	// CreateRandomExerciseOfTypes already marks the exercise as in-progress, so
	// attach the Telegram message id directly, as for immediate basic exercises.
	if err := db.DB.Model(&models.Exercise{}).
		Where("id = ?", result.ExerciseID).
		Update("telegram_message_id", *messageID).Error; err != nil {
		fatal("failed to store telegram message id", err)
	}

	logger.L().Infow("audio exercise sent to telegram",
		"exercise_id", result.ExerciseID,
		"exercise_type", result.Type,
		"telegram_id", user.TelegramID,
		"message_id", *messageID,
	)
}

func sendRepetitionExercise(user models.User, texts telegram.BotTexts) {
	scheduledFor := time.Now().UTC().Add(testExerciseRunnerBuffer)
	exerciseID, err := services.CreatePendingKnownVocabularyRepetition(user.ID, scheduledFor)
	if err != nil {
		fatal("failed to create repetition exercise (needs a word with 100% knowledge)", err)
	}

	exercise, err := services.GetExerciseByTelegramExerciseID(exerciseID, user.TelegramID)
	if err != nil {
		fatal("failed to load repetition exercise", err)
	}
	if exercise == nil || len(exercise.Vocabulary) != 1 {
		fatal("repetition exercise has invalid vocabulary", errors.New("expected exactly one known word"))
	}

	questionText := telegram.BuildBasicExerciseQuestion(
		exercise.OriginalWord,
		exercise.TranslationWord,
		exercise.OriginalLanguage,
		exercise.TranslationLanguage,
		exercise.ExerciseType,
		texts,
	)
	questionText = telegram.BuildKnownVocabularyRepetitionQuestion(questionText, texts)

	messageID, err := telegram.SendBasicExerciseMessage(user.TelegramID, questionText, exerciseID, texts)
	if err != nil {
		fatal("failed to send repetition exercise to telegram", err)
	}
	if messageID == nil {
		fatal("telegram did not return a message id", errors.New("user may have blocked the bot or disabled it"))
	}

	if err := services.StartTelegramExercise(exerciseID, *messageID); err != nil {
		fatal("failed to mark repetition exercise as started", err)
	}

	logger.L().Infow("repetition exercise sent to telegram",
		"exercise_id", exerciseID,
		"exercise_type", exercise.ExerciseType,
		"telegram_id", user.TelegramID,
		"message_id", *messageID,
	)
}

// sendBasicOrChoiceExercise mirrors processDueExercises for an immediate
// (in-progress) exercise restricted to the requested direct/reversed pair.
func sendBasicOrChoiceExercise(user models.User, texts telegram.BotTexts, exerciseTypes ...enums.ExerciseType) {
	result, err := services.CreateRandomExerciseOfTypes(user.ID, exerciseTypes...)
	if err != nil {
		fatal("failed to create the requested exercise type", err)
	}

	questionText := buildExerciseText(result, texts)

	var messageID *int64
	if isChoiceExerciseType(result.Type) {
		options, optionsErr := services.GetExerciseAnswerOptions(result.ExerciseID, result.Type)
		if optionsErr != nil {
			fatal("failed to load exercise options", optionsErr)
		}
		messageID, err = telegram.SendChoiceExerciseMessage(user.TelegramID, questionText, result.ExerciseID, options, texts)
	} else {
		messageID, err = telegram.SendBasicExerciseMessage(user.TelegramID, questionText, result.ExerciseID, texts)
	}
	if err != nil {
		fatal("failed to send exercise to telegram", err)
	}
	if messageID == nil {
		fatal("telegram did not return a message id", errors.New("user may have blocked the bot or disabled it"))
	}

	// CreateRandomExercise already marks the exercise as in-progress, so just attach
	// the telegram message id (StartTelegramExercise only applies to pending exercises).
	if err := db.DB.Model(&models.Exercise{}).
		Where("id = ?", result.ExerciseID).
		Update("telegram_message_id", *messageID).Error; err != nil {
		fatal("failed to store telegram message id", err)
	}

	logger.L().Infow("exercise sent to telegram",
		"exercise_id", result.ExerciseID,
		"exercise_type", result.Type,
		"telegram_id", user.TelegramID,
		"message_id", *messageID,
	)
}

// buildExerciseText maps RandomExerciseResult onto BuildBasicExerciseQuestion, which
// picks the shown word and target language from its arguments based on exercise type.
func buildExerciseText(result *services.RandomExerciseResult, texts telegram.BotTexts) string {
	switch result.Type {
	case enums.ExerciseTypeBasicReversed, enums.ExerciseTypeChoiceReversed, enums.ExerciseTypeCharactersReversed:
		return telegram.BuildBasicExerciseQuestion("", result.QuestionWord, result.AnswerLanguage, result.Language, result.Type, texts)
	default:
		return telegram.BuildBasicExerciseQuestion(result.QuestionWord, "", result.Language, result.AnswerLanguage, result.Type, texts)
	}
}

func isChoiceExerciseType(exerciseType enums.ExerciseType) bool {
	return exerciseType == enums.ExerciseTypeChoiceDirect || exerciseType == enums.ExerciseTypeChoiceReversed
}

func fatal(message string, err error) {
	monitoring.CaptureException(nil, err)
	monitoring.Flush()
	logger.L().Fatalw(message, "error", err)
}
