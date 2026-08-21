package services

import (
	"strconv"
	"strings"
	"termorize/src/logger"
	"termorize/src/models"
)

func parseHHMM(value string) (int, int, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}

	return hour, minute, true
}

// scheduleIntervalMinutes returns the inclusive minute count of a schedule
// interval, or zero when the interval is malformed or inverted.
func scheduleIntervalMinutes(item models.UserTelegramQuestionsScheduleItem) int {
	fromHour, fromMinute, ok := parseHHMM(item.From)
	if !ok {
		return 0
	}

	toHour, toMinute, ok := parseHHMM(item.To)
	if !ok {
		return 0
	}

	minutes := (toHour-fromHour)*60 + toMinute - fromMinute + 1
	if minutes <= 0 {
		return 0
	}

	return minutes
}

func CountTotalMinutesInSchedule(schedule []models.UserTelegramQuestionsScheduleItem) int {
	total := 0

	for _, item := range schedule {
		total += scheduleIntervalMinutes(item)
	}

	return total
}

func MapOffsetOnSchedule(schedule []models.UserTelegramQuestionsScheduleItem, midnightOffset int) int {
	remainingOffset := midnightOffset

	for _, item := range schedule {
		minutesOnInterval := scheduleIntervalMinutes(item)
		if minutesOnInterval <= 0 {
			continue
		}

		if remainingOffset < minutesOnInterval {
			fromHour, fromMinute, _ := parseHHMM(item.From)
			return fromHour*60 + fromMinute + remainingOffset
		}

		remainingOffset -= minutesOnInterval
	}

	logger.L().Error("can't map offset on schedule", "schedule", schedule, "offset", midnightOffset)

	return 0
}
