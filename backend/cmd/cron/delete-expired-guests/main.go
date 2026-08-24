package main

import _ "termorize/src/utils"

import (
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/logger"
	"termorize/src/monitoring"
	"termorize/src/services"
	"time"
)

func main() {
	defer logger.Sync()
	config.LoadEnv()

	monitoring.Init()
	defer monitoring.Flush()

	if err := db.Connect(); err != nil {
		fatal("database connection failed", err)
	}

	deleted, err := services.DeleteExpiredGuestUsers(time.Now().UTC())
	if err != nil {
		fatal("expired guest account deletion failed", err)
	}

	logger.L().Infow("expired guest accounts deleted", "count", deleted)
}

func fatal(message string, err error) {
	monitoring.CaptureException(nil, err)
	monitoring.Flush()
	logger.L().Fatalw(message, "error", err)
}
