package tests

import (
	"net/http"
	"testing"

	"termorize/src/models"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPing proves the router is wired and serves the public health endpoint.
func TestPing(t *testing.T) {
	rec := testkit.Request(t, http.MethodGet, "/api/ping", nil)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]string
	testkit.DecodeJSON(t, rec, &body)
	assert.Equal(t, "nice", body["status"])
}

// TestMeRequiresAuth proves the auth middleware rejects unauthenticated requests.
func TestMeRequiresAuth(t *testing.T) {
	testkit.Truncate(t)

	rec := testkit.Request(t, http.MethodGet, "/api/me", nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestMeWithAuth proves DB + router + auth + truncation work together: a user is
// created in the (truncated) test DB, authenticated via a real JWT cookie, and
// returned by the protected /api/me endpoint.
func TestMeWithAuth(t *testing.T) {
	testkit.Truncate(t)

	user := testkit.CreateUser(t, testkit.WithName("Ada Lovelace"))

	rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/me", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var got models.User
	testkit.DecodeJSON(t, rec, &got)
	assert.Equal(t, user.ID, got.ID)
	assert.Equal(t, "Ada Lovelace", got.Name)
}
