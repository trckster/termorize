package runners

import (
	"context"
	"termorize/src/classification"
	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/integrations/jev"
)

func StartClassificationRunner(ctx context.Context) func() {
	client := jev.NewClient(config.GetOpenRouterApiKey())
	worker := classification.NewWorker(classification.WordStore{DB: db.DB}, client.Classify)
	restoreActive := classification.Activate(worker)
	restoreEvents := db.SetWordCreatedHandler(worker.Enqueue)
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Run(ctx)
	}()
	return func() {
		<-done
		restoreEvents()
		restoreActive()
	}
}
