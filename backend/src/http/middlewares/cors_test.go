package middlewares

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveAllowedCorsOrigin(t *testing.T) {
	const publicURL = "https://termorize.daniil.online/"

	tests := []struct {
		name          string
		requestOrigin string
		isLocal       bool
		expected      string
	}{
		{name: "production app", requestOrigin: "https://termorize.daniil.online", expected: "https://termorize.daniil.online"},
		{name: "Google Translate", requestOrigin: googleTranslateOrigin, expected: googleTranslateOrigin},
		{name: "untrusted production origin", requestOrigin: "https://example.com", expected: ""},
		{name: "missing production origin", expected: ""},
		{name: "local browser origin", requestOrigin: "http://localhost:5173", isLocal: true, expected: "http://localhost:5173"},
		{name: "local non-browser client", isLocal: true, expected: "*"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, resolveAllowedCorsOrigin(publicURL, test.requestOrigin, test.isLocal))
		})
	}
}
