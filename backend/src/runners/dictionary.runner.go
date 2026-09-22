package runners

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/logger"
	"termorize/src/services"
)

func StartDictionaryRunner(ctx context.Context) {
	identity := sha256.Sum256([]byte(config.GetDBHost() + ":" + config.GetDBPort() + "/" + config.GetDBName()))
	worker := &services.DictionaryWorker{
		DB:      db.DB,
		Client:  &http.Client{Timeout: 6 * time.Hour},
		TempDir: filepath.Join(os.TempDir(), fmt.Sprintf("termorize-dictionaries-%x", identity[:8])),
	}
	go func() {
		for ctx.Err() == nil {
			if err := worker.Run(ctx); err != nil {
				logger.L().Errorw("dictionary worker failed", "error", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}()
}
