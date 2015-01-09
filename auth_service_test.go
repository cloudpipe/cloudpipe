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
	service, err := ConnectToAuthService("")
	if err != nil {
		t.Fatalf("Unexpect error connecting to auth service: %v", err)
	}

	if _, ok := service.(NullAuthService); !ok {
		t.Errorf("Expected %#v to be a NullAuthService", service)
	}
}

func TestCreateRemoteService(t *testing.T) {
	authSetup()
	defer authTeardown()

	service, err := ConnectToAuthService(server.URL)
	if err != nil {
		t.Fatalf("Unexpect error connecting to auth service: %v", err)
	}

	if _, ok := service.(RemoteAuthService); !ok {
		t.Errorf("Expected %#v to be a RemoteAuthService", service)
	}
}
