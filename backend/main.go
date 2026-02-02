package main

// Import first to set UTC timezone before any other package uses invalid timezone
import _ "termorize/src/utils"

import (
	"log"
	"termorize/src/config"
	"termorize/src/database"
	"termorize/src/http"
)

func main() {
	config.LoadEnv()

	if err := database.Connect(); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.Migrate(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	http.LaunchServer()
}
