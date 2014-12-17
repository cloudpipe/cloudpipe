package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJobHandlerBadRequest(t *testing.T) {
	r, err := http.NewRequest("PUT", "https://localhost/api/jobs", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	w := httptest.NewRecorder()
	c := &Context{}

	JobHandler(c, w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Unexpected HTTP status: [%d]", w.Code)
	}
	if contentType := w.HeaderMap.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Incorrect or missing content-type header: [%s]", contentType)
	}

	var e struct {
		Error RhoError
	}
	body := w.Body.Bytes()
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("Unable to parse response body as JSON: %s", body)
	}

	if e.Error.Code != "3" {
		t.Errorf("Unexpected error code: [%s]", e.Error.Code)
	}
	if e.Error.Message != "Method not supported" {
		t.Errorf("Unexpected error message: [%s]", e.Error.Message)
	}
	if e.Error.Retry {
		t.Errorf("Retry is set to true and should be false.")
	}
}

func TestSubmitJob(t *testing.T) {
	body := strings.NewReader(`
	{
		"jobs": [{
			"cmd": "id",
			"name": "wat",
			"result_source": "stdout",
			"result_type": "binary"
		}]
	}
	`)
	r, err := http.NewRequest("POST", "https://localhost/api/jobs", body)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	r.SetBasicAuth("admin", "12345")
	w := httptest.NewRecorder()
	c := &Context{
		Settings: Settings{
			AdminName: "admin",
			AdminKey:  "12345",
		},
	}

	JobHandler(c, w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected HTTP status: [%d]", w.Code)
	}

	var response struct {
		JIDs []uint `json:"jids"`
	}
	out := w.Body.Bytes()
	if err := json.Unmarshal(out, &response); err != nil {
		t.Fatalf("Unable to parse response body as JSON: [%s]", string(out))
	}
	if len(response.JIDs) != 1 {
		t.Errorf("Expected one JID, received [%d]", len(response.JIDs))
	}
	if response.JIDs[0] != 0 {
		t.Errorf("Expected to be assigned ID 0, got [%d]", response.JIDs[0])
	}
}

func TestSubmitJobBadResultSource(t *testing.T) {
	body := strings.NewReader(`
	{
		"jobs": [{
			"cmd": "id",
			"name": "wat",
			"result_source": "magic",
			"result_type": "binary"
		}]
	}
	`)
	r, err := http.NewRequest("POST", "https://localhost/api/jobs", body)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	r.SetBasicAuth("admin", "12345")
	w := httptest.NewRecorder()
	c := &Context{
		Settings: Settings{
			AdminName: "admin",
			AdminKey:  "12345",
		},
	}

	JobHandler(c, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected a bad request, got [%d]", w.Code)
	}

	var e struct {
		Error RhoError
	}
	out := w.Body.Bytes()
	if err := json.Unmarshal(out, &e); err != nil {
		t.Fatalf("Unable to parse response body as JSON: [%s]", string(out))
	}

	if e.Error.Code != "6" {
		t.Errorf("Unexpected error code: [%s]", e.Error.Code)
	}
	if e.Error.Message != "Invalid result_source." {
		t.Errorf("Unexpected error message: [%s]", e.Error.Message)
	}
	if e.Error.Retry {
		t.Errorf("Retry is set to true and should be false.")
	}
}

func TestSubmitJobBadResultType(t *testing.T) {
	body := strings.NewReader(`
	{
		"jobs": [{
			"cmd": "id",
			"name": "wat",
			"result_source": "stdout",
			"result_type": "elsewhere"
		}]
	}
	`)
	r, err := http.NewRequest("POST", "https://localhost/api/jobs", body)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	r.SetBasicAuth("admin", "12345")
	w := httptest.NewRecorder()
	c := &Context{
		Settings: Settings{
			AdminName: "admin",
			AdminKey:  "12345",
		},
	}

	JobHandler(c, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected a bad request, got [%d]", w.Code)
	}

	var e struct {
		Error RhoError
	}
	out := w.Body.Bytes()
	if err := json.Unmarshal(out, &e); err != nil {
		t.Fatalf("Unable to parse response body as JSON: [%s]", string(out))
	}

	if e.Error.Code != "7" {
		t.Errorf("Unexpected error code: [%s]", e.Error.Code)
	}
	if e.Error.Message != "Invalid result_type." {
		t.Errorf("Unexpected error message: [%s]", e.Error.Message)
	}
	if e.Error.Retry {
		t.Errorf("Retry is set to true and should be false.")
	}
}
