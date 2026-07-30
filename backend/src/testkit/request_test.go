package testkit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestWithHeadersPreservesExplicitContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previousRouter := router
	t.Cleanup(func() {
		router = previousRouter
	})

	router = gin.New()
	router.POST("/content-type", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetHeader("Content-Type"))
	})

	rec := RequestWithHeaders(
		t,
		http.MethodPost,
		"/content-type",
		map[string]string{"value": "test"},
		http.Header{"Content-Type": []string{"text/plain"}},
	)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/plain", rec.Body.String())
}

func TestFinalizedContentTypeIgnoresLateHeaderMutation(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusOK)
	rec.Header().Set("Content-Type", "application/json")

	assert.Empty(t, finalizedContentType(rec))
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}
