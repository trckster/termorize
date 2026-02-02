package http

import (
	"net/http"
	"termorize/src/config"
	"termorize/src/controllers"
	"termorize/src/http/middlewares"

	"github.com/gin-gonic/gin"
)

func LaunchServer() {
	router := gin.Default()
	router.SetTrustedProxies(nil)

	router.Use(middlewares.CorsMiddleware())

	apiGroup := router.Group("/api")
	definePublicRoutes(apiGroup)

	protectedApiGroup := apiGroup.Group("")
	protectedApiGroup.Use(middlewares.AuthMiddleware())
	defineProtectedRoutes(protectedApiGroup)

	router.Run(":" + config.GetPort())
}

func defineProtectedRoutes(group *gin.RouterGroup) {
	group.GET("/me", controllers.Me)
}

func definePublicRoutes(group *gin.RouterGroup) {
	group.POST("/telegram/login", controllers.TelegramLogin)
	group.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "nice"})
	})
	group.POST("/logout", controllers.Logout)
}
