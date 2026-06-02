package controllers

import (
	"net/http"
	"termorize/src/logger"
	"termorize/src/monitoring"

	"github.com/gin-gonic/gin"
)

func ServerError(c *gin.Context, err error) {
	logger.L().Errorw(
		"server error",
		"error", err.Error(),
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)
	monitoring.CaptureException(c, err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
}
