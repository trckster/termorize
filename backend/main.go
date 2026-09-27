package main

// Import first to set UTC timezone before any other package uses invalid timezone
import _ "termorize/src/utils"

import (
	"context"
	"os"
	"os/signal"
	"syscall"
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

	if err := runBackend(); err != nil {
		fatal("http server stopped", err)
	}
}

func runBackend() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	waitClassifier := runners.StartClassificationRunner(ctx)
	defer func() { stop(); waitClassifier() }()
	runners.StartExerciseRunner()
	runners.StartDailyIdiomRunner(ctx)
	runners.StartDictionaryRunner(ctx)

	return http.LaunchServer(ctx)
}

func fatal(message string, err error) {
	monitoring.CaptureException(nil, err)
	monitoring.Flush()
	logger.L().Fatalw(message, "error", err)
}
