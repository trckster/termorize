package telegram

import (
	"errors"
	"net/http"
	"testing"
)

func TestParseAPIErrorReturnsStructuredError(t *testing.T) {
	err := parseAPIError(
		http.StatusBadRequest,
		[]byte(`{"ok":false,"error_code":400,"description":"Bad Request: query is too old and response timeout expired or query ID is invalid"}`),
	)

	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *apiError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
	}
	if apiErr.ErrorCode != 400 {
		t.Errorf("ErrorCode = %d, want 400", apiErr.ErrorCode)
	}
	if !isExpiredCallbackQueryError(err) {
		t.Error("expected error to be classified as an expired callback query")
	}
}

func TestIsExpiredCallbackQueryErrorRejectsOtherErrors(t *testing.T) {
	tests := []error{
		errors.New("query is too old"),
		&apiError{StatusCode: http.StatusBadRequest, ErrorCode: 400, Description: "Bad Request: chat not found"},
		&apiError{StatusCode: http.StatusBadRequest, ErrorCode: 401, Description: "Bad Request: query is too old"},
	}

	for _, err := range tests {
		if isExpiredCallbackQueryError(err) {
			t.Errorf("unexpected expired callback classification for %v", err)
		}
	}
}
