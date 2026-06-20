package testkit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"termorize/src/integrations/telegram"
)

// TelegramRequest is a single outbound call captured by FakeTelegramServer. Action
// is the Telegram method name (e.g. "sendMessage"), and Body is the raw JSON request
// body the handler sent.
type TelegramRequest struct {
	Action string
	Body   []byte
}

// FakeTelegramServer is an in-process httptest.Server that stands in for the real
// Telegram Bot API. It returns canned, valid `{"ok":true,...}` responses for the
// actions the webhook handlers call and records every request it receives so tests
// can assert on outbound side effects (e.g. that a reply was sent).
type FakeTelegramServer struct {
	server *httptest.Server

	mu       sync.Mutex
	requests []TelegramRequest
}

// MockTelegramAPI starts a FakeTelegramServer, points the telegram package's API
// base URL at it for the duration of the test, and restores the original base URL
// (and shuts the server down) via t.Cleanup. No outbound call made through
// telegram.CallAPI will reach the real network while it is installed.
//
//	tg := testkit.MockTelegramAPI(t)
//	// ... drive the webhook ...
//	require.True(t, tg.Sent("sendMessage"))
func MockTelegramAPI(t *testing.T) *FakeTelegramServer {
	t.Helper()

	fake := &FakeTelegramServer{}
	fake.server = httptest.NewServer(http.HandlerFunc(fake.handle))

	restore := telegram.SetAPIBaseURLForTest(fake.server.URL)
	t.Cleanup(func() {
		restore()
		fake.server.Close()
	})

	return fake
}

// handle records the request and replies with a canned valid response for the
// requested action. URLs look like /bot<token>/<action>.
func (f *FakeTelegramServer) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	action := r.URL.Path
	if idx := strings.LastIndex(action, "/"); idx >= 0 {
		action = action[idx+1:]
	}

	f.mu.Lock()
	f.requests = append(f.requests, TelegramRequest{Action: action, Body: body})
	f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	// Every Telegram response is shaped `{"ok":true,"result":...}`. A result object
	// that carries a message_id satisfies the response types that read it back
	// (e.g. sendMessage), and is harmless for the actions that ignore result.
	resp := map[string]any{
		"ok": true,
		"result": map[string]any{
			"message_id": 1,
			"date":       0,
			"chat":       map[string]any{"id": 1, "type": "private"},
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// Requests returns a copy of all captured outbound requests, in order.
func (f *FakeTelegramServer) Requests() []TelegramRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]TelegramRequest, len(f.requests))
	copy(out, f.requests)
	return out
}

// RequestsFor returns every captured request whose action matches the given name.
func (f *FakeTelegramServer) RequestsFor(action string) []TelegramRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []TelegramRequest
	for _, req := range f.requests {
		if req.Action == action {
			out = append(out, req)
		}
	}
	return out
}

// Sent reports whether at least one request was made for the given action.
func (f *FakeTelegramServer) Sent(action string) bool {
	return len(f.RequestsFor(action)) > 0
}

// Count returns how many requests were made for the given action.
func (f *FakeTelegramServer) Count(action string) int {
	return len(f.RequestsFor(action))
}
