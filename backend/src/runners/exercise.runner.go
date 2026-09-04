package runners

import (
	"errors"
	"sync"
	"termorize/src/enums"
	"termorize/src/integrations/telegram"
	"termorize/src/logger"
	"termorize/src/monitoring"
	"termorize/src/services"
	"time"
)

var exerciseRunnerOnce sync.Once

func StartExerciseRunner() {
	exerciseRunnerOnce.Do(func() {
		go runExerciseRunner()
	})
}

func runExerciseRunner() {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.L().Errorw("exercise runner panicked", "panic", recovered)
			monitoring.Recover(nil, recovered)
		}
	}()

	processDueExercises()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		processDueExercises()
	}
}

func processDueExercises() {
	now := time.Now().UTC()

	if err := services.ExpireStaleInProgressExercises(now); err != nil {
		logger.L().Errorw("failed to expire stale in-progress exercises", "error", err)
		monitoring.CaptureException(nil, err)
	}

	if err := processDueExerciseReminders(now); err != nil {
		logger.L().Errorw("failed to process exercise reminders", "error", err)
		monitoring.CaptureException(nil, err)
	}

	if err := services.IgnoreDuePendingExercisesWithoutActiveVocabulary(now); err != nil {
		logger.L().Errorw("failed to ignore invalid pending exercises", "error", err)
		monitoring.CaptureException(nil, err)
	}

	exercises, err := services.GetDuePendingExercises(now)
	if err != nil {
		logger.L().Errorw("failed to fetch due pending exercises", "error", err)
		monitoring.CaptureException(nil, err)
		return
	}

	for _, exercise := range exercises {
		if !isSupportedExerciseType(exercise.ExerciseType) {
			continue
		}

		texts := telegram.GetBotTexts(exercise.SystemLanguage)
		questionText := telegram.BuildBasicExerciseQuestion(
			exercise.OriginalWord,
			exercise.TranslationWord,
			exercise.OriginalLanguage,
			exercise.TranslationLanguage,
			exercise.ExerciseType,
			texts,
		)
		if exercise.IsKnownVocabularyRepetition {
			questionText = telegram.BuildKnownVocabularyRepetitionQuestion(questionText, texts)
		}
		if exercise.ExerciseType == enums.ExerciseTypeDescriptionReversed {
			eligible, checkErr := services.IsDescriptionLanguageEligible(exercise.UserID, exercise.OriginalLanguage)
			if checkErr != nil {
				logger.L().Warnw("failed to check description language", "error", checkErr, "exercise_id", exercise.ExerciseID)
				continue
			}
			if !eligible {
				if _, replaceErr := services.ReplacePendingDescriptionExercise(exercise.ExerciseID, false); replaceErr != nil {
					logger.L().Warnw("failed to replace ineligible description exercise", "error", replaceErr, "exercise_id", exercise.ExerciseID)
				}
				continue
			}

			description, loadErr := services.GetOrCreateTranslationDescription(exercise.TranslationID)
			if loadErr != nil {
				logger.L().Warnw("replacing description exercise after generation failed", "error", loadErr, "exercise_id", exercise.ExerciseID)
				if _, replaceErr := services.ReplacePendingDescriptionExercise(exercise.ExerciseID, true); replaceErr != nil {
					logger.L().Warnw("failed to replace description exercise", "error", replaceErr, "exercise_id", exercise.ExerciseID)
				}
				continue
			}
			questionText = telegram.BuildDescriptionExerciseQuestion(description.Description, exercise.OriginalLanguage, texts)
		}

		var (
			messageID      *int64
			characterBoard *services.CharacterBoardState
			err            error
		)

		if isAudioExerciseType(exercise.ExerciseType) {
			spokenLanguage := exercise.OriginalLanguage
			answerLanguage := exercise.TranslationLanguage
			audioWordID := exercise.OriginalWordID
			if exercise.ExerciseType == enums.ExerciseTypeAudioReversed {
				spokenLanguage = exercise.TranslationLanguage
				answerLanguage = exercise.OriginalLanguage
				audioWordID = exercise.TranslationWordID
			}

			ignored, checkErr := services.IsAudioLanguageIgnored(exercise.UserID, spokenLanguage)
			if checkErr != nil {
				logger.L().Warnw("failed to check ignored audio language", "error", checkErr, "exercise_id", exercise.ExerciseID)
				continue
			}
			if ignored {
				if _, replaceErr := services.ReplacePendingAudioExercise(exercise.ExerciseID, false); replaceErr != nil {
					logger.L().Warnw("failed to replace ignored audio exercise", "error", replaceErr, "exercise_id", exercise.ExerciseID)
				}
				continue
			}

			pronunciation, loadErr := services.FindConfiguredWordPronunciationMetadata(audioWordID, string(spokenLanguage))
			if loadErr == nil && pronunciation == nil {
				pronunciation, loadErr = services.GetOrCreateWordPronunciation(audioWordID)
			}
			if loadErr != nil {
				if errors.Is(loadErr, services.ErrPronunciationGenerationFailed) {
					logger.L().Warnw("replacing audio exercise after pronunciation generation failed", "error", loadErr, "exercise_id", exercise.ExerciseID)
					if _, replaceErr := services.ReplacePendingAudioExercise(exercise.ExerciseID, true); replaceErr != nil {
						logger.L().Warnw("failed to replace audio exercise after pronunciation failure", "error", replaceErr, "exercise_id", exercise.ExerciseID)
					}
				} else {
					logger.L().Warnw("failed to prepare audio exercise", "error", loadErr, "exercise_id", exercise.ExerciseID)
				}
				continue
			}

			ignored, checkErr = services.IsAudioLanguageIgnored(exercise.UserID, spokenLanguage)
			if checkErr != nil || ignored {
				if checkErr != nil {
					logger.L().Warnw("failed final ignored audio language check", "error", checkErr, "exercise_id", exercise.ExerciseID)
				} else if _, replaceErr := services.ReplacePendingAudioExercise(exercise.ExerciseID, false); replaceErr != nil {
					logger.L().Warnw("failed to replace ignored audio exercise", "error", replaceErr, "exercise_id", exercise.ExerciseID)
				}
				continue
			}

			messageID, err = telegram.SendAudioExerciseMessage(
				exercise.TelegramID,
				exercise.ExerciseID,
				pronunciation,
				spokenLanguage,
				answerLanguage,
				texts,
			)
		} else if isCharacterExerciseType(exercise.ExerciseType) {
			answer := exercise.TranslationWord
			if exercise.ExerciseType == enums.ExerciseTypeCharactersReversed {
				answer = exercise.OriginalWord
			}
			characterBoard = services.BuildCharacterBoardForAnswer(answer)
			if len(characterBoard.Characters) == 0 {
				logger.L().Warnw("ignoring character exercise with an empty answer", "exercise_id", exercise.ExerciseID)
				if markErr := services.MarkExerciseVocabularyResultWithoutProgress(exercise.ExerciseID, services.ExerciseVocabularyResultIgnored, services.ExerciseVocabularyResultReasonInvalidOptions); markErr != nil {
					logger.L().Warnw("failed to mark invalid exercise vocabulary result", "error", markErr, "exercise_id", exercise.ExerciseID)
				}
				if ignoreErr := services.IgnoreExercise(exercise.ExerciseID); ignoreErr != nil {
					logger.L().Warnw("failed to ignore invalid exercise", "error", ignoreErr, "exercise_id", exercise.ExerciseID)
				}
				continue
			}

			messageID, err = telegram.SendCharacterExerciseMessage(exercise.TelegramID, questionText, exercise.ExerciseID, characterBoard, texts)
		} else if isChoiceExerciseType(exercise.ExerciseType) {
			options, loadErr := services.GetExerciseAnswerOptions(exercise.ExerciseID, exercise.ExerciseType)
			if loadErr != nil {
				logger.L().Warnw("failed to load exercise options", "error", loadErr, "exercise_id", exercise.ExerciseID)
				continue
			}
			if len(options) != 4 {
				logger.L().Warnw("ignoring choice exercise with incomplete options", "exercise_id", exercise.ExerciseID, "options_count", len(options))
				if markErr := services.MarkExerciseVocabularyResultWithoutProgress(exercise.ExerciseID, services.ExerciseVocabularyResultIgnored, services.ExerciseVocabularyResultReasonInvalidOptions); markErr != nil {
					logger.L().Warnw("failed to mark invalid exercise vocabulary result", "error", markErr, "exercise_id", exercise.ExerciseID)
				}
				if ignoreErr := services.IgnoreExercise(exercise.ExerciseID); ignoreErr != nil {
					logger.L().Warnw("failed to ignore invalid exercise", "error", ignoreErr, "exercise_id", exercise.ExerciseID)
				}
				continue
			}

			messageID, err = telegram.SendChoiceExerciseMessage(exercise.TelegramID, questionText, exercise.ExerciseID, options, texts)
		} else {
			messageID, err = telegram.SendBasicExerciseMessage(exercise.TelegramID, questionText, exercise.ExerciseID, texts)
		}

		if err != nil {
			logger.L().Warnw("failed to send scheduled exercise", "error", err, "exercise_id", exercise.ExerciseID, "telegram_id", exercise.TelegramID)
			continue
		}

		if messageID == nil {
			continue
		}

		logger.L().Infow("exercise sent", "username", exercise.Username)

		if characterBoard != nil {
			err = services.StartCharacterExercise(exercise.ExerciseID, *messageID, characterBoard.Order)
		} else {
			err = services.StartTelegramExercise(exercise.ExerciseID, *messageID)
		}
		if err != nil {
			logger.L().Warnw("failed to mark exercise in progress", "error", err, "exercise_id", exercise.ExerciseID)
		}
	}

	processDueMatchExercises(now)
}

