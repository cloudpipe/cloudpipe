package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// AuthService describes the required and optional services that may be supplied by an authentication
// backend for cloudpipe.
type AuthService interface {
	Validate(username, token string) (bool, error)
}

// ConnectToAuthService initializes an appropriate AuthService implementation based on a (possibly
// omitted) service address.
func ConnectToAuthService(c *Context, address string) (AuthService, error) {
	if address == "" {
		return NullAuthService{}, nil
	}

	if !strings.HasPrefix(address, "https://") {
		log.WithFields(log.Fields{
			"address": address,
		}).Warn("Non-HTTPS address in use for authentication. Bad! Bad! Bad!")
	}

	if !strings.HasSuffix(address, "/") {
		address = address + "/"
	}

	return RemoteAuthService{
		HTTPS:       c.HTTPS,
		ValidateURL: address + "validate",
	}, nil
}

// RemoteAuthService is an auth service that's implemented by calls to an HTTPS remote API.
type RemoteAuthService struct {
	HTTPS       *http.Client
	ValidateURL string
}

// Validate sends a request to the configured authentication service to determine whether or not
// a username-token pair is valid.
func (service RemoteAuthService) Validate(username, token string) (bool, error) {
	v := url.Values{}
	v.Set("username", username)
	v.Set("token", token)
	resp, err := service.HTTPS.Get(service.ValidateURL + "?" + v.Encode())
	if err != nil {
		return false, err
	}

	switch resp.StatusCode {
	case http.StatusNoContent:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			body = []byte(fmt.Sprintf("Error fetching body: %v", err))
		}
		log.WithFields(log.Fields{
			"status": resp.Status,
			"body":   string(body),
		}).Error("The authentication service did something unexpected.")
		return false, fmt.Errorf("unexpected HTTP status %d from auth service", resp.StatusCode)
	}
}

// NullAuthService is an AuthService implementation that refuses all users and provides no optional
// capabilities. It's used as a default if no AuthService is provided and is useful to embed in
// test cases.
type NullAuthService struct{}

// Validate rejects all username-token pairs.
func (service NullAuthService) Validate(username, token string) (bool, error) {
	return false, nil
}

// Ensure that NullAuthService adheres to the AuthService interface.

var _ AuthService = NullAuthService{}
