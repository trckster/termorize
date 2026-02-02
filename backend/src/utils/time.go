package utils

import (
	"os"
	"time"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func WasWithin(timestampMs int64, duration time.Duration) bool {
	t := time.UnixMilli(timestampMs)

	return time.Since(t) < duration
}
