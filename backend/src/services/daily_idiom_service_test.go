package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDailyIdiomDate(t *testing.T) {
	for _, tc := range []struct{ name, instant, timezone, date string }{
		{"before Rome midnight", "2026-09-23T21:59:59Z", "Europe/Rome", "2026-09-23"},
		{"at Rome midnight", "2026-09-23T22:00:00Z", "Europe/Rome", "2026-09-24"},
		{"previous date", "2026-09-23T00:00:00Z", "America/Los_Angeles", "2026-09-22"},
		{"next date", "2026-09-23T10:00:00Z", "Pacific/Kiritimati", "2026-09-24"},
		{"DST spring", "2026-03-29T01:00:00Z", "Europe/Rome", "2026-03-29"},
		{"DST fall", "2026-10-25T01:00:00Z", "Europe/Rome", "2026-10-25"},
		{"missing timezone", "2026-09-23T23:00:00Z", "", "2026-09-23"},
		{"invalid timezone", "2026-09-23T23:00:00Z", "invalid/timezone", "2026-09-23"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.instant)
			assert.NoError(t, err)
			assert.Equal(t, tc.date, dailyIdiomDate(now, tc.timezone))
		})
	}
}
