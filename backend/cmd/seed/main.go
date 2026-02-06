package main

import (
	"flag"
	"log"
	"strconv"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/data/seeders"
)

func main() {
	userIDFlag := flag.String("uid", "", "User ID to seed vocabulary for (optional)")
	flag.Parse()

	config.LoadEnv()

	if err := db.Connect(); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := db.Migrate(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	req := seeders.VocabularySeedRequest{}

	if *userIDFlag != "" {
		userID, err := strconv.ParseUint(*userIDFlag, 10, 32)
		if err != nil {
			log.Fatalf("Invalid user ID: %v", err)
		}
		id := uint(userID)
		req.UserID = &id
	}

	if err := seeders.SeedVocabulary(req); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}
}
