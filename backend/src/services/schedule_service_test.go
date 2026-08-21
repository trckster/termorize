package services

import (
	"testing"

	"termorize/src/models"

	"github.com/stretchr/testify/assert"
)

func TestCountTotalMinutesInSchedule(t *testing.T) {
	tests := []struct {
		name      string
		schedule  []models.UserTelegramQuestionsScheduleItem
		wantTotal int
	}{
		{
			name:      "empty schedule",
			schedule:  nil,
			wantTotal: 0,
		},
		{
			name:      "single valid interval counts inclusive minutes",
			schedule:  []models.UserTelegramQuestionsScheduleItem{{From: "09:00", To: "18:00"}},
			wantTotal: 541,
		},
		{
			name:      "inverted interval contributes nothing",
			schedule:  []models.UserTelegramQuestionsScheduleItem{{From: "22:00", To: "06:00"}},
			wantTotal: 0,
		},
		{
			name: "malformed interval contributes nothing without panicking",
			schedule: []models.UserTelegramQuestionsScheduleItem{
				{From: "9am", To: "25:00"},
				{From: "10", To: "12:00"},
				{From: "10:00"},
			},
			wantTotal: 0,
		},
		{
			name: "valid intervals are summed while invalid ones are skipped",
			schedule: []models.UserTelegramQuestionsScheduleItem{
				{From: "22:00", To: "06:00"},
				{From: "10:00", To: "10:30"},
			},
			wantTotal: 31,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantTotal, CountTotalMinutesInSchedule(tt.schedule))
		})
	}
}

func TestMapOffsetOnSchedule(t *testing.T) {
	tests := []struct {
		name        string
		schedule    []models.UserTelegramQuestionsScheduleItem
		midnightOff int
		wantMinutes int
	}{
		{
			name:        "offset at the start of the first interval",
			schedule:    []models.UserTelegramQuestionsScheduleItem{{From: "09:00", To: "18:00"}},
			midnightOff: 0,
			wantMinutes: 540,
		},
		{
			name:        "offset inside the first interval",
			schedule:    []models.UserTelegramQuestionsScheduleItem{{From: "09:00", To: "18:00"}},
			midnightOff: 100,
			wantMinutes: 640,
		},
		{
			name:        "offset lands on the last inclusive minute",
			schedule:    []models.UserTelegramQuestionsScheduleItem{{From: "09:00", To: "18:00"}},
			midnightOff: 540,
			wantMinutes: 1080,
		},
		{
			name: "invalid intervals are skipped during mapping",
			schedule: []models.UserTelegramQuestionsScheduleItem{
				{From: "22:00", To: "06:00"},
				{From: "10:00", To: "11:00"},
			},
			midnightOff: 0,
			wantMinutes: 600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMinutes, MapOffsetOnSchedule(tt.schedule, tt.midnightOff))
		})
	}
}
