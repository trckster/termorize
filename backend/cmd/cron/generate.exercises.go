package main

// Import first to set UTC timezone before any other package uses invalid timezone
import _ "termorize/src/utils"

import (
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/logger"
	"termorize/src/monitoring"
	"termorize/src/services"
)

func main() {
	defer logger.Sync()
	config.LoadEnv()

	monitoring.Init()
	defer monitoring.Flush()

	if err := db.Connect(); err != nil {
		fatal("database connection failed", err)
	}

	if err := services.GenerateDailyExercises(); err != nil {
		fatal("generation of daily exercises failed", err)
	}
}

func fatal(message string, err error) {
	monitoring.CaptureException(nil, err)
	monitoring.Flush()
	logger.L().Fatalw(message, "error", err)
}
