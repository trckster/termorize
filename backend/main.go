package main

// Import first to set UTC timezone before any other package uses invalid timezone
import _ "termorize/src/utils"

import (
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/http"
	"termorize/src/integrations/telegram"
	"termorize/src/logger"
	"termorize/src/monitoring"
	"termorize/src/runners"
)

func main() {
	defer logger.Sync()

	config.LoadEnv()

	monitoring.Init()
	defer monitoring.Flush()

	if err := db.Connect(); err != nil {
		fatal("database connection failed", err)
	}

	if err := db.Migrate(); err != nil {
		fatal("migration failed", err)
	}

	if err := telegram.SetupWebhook(); err != nil {
		fatal("telegram webhook setup failed", err)
	}

	runners.StartExerciseRunner()

	http.LaunchServer()
}

func fatal(message string, err error) {
	monitoring.CaptureException(nil, err)
	monitoring.Flush()
	logger.L().Fatalw(message, "error", err)
}
