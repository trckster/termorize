package services

import (
	"math/rand"
	"termorize/src/logger"
	"termorize/src/models"
	"time"
)

func GenerateDailyExercises() error {
	users, err := GetUsersWithEnabledDailyQuestions()
	if err != nil {
		return err
	}

	for _, user := range users {
		logger.L().Infow("test", ">", user)
		GenerateExercises(user)
	}

	return nil
}

func GenerateExercises(user models.User) {
	totalMinutes := CountTotalMinutesInSchedule(user.Settings.Telegram.DailyQuestionsSchedule)

	now := time.Now().Round(time.Hour * 24)

	for _, _ := range user.Settings.Telegram.DailyQuestionsSchedule {
		midnightOffset := rand.Intn(totalMinutes)

		realOffsetInMinutes := MapOffsetOnSchedule(user.Settings.Telegram.DailyQuestionsSchedule, midnightOffset)

		GetRandomTime()
	}

	var timings

	println(totalMinutes)
}
