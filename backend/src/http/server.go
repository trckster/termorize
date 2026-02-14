package http

import (
	"net/http"
	"termorize/src/config"
	"termorize/src/controllers"
	"termorize/src/http/middlewares"
	"termorize/src/http/validators"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func LaunchServer() {
	router := gin.Default()
	router.SetTrustedProxies(nil)

	registerValidators()

	router.Use(middlewares.CorsMiddleware())

	apiGroup := router.Group("/api")
	definePublicRoutes(apiGroup)

	protectedApiGroup := apiGroup.Group("")
	protectedApiGroup.Use(middlewares.AuthMiddleware())
	defineProtectedRoutes(protectedApiGroup)

	router.Run(":" + config.GetPort())
}

func registerValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("enum", validators.ValidateEnum)
		v.RegisterValidation("timezone", validators.ValidateTimezone)
		v.RegisterValidation("hhmm", validators.ValidateHHMM)
	}
}

func defineProtectedRoutes(group *gin.RouterGroup) {
	group.GET("/me", controllers.Me)
	group.PUT("/settings", controllers.UpdateSettings)

	group.GET("/vocabulary", controllers.GetVocabulary)
	group.POST("/vocabulary", controllers.CreateVocabulary)
	group.DELETE("/vocabulary/:id", controllers.DeleteVocabulary)

	group.POST("/translate", controllers.Translate)
}

func definePublicRoutes(group *gin.RouterGroup) {
	group.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "nice"})
	})

	group.POST("/telegram/login", controllers.TelegramLogin)
	group.POST("/logout", controllers.Logout)

	group.GET("/settings", controllers.GetSettings)
}
