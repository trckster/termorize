package http

import (
	"termorize/src/config"
	"termorize/src/http/middlewares"
	"termorize/src/http/validators"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func LaunchServer() {
	if config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.SetTrustedProxies(nil)

	registerCustomValidators()

	router.Use(middlewares.CorsMiddleware())

	apiGroup := router.Group("/api")
	definePublicRoutes(apiGroup)

	protectedApiGroup := apiGroup.Group("")
	protectedApiGroup.Use(middlewares.AuthMiddleware())
	defineProtectedRoutes(protectedApiGroup)

	router.Run(":" + config.GetPort())
}

func registerCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("enum", validators.ValidateEnum)
		v.RegisterValidation("timezone", validators.ValidateTimezone)
		v.RegisterValidation("hhmm", validators.ValidateHHMM)
	}
}
