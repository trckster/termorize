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
	}
}

func defineProtectedRoutes(group *gin.RouterGroup) {
	group.GET("/me", controllers.Me)
	group.GET("/vocabulary", controllers.GetVocabulary)
	group.POST("/vocabulary", controllers.CreateVocabulary)
	group.DELETE("/vocabulary/:id", controllers.DeleteVocabulary)
}

func definePublicRoutes(group *gin.RouterGroup) {
	group.POST("/telegram/login", controllers.TelegramLogin)
	group.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "nice"})
	})
	group.POST("/logout", controllers.Logout)
}
