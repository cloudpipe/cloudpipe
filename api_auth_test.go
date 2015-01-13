package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverNullAuthService(t *testing.T) {
	r, err := http.NewRequest("GET", "https://localhost/v1/auth_service", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	w := httptest.NewRecorder()
	c := &Context{
		AuthService: NullAuthService{},
	}

	AuthDiscoverHandler(c, w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected HTTP status: [%d]", w.Code)
	}

	var response struct {
		Address string `json:"address"`
		Style   string `json:"style"`
	}
	out := w.Body.Bytes()
	if err := json.Unmarshal(out, &response); err != nil {
		t.Fatalf("Unable to parse response body as JSON: [%s]", string(out))
	}

	if response.Address != "" {
		t.Errorf("Unexpected auth service address: %s", response.Address)
	}
	if response.Style != "null" {
		t.Errorf("Unexpected auth service style: %s", response.Style)
	}
}

func TestDiscoverRemoteAuthService(t *testing.T) {
	r, err := http.NewRequest("GET", "https://localhost/v1/auth_service", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	w := httptest.NewRecorder()
	c := &Context{
		Settings: Settings{AuthService: "https://somewhere.com/"},
		AuthService: RemoteAuthService{
			ValidateURL: "https://somewhere.com/validate",
		},
	}

	AuthDiscoverHandler(c, w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected HTTP status: [%d]", w.Code)
	}

	var response struct {
		Address string `json:"address"`
		Style   string `json:"style"`
	}
	out := w.Body.Bytes()
	if err := json.Unmarshal(out, &response); err != nil {
		t.Fatalf("Unable to parse response body as JSON: [%s]", string(out))
	}

	if response.Address != "https://somewhere.com/" {
		t.Errorf("Unexpected auth service address: %s", response.Address)
	}
	if response.Style != "local" {
		t.Errorf("Unexpected auth service style: %s", response.Style)
	}
}
