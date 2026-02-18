package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
)

var (
	baseLogger *zap.Logger
	sugar      *zap.SugaredLogger
	initOnce   sync.Once
)

func initLogger() {
	if os.Getenv("ENV") == "prod" {
		baseLogger = zap.Must(zap.NewProduction())
	} else {
		baseLogger = zap.Must(zap.NewDevelopment())
	}

	sugar = baseLogger.Sugar()
}

func L() *zap.SugaredLogger {
	initOnce.Do(initLogger)
	return sugar
}

func Sync() {
	if baseLogger == nil {
		return
	}

	_ = baseLogger.Sync()
}
