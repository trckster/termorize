package main

// Import first to set UTC timezone before any other package uses invalid timezone
import _ "termorize/src/utils"

import (
	"flag"
	"strconv"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/data/seeders"
	"termorize/src/logger"
)

func main() {
	defer logger.Sync()

	userIDFlag := flag.String("uid", "", "User ID to seed vocabulary for (optional)")
	flag.Parse()

	config.LoadEnv()

	if err := db.Connect(); err != nil {
		logger.L().Fatalw("database connection failed", "error", err)
	}

	if err := db.Migrate(); err != nil {
		logger.L().Fatalw("migration failed", "error", err)
	}

	req := seeders.VocabularySeedRequest{}

	if *userIDFlag != "" {
		userID, err := strconv.ParseUint(*userIDFlag, 10, 32)
		if err != nil {
			logger.L().Fatalw("invalid user id", "error", err)
		}
		id := uint(userID)
		req.UserID = &id
	}

	if err := seeders.SeedVocabulary(req); err != nil {
		logger.L().Fatalw("seeding failed", "error", err)
	}
}
