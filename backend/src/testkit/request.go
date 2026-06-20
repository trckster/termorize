package testkit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Request performs an in-process HTTP request against the shared router and
// returns the recorder. No real network is used (router.ServeHTTP is called
// directly).
//
// If body is non-nil it is JSON-encoded and the Content-Type header is set to
// application/json. Any cookies provided are attached to the request (use
// AuthCookie to authenticate).
//
//	rec := testkit.Request(t, http.MethodPost, "/api/translate",
//	    map[string]string{"text": "hello"}, testkit.AuthCookie(user))
func Request(t *testing.T, method, path string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("testkit.Request: failed to marshal body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range cookies {
		if c != nil {
			req.AddCookie(c)
		}
	}

	rec := httptest.NewRecorder()
	Router().ServeHTTP(rec, req)
	return rec
}

// DecodeJSON unmarshals a recorder's JSON body into dst, failing the test on
// error. dst must be a pointer.
//
//	var user models.User
//	testkit.DecodeJSON(t, rec, &user)
func DecodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("testkit.DecodeJSON: failed to decode response body %q: %v", rec.Body.String(), err)
	}
}
