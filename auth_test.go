package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TrustingAuthService accepts all usernames and tokens.
type TrustingAuthService struct{}

// Validate always returns true.
func (service TrustingAuthService) Validate(username, token string) (bool, error) {
	return true, nil
}

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
	c := &Context{
		Storage:     NullStorage{},
		AuthService: NullAuthService{},
	}

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
		Storage:     NullStorage{},
		AuthService: NullAuthService{},
	}

	a, err := Authenticate(c, w, r)
	if err != nil {
		t.Fatalf("Unable to authenticate: %v", err)
	}

	if a.Name != "admin" {
		t.Errorf("Unexpected account name: [%s]", a.Name)
	}
	if !a.Admin {
		t.Error("Expected account to be an administrator")
	}
}

func TestAuthenticateUnknownAccount(t *testing.T) {
	r, w := setupAuthRecorder(t, "wrong", "1234512345")
	c := &Context{
		Storage:     NullStorage{},
		AuthService: NullAuthService{},
	}

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

func TestAuthenticateNonAdminAccount(t *testing.T) {
	r, w := setupAuthRecorder(t, "nonadmin", "1234512345")
	c := &Context{
		Storage:     NullStorage{},
		AuthService: TrustingAuthService{},
	}

	a, err := Authenticate(c, w, r)
	if err != nil {
		t.Errorf("Unable to authenticate: %v", err)
	}

	if a.Name != "nonadmin" {
		t.Errorf("Unexpected account name: %s", a.Name)
	}
	if a.Admin {
		t.Errorf("Expected account not to be an administrator")
	}
}
