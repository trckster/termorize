package main

// Import first to set UTC timezone before any other package uses invalid timezone
import _ "termorize/src/utils"

import (
	"log"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/http"
)

func main() {
	config.LoadEnv()

	if err := db.Connect(); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := db.Migrate(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	http.LaunchServer()
}
