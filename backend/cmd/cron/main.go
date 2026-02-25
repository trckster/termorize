package main

// Import first to set UTC timezone before any other package uses invalid timezone
import _ "termorize/src/utils"

import (
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/logger"
	"termorize/src/services"
)

func main() {
	defer logger.Sync()
	config.LoadEnv()

	if err := db.Connect(); err != nil {
		logger.L().Fatalw("database connection failed", "error", err)
	}

	if err := db.Migrate(); err != nil {
		logger.L().Fatalw("migration failed", "error", err)
	}

	if err := services.GenerateDailyExercises(); err != nil {
		logger.L().Fatalw("generation of daily exercises failed", "error", err)
	}
}
