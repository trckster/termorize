package http

import (
	"sync"
	"termorize/src/config"
	"termorize/src/http/middlewares"
	"termorize/src/http/validators"
	"termorize/src/logger"
	"termorize/src/monitoring"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// BuildRouter constructs the fully-configured Gin engine (middleware, custom
// validators and all routes) WITHOUT starting the HTTP server. It is the single
// source of truth for the application's routing and is safe to call from tests.
//
// Custom validator registration is guarded so repeated calls do not register the
// same validators multiple times.
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

func LaunchServer() {
	router := BuildRouter()

	if err := router.Run(":" + config.GetPort()); err != nil {
		logger.L().Fatalw("failed to start http server", "error", err)
	}
}

var registerValidatorsOnce sync.Once

func registerCustomValidators() {
	registerValidatorsOnce.Do(func() {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			v.RegisterValidation("enum", validators.ValidateEnum)
			v.RegisterValidation("timezone", validators.ValidateTimezone)
			v.RegisterValidation("hhmm", validators.ValidateHHMM)
		}
	})
}
