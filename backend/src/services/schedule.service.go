package services

import (
	"strconv"
	"strings"
	"termorize/src/logger"
	"termorize/src/models"
)

func CountTotalMinutesInSchedule(schedule []models.UserTelegramQuestionsScheduleItem) int {
	total := 0

	for _, item := range schedule {
		from := strings.Split(item.From, ":")
		to := strings.Split(item.To, ":")

		fromHour, _ := strconv.Atoi(from[0])
		fromMinute, _ := strconv.Atoi(from[1])

		toHour, _ := strconv.Atoi(to[0])
		toMinute, _ := strconv.Atoi(to[1])

		total += (toHour - fromHour) * 60
		total += toMinute - fromMinute + 1
	}

	return total
}

func MapOffsetOnSchedule(schedule []models.UserTelegramQuestionsScheduleItem, midnightOffset int) int {
	remainingOffset := midnightOffset

	for _, item := range schedule {
		from := strings.Split(item.From, ":")
		to := strings.Split(item.To, ":")

		fromHour, _ := strconv.Atoi(from[0])
		fromMinute, _ := strconv.Atoi(from[1])

		toHour, _ := strconv.Atoi(to[0])
		toMinute, _ := strconv.Atoi(to[1])

		minutesOnInterval := (toHour-fromHour)*60 + toMinute - fromMinute + 1

		if remainingOffset < minutesOnInterval {
			return fromHour*60 + fromMinute + remainingOffset
		}

		remainingOffset -= minutesOnInterval
	}

	logger.L().Error("can't map offset on schedule", "schedule", schedule, "offset", midnightOffset)

	return 0
}
