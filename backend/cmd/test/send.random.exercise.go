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
//	go run ./cmd/test/ basic      # random basic/choice exercise
//	go run ./cmd/test/ characters # character exercise, random direction
const maxGenerationAttempts = 10

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
	case "basic", "choice", "random":
		sendBasicOrChoiceExercise(user, texts)
	case "characters":
		sendCharacterExercise(user, texts)
	default:
		fatal("unknown mode", errors.New("supported modes: match, basic, characters"))
	}
}

// sendMatchExercise mirrors processDueMatchExercises: create a pending match
// exercise, build its board, send it, then mark it started.
func sendMatchExercise(user models.User, texts telegram.BotTexts) {
	exerciseID, err := services.CreatePendingMatchExercise(user.ID, time.Now().UTC())
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
	result, err := services.CreatePendingCharacterExercise(user.ID, time.Now().UTC())
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
		texts,
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

// sendBasicOrChoiceExercise mirrors processDueExercises for the immediate
// (in-progress) exercise created via CreateRandomExercise.
func sendBasicOrChoiceExercise(user models.User, texts telegram.BotTexts) {
	result, err := generateSendableExercise(user.ID)
	if err != nil {
		fatal("failed to create a sendable basic/choice exercise", err)
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

// generateSendableExercise returns a basic/choice exercise. Match/pairs is created
// through its own pending+runner path (see sendMatchExercise), so it is skipped here.
func generateSendableExercise(userID uint) (*services.RandomExerciseResult, error) {
	for attempt := 0; attempt < maxGenerationAttempts; attempt++ {
		result, err := services.CreateRandomExercise(userID)
		if err != nil {
			return nil, err
		}

		if result.Type != enums.ExerciseTypeMatchPairs &&
			result.Type != enums.ExerciseTypeCharactersDirect &&
			result.Type != enums.ExerciseTypeCharactersReversed {
			return result, nil
		}

		logger.L().Infow("skipping board-based result, retrying for basic/choice", "exercise_id", result.ExerciseID)
		if ignoreErr := services.IgnoreExercise(result.ExerciseID); ignoreErr != nil {
			logger.L().Warnw("failed to ignore unused match/pairs exercise", "error", ignoreErr, "exercise_id", result.ExerciseID)
		}
	}

	return nil, errors.New("could not generate a basic/choice exercise within attempt limit")
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