func processDueMatchExercises(now time.Time) {
	matchExercises, err := services.GetDuePendingMatchExercises(now)
	if err != nil {
		logger.L().Errorw("failed to fetch due pending match exercises", "error", err)
		monitoring.CaptureException(nil, err)
		return
	}

	for _, exercise := range matchExercises {
		texts := telegram.GetBotTexts(exercise.SystemLanguage)

		cards, order, err := services.BuildMatchBoard(exercise.ExerciseID)
		if errors.Is(err, services.ErrExerciseVocabularyDeleted) {
			if markErr := services.MarkExerciseVocabularyResultWithoutProgress(exercise.ExerciseID, services.ExerciseVocabularyResultIgnored, services.ExerciseVocabularyResultReasonDeletedVocabulary); markErr != nil {
				logger.L().Warnw("failed to mark deleted match exercise vocabulary result", "error", markErr, "exercise_id", exercise.ExerciseID)
			}
			if ignoreErr := services.IgnoreExercise(exercise.ExerciseID); ignoreErr != nil {
				logger.L().Warnw("failed to ignore match exercise with deleted vocabulary", "error", ignoreErr, "exercise_id", exercise.ExerciseID)
			}
			continue
		}
		if err != nil {
			logger.L().Warnw("failed to build match board", "error", err, "exercise_id", exercise.ExerciseID)
			monitoring.CaptureException(nil, err)
			continue
		}

		messageID, err := telegram.SendMatchExerciseMessage(exercise.TelegramID, exercise.ExerciseID, cards, order, texts)
		if err != nil {
			logger.L().Warnw("failed to send match exercise", "error", err, "exercise_id", exercise.ExerciseID, "telegram_id", exercise.TelegramID)
			continue
		}

		if messageID == nil {
			continue
		}

		logger.L().Infow("match exercise sent", "username", exercise.Username)

		if err := services.StartMatchExercise(exercise.ExerciseID, *messageID, order); err != nil {
			logger.L().Warnw("failed to start match exercise", "error", err, "exercise_id", exercise.ExerciseID)
		}
	}
}

