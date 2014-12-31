package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupAuthRecorder(t *testing.T, username, key string) (*http.Request, *httptest.ResponseRecorder) {
	r, err := http.NewRequest("GET", "https://localhost/v1/jobs", nil)
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

	hasError(t, w, http.StatusUnauthorized, APIError{
		Code:    CodeCredentialsMissing,
		Message: "You must authenticate.",
		Retry:   false,
	})
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

	hasError(t, w, http.StatusUnauthorized, APIError{
		Code:    CodeCredentialsIncorrect,
		Message: "Unable to authenticate account [wrong]",
		Retry:   false,
	})
}
