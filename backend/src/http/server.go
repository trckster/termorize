package http

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"termorize/src/config"
	"termorize/src/controllers"
	"termorize/src/http/middlewares"
	"termorize/src/http/validators"
	"termorize/src/monitoring"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func BuildRouter() *gin.Engine {
	if config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.SetTrustedProxies(nil)

	registerCustomValidators()

	router.Use(monitoring.Middleware())
	router.Use(middlewares.RequestLoggerMiddleware())
	router.Use(middlewares.RecoveryMiddleware())
	router.Use(middlewares.CorsMiddleware())

	apiGroup := router.Group("/api")
	definePublicRoutes(apiGroup)

	protectedApiGroup := apiGroup.Group("")
	protectedApiGroup.Use(middlewares.AuthMiddleware())
	defineProtectedRoutes(protectedApiGroup)

	return router
}

func LaunchServer(ctx context.Context) error {
	server := &http.Server{Addr: ":" + config.GetPort(), Handler: BuildRouter(), ReadHeaderTimeout: 10 * time.Second}
	shutdownDone := make(chan struct{})
	serverCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		defer close(shutdownDone)
		<-serverCtx.Done()
		shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
		}
	}()
	err := server.ListenAndServe()
	cancel()
	<-shutdownDone
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

var registerValidatorsOnce sync.Once

func registerCustomValidators() {
	registerValidatorsOnce.Do(func() {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			v.RegisterValidation("enum", validators.ValidateEnum)
			v.RegisterValidation("timezone", validators.ValidateTimezone)
			v.RegisterValidation("hhmm", validators.ValidateHHMM)
			v.RegisterStructValidation(validators.ValidateHHMMInterval, controllers.UpdateSettingsTelegramScheduleItemRequest{})
		}
	})
}
