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

type TelegramRequest struct {
	Action string
	Body   []byte
}

type FakeTelegramServer struct {
	server *httptest.Server

	mu       sync.Mutex
	requests []TelegramRequest
}

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

func (f *FakeTelegramServer) Requests() []TelegramRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]TelegramRequest, len(f.requests))
	copy(out, f.requests)
	return out
}

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

func (f *FakeTelegramServer) Sent(action string) bool {
	return len(f.RequestsFor(action)) > 0
}

func (f *FakeTelegramServer) Count(action string) int {
	return len(f.RequestsFor(action))
}
