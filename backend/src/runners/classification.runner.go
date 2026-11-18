package runners

import (
	"context"
	"termorize/src/classification"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/integrations/jev"
	"termorize/src/services"
)

func StartClassificationRunner(ctx context.Context) func() {
	client := jev.NewClient(config.GetOpenRouterApiKey())
	worker := classification.NewWorker(classification.WordStore{DB: db.DB}, client.Classify)
	restoreEvents := db.SetWordCreatedHandler(worker.Enqueue)
	var available *classification.Worker
	if config.GetOpenRouterApiKey() != "" {
		available = worker
		worker.RequestSweep()
	}
	restoreWorker := services.SetCategorizationWorker(available)
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Run(ctx)
	}()
	return func() {
		<-done
		restoreEvents()
		restoreWorker()
	}
}
