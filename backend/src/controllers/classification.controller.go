package controllers

import (
	"crypto/subtle"
	"net/http"
	"termorize/src/classification"
	"termorize/src/config"

	"github.com/gin-gonic/gin"
)

func RequestClassificationSweep(c *gin.Context) {
	expected := "Bearer " + classification.TriggerToken(config.GetSecret())
	if subtle.ConstantTimeCompare([]byte(c.GetHeader("Authorization")), []byte(expected)) != 1 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if !classification.RequestSweep() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "classifier unavailable"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "requested"})
}
