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

// UseNop replaces the logger with a no-op one that discards all output. It is
// intended for tests, where log lines would otherwise pollute test output. Call
// it before any other code touches the logger so that initLogger never runs.
func UseNop() {
	initOnce.Do(func() {})
	baseLogger = zap.NewNop()
	sugar = baseLogger.Sugar()
}

func Sync() {
	if baseLogger == nil {
		return
	}

	_ = baseLogger.Sync()
}