func processDueExerciseReminders(now time.Time) error {
	reminders, err := services.GetDueExerciseReminders(now)
	if err != nil {
		return err
	}

	for _, reminder := range reminders {
		texts := telegram.GetBotTexts(reminder.SystemLanguage)
		if err := telegram.SendReplyMessage(
			reminder.TelegramID,
			telegram.BuildExerciseReminderText(texts),
			reminder.TelegramMessageID,
		); err != nil {
			logger.L().Warnw("failed to send exercise reminder", "error", err, "exercise_id", reminder.ExerciseID, "telegram_id", reminder.TelegramID)
			continue
		}

		updated, err := services.MarkExerciseReminderSent(reminder.ExerciseID, now)
		if err != nil {
			logger.L().Warnw("failed to mark exercise reminder as sent", "error", err, "exercise_id", reminder.ExerciseID)
			continue
		}

		if !updated {
			continue
		}

		logger.L().Infow("exercise reminder sent", "exercise_id", reminder.ExerciseID, "telegram_id", reminder.TelegramID)
	}

	return nil
}

func isSupportedExerciseType(exerciseType enums.ExerciseType) bool {
	switch exerciseType {
	case enums.ExerciseTypeBasicDirect, enums.ExerciseTypeBasicReversed,
		enums.ExerciseTypeChoiceDirect, enums.ExerciseTypeChoiceReversed,
		enums.ExerciseTypeCharactersDirect, enums.ExerciseTypeCharactersReversed,
		enums.ExerciseTypeAudioDirect, enums.ExerciseTypeAudioReversed,
		enums.ExerciseTypeDescriptionReversed:
		return true
	default:
		return false
	}
}

func isAudioExerciseType(exerciseType enums.ExerciseType) bool {
	return exerciseType == enums.ExerciseTypeAudioDirect || exerciseType == enums.ExerciseTypeAudioReversed
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
