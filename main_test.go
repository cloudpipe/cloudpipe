package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func hasError(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErr APIError) {
	if w.Code != expectedStatus {
		t.Errorf("Unexpected HTTP status: wanted [%d], got [%d]", expectedStatus, w.Code)
	}
	if contentType := w.HeaderMap.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Incorrect or missing content-type header: [%s]", contentType)
	}

	var e struct {
		Error APIError
	}
	body := w.Body.Bytes()
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("Unable to parse response body as JSON: [%s]", string(body))
	}

	if e.Error.Code != expectedErr.Code {
		t.Errorf("Unexpected error code: [%s]", e.Error.Code)
	}
	if e.Error.Message != expectedErr.Message {
		t.Errorf("Unexpected error message: [%s]", e.Error.Message)
	}
	if e.Error.Retry != expectedErr.Retry {
		t.Errorf("Retry is set to true and should be false.")
	}
}
