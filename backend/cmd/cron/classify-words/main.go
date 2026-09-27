package main

import _ "termorize/src/utils"

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"termorize/src/classification"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/integrations/jev"
	"termorize/src/logger"
	"termorize/src/monitoring"
)

func main() {
	defer logger.Sync()
	config.LoadClassificationEnv()
	monitoring.Init()
	defer monitoring.Flush()

	if err := run(); err != nil {
		monitoring.CaptureException(nil, err)
		monitoring.Flush()
		logger.L().Fatalw("classification sweep failed", "error", err)
	}
	logger.L().Info("classification job finished")
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if config.GetOpenRouterApiKey() == "" {
		return errors.New("OPENROUTER_API_KEY is required")
	}
	if err := db.Connect(); err != nil {
		return err
	}
	connection, err := db.DB.DB()
	if err != nil {
		return err
	}
	defer connection.Close()
	client := jev.NewClient(config.GetOpenRouterApiKey())
	worker := classification.NewWorker(classification.WordStore{DB: db.DB}, client.Classify)
	return worker.RunSweep(ctx)
}
