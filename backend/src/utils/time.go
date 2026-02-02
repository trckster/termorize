package utils

import "time"

func WasWithin(timestampMs int64, duration time.Duration) bool {
	t := time.UnixMilli(timestampMs)

	return time.Since(t) < duration
}
