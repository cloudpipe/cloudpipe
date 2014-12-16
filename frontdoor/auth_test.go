package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupAuthRecorder(t *testing.T, username, key string) (*http.Request, *httptest.ResponseRecorder) {
	r, err := http.NewRequest("GET", "https://localhost/api/jobs", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	if username != "" {
		r.SetBasicAuth(username, key)
	}
	w := httptest.NewRecorder()
	return r, w
}

func TestAuthenticateMissingCredentials(t *testing.T) {
	r, w := setupAuthRecorder(t, "", "")
	c := &Context{}

	_, err := Authenticate(c, w, r)
	if err == nil {
		t.Error("Expected Authenticate to return an error without authentication provided.")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Wrong HTTP status code: %d", w.Code)
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

	if e.Error.Code != "1" {
		t.Errorf("Unexpected error code: [%s]", e.Error.Code)
	}
	if e.Error.Message != "You must authenticate." {
		t.Errorf("Unexpected error message: [%s]", e.Error.Message)
	}
	if e.Error.Retry {
		t.Errorf("Retry is set to true and should be false.")
	}
}

func TestAuthenticateAdminCredentials(t *testing.T) {
	r, w := setupAuthRecorder(t, "admin", "12345edcba")
	c := &Context{
		Settings: Settings{
			AdminName: "admin",
			AdminKey:  "12345edcba",
		},
	}

	a, err := Authenticate(c, w, r)
	if err != nil {
		t.Fatalf("Unable to authenticate: %v", err)
	}

	if a.Name != "admin" {
		t.Errorf("Unexpected account name: [%s]", a.Name)
	}
	if a.APIKey != "12345edcba" {
		t.Errorf("Unexpected API key: [%s]", a.APIKey)
	}
	if !a.Admin {
		t.Error("Expected account to be an administrator")
	}
}

func TestAuthenticateUnknownAccount(t *testing.T) {
	r, w := setupAuthRecorder(t, "wrong", "1234512345")
	c := &Context{}

	_, err := Authenticate(c, w, r)
	if err == nil {
		t.Error("Expected Authenticate to return an error with unrecognized credentials.")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Wrong HTTP status code: %d", w.Code)
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

	if e.Error.Code != "2" {
		t.Errorf("Unexpected error code: [%s]", e.Error.Code)
	}
	if e.Error.Message != "Unable to authenticate." {
		t.Errorf("Unexpected error message: [%s]", e.Error.Message)
	}
	if e.Error.Retry {
		t.Errorf("Retry is set to true and should be false.")
	}
}
