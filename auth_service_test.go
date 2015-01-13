package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	mux    *http.ServeMux
	server *httptest.Server
)

func authSetup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
}

func authTeardown() {
	server.Close()
}

func TestDefaultToNullService(t *testing.T) {
	service := ConnectToAuthService(&Context{}, "")
	if _, ok := service.(NullAuthService); !ok {
		t.Errorf("Expected %#v to be a NullAuthService", service)
	}
}

func TestCreateRemoteService(t *testing.T) {
	authSetup()
	defer authTeardown()

	service := ConnectToAuthService(&Context{}, server.URL)
	if _, ok := service.(RemoteAuthService); !ok {
		t.Errorf("Expected %#v to be a RemoteAuthService", service)
	}
}

func TestSuccessfulRemoteAuth(t *testing.T) {
	authSetup()
	defer authTeardown()

	hit := false
	mux.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		hit = true
		if r.Method != "GET" {
			t.Errorf("Expected a GET request, but was [%s]", r.Method)
		}

		err := r.ParseForm()
		if err != nil {
			t.Errorf("Unexpected error parsing form: %v", err)
		}

		if username := r.FormValue("username"); username != "someuser" {
			t.Errorf("Unexpected username: [%s]", username)
		}

		if token := r.FormValue("token"); token != "1234567" {
			t.Errorf("Unexpected token: [%s]", token)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	c := &Context{HTTPS: http.DefaultClient}
	service := ConnectToAuthService(c, server.URL)

	ok, err := service.Validate("someuser", "1234567")
	if err != nil {
		t.Fatalf("Unexpected error calling auth service: %v", err)
	}

	if !hit {
		t.Errorf("Service never called remote endpoint")
	}

	if !ok {
		t.Errorf("Service unexpectedly rejected authentication")
	}
}

func TestUnsuccessfulRemoteAuth(t *testing.T) {
	authSetup()
	defer authTeardown()

	hit := false
	mux.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNotFound)
	})

	c := &Context{HTTPS: http.DefaultClient}
	service := ConnectToAuthService(c, server.URL)

	ok, err := service.Validate("someuser", "1234567")
	if err != nil {
		t.Fatalf("Unexpected error calling auth service: %v", err)
	}

	if !hit {
		t.Errorf("Service never called remote endpoint")
	}

	if ok {
		t.Errorf("Service unexpectedly accepted authentication")
	}
}
