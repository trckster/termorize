package monitoring

import (
	"context"
	"time"

	"termorize/src/config"
	"termorize/src/logger"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

const (
	flushTimeout     = 2 * time.Second
	tracesSampleRate = 0.0
)

var enabled bool

func Init() {
	dsn := config.GetSentryDSN()
	if dsn == "" {
		logger.L().Infow("sentry disabled, no DSN configured")
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      config.GetEnv(),
		EnableTracing:    tracesSampleRate > 0,
		TracesSampleRate: tracesSampleRate,
	})
	if err != nil {
		logger.L().Errorw("sentry initialization failed", "error", err)
		return
	}

	enabled = true
	logger.L().Infow("sentry initialized", "environment", config.GetEnv())
}

func Middleware() gin.HandlerFunc {
	return sentrygin.New(sentrygin.Options{Repanic: true})
}

func CaptureException(c *gin.Context, err error) {
	if !enabled || err == nil {
		return
	}

	hub(c).CaptureException(err)
}

func Recover(c *gin.Context, recovered any) {
	if !enabled || recovered == nil {
		return
	}

	hub(c).RecoverWithContext(requestContext(c), recovered)
}

func Flush() {
	if !enabled {
		return
	}

	sentry.Flush(flushTimeout)
}

func hub(c *gin.Context) *sentry.Hub {
	if c != nil {
		if requestHub := sentrygin.GetHubFromContext(c); requestHub != nil {
			return requestHub
		}
	}

	return sentry.CurrentHub()
}

func requestContext(c *gin.Context) context.Context {
	if c != nil {
		return c
	}

	return context.Background()
}
