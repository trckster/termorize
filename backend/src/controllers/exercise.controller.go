package controllers

import (
	nethttp "net/http"
	"strconv"
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

func GetExercises(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	page := 1
	pageSize := 50

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil {
			pageSize = parsed
		}
	}

	response, err := services.GetExercises(userID, page, pageSize)
	if err != nil {
		if services.InvalidPaginationError(err) {
			c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, response)
}
