package controllers

import (
	nethttp "net/http"
	"termorize/src/services"

	"github.com/gin-gonic/gin"
)

func GetExerciseStatistics(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	statistics, err := services.GetExerciseStatistics(userID)
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, statistics)
}
