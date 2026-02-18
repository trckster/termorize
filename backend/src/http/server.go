package http

import (
	"termorize/src/config"
	"termorize/src/http/middlewares"
	"termorize/src/http/validators"
	"termorize/src/logger"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func LaunchServer() {
	if config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.SetTrustedProxies(nil)

	registerCustomValidators()

	router.Use(middlewares.RequestLoggerMiddleware())
	router.Use(middlewares.RecoveryMiddleware())
	router.Use(middlewares.CorsMiddleware())

	apiGroup := router.Group("/api")
	definePublicRoutes(apiGroup)

	protectedApiGroup := apiGroup.Group("")
	protectedApiGroup.Use(middlewares.AuthMiddleware())
	defineProtectedRoutes(protectedApiGroup)

	if err := router.Run(":" + config.GetPort()); err != nil {
		logger.L().Fatalw("failed to start http server", "error", err)
	}
}

func registerCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("enum", validators.ValidateEnum)
		v.RegisterValidation("timezone", validators.ValidateTimezone)
		v.RegisterValidation("hhmm", validators.ValidateHHMM)
	}
}
