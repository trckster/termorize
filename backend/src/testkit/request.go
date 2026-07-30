package testkit

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

	return serveRequest(t, method, path, reader, body != nil, nil, cookies...)
}

func RequestWithHeaders(
	t *testing.T,
	method,
	path string,
	body any,
	headers http.Header,
	cookies ...*http.Cookie,
) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("testkit.RequestWithHeaders: failed to marshal body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}

	return serveRequest(t, method, path, reader, body != nil, headers, cookies...)
}

func RawRequest(
	t *testing.T,
	method,
	path,
	body string,
	headers http.Header,
	cookies ...*http.Cookie,
) *httptest.ResponseRecorder {
	t.Helper()

	return serveRequest(t, method, path, strings.NewReader(body), true, headers, cookies...)
}

func serveRequest(
	t *testing.T,
	method,
	path string,
	body io.Reader,
	hasBody bool,
	headers http.Header,
	cookies ...*http.Cookie,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, body)
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
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

func RequireStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int, context ...string) {
	t.Helper()

	if rec.Code == expected {
		return
	}

	detail := ""
	if len(context) > 0 && context[0] != "" {
		detail = ": " + context[0]
	}
	t.Fatalf(
		"unexpected HTTP status%s: got %d (%s), want %d (%s); body=%q",
		detail,
		rec.Code,
		http.StatusText(rec.Code),
		expected,
		http.StatusText(expected),
		rec.Body.String(),
	)
}

func DecodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()

	contentType := rec.Header().Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		t.Fatalf("testkit.DecodeJSON: response Content-Type = %q, want application/json", contentType)
	}

	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("testkit.DecodeJSON: failed to decode response body %q: %v", rec.Body.String(), err)
	}
}
