package controllers

import (
	"errors"
	nethttp "net/http"
	"strconv"
	"strings"
	"termorize/src/services"

	"github.com/google/uuid"

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

func RandomExercise(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	result, err := services.CreateRandomExercise(userID)
	if err != nil {
		if errors.Is(err, services.ErrNoEligibleVocabulary) {
			c.JSON(nethttp.StatusUnprocessableEntity, gin.H{"error": "no eligible vocabulary found"})
			return
		}

		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, gin.H{
		"exercise_id":     result.ExerciseID,
		"type":            result.Type,
		"question_word":   result.QuestionWord,
		"language":        result.Language,
		"answer_language": result.AnswerLanguage,
	})
}

func VerifyExercise(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	exerciseIDStr := c.Param("id")
	exerciseID, err := uuid.Parse(exerciseIDStr)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid exercise id"})
		return
	}

	var body struct {
		Answer string `json:"answer" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "answer is required"})
		return
	}

	result, err := services.VerifyExerciseAnswer(exerciseID, userID, body.Answer)
	if err != nil {
		if errors.Is(err, services.ErrExerciseNotFound) {
			c.JSON(nethttp.StatusNotFound, gin.H{"error": "exercise not found"})
			return
		}

		if errors.Is(err, services.ErrExerciseNotInProgress) {
			c.JSON(nethttp.StatusConflict, gin.H{"error": "exercise is not in progress"})
			return
		}

		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, gin.H{
		"result":         result.Result,
		"correct_answer": result.CorrectAnswer,
		"knowledge":      result.Knowledge,
	})
}

func GetExercisesByIDs(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	idsParam := c.Query("ids")
	if idsParam == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "ids parameter is required"})
		return
	}

	rawIDs := strings.Split(idsParam, ",")
	ids := make([]uuid.UUID, 0, len(rawIDs))

	for _, raw := range rawIDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid id: " + raw})
			return
		}

		ids = append(ids, id)
	}

	exercises, err := services.GetExercisesByIDs(userID, ids)
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, exercises)
}
