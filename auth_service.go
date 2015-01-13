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
	// Validate determines whether or not an API key is valid for a specific, named account.
	Validate(accountName, apiKey string) (bool, error)

	// Style provides a hint to the UI to indicate what other calls may be valid against this service.
	Style() string
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

	req, err := http.NewRequest("GET", address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain")
	resp, err := c.HTTPS.Do(req)
	if err != nil {
		return nil, err
	}

	rawStyle, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	style := strings.TrimSpace(string(rawStyle))

	return RemoteAuthService{
		HTTPS:         c.HTTPS,
		ReportedStyle: style,
		ValidateURL:   address + "validate",
	}, nil
}

// RemoteAuthService is an auth service that's implemented by calls to an HTTPS remote API.
type RemoteAuthService struct {
	HTTPS         *http.Client
	ReportedStyle string
	ValidateURL   string
}

// Validate sends a request to the configured authentication service to determine whether or not
// an API key is valid for an account.
func (service RemoteAuthService) Validate(accountName, apiKey string) (bool, error) {
	v := url.Values{}
	v.Set("accountName", accountName)
	v.Set("apiKey", apiKey)
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
		}).Error("The authentication service returned an unexpected response.")
		return false, fmt.Errorf("unexpected HTTP status %d from auth service", resp.StatusCode)
	}
}

// Style provides a hint to external API consumers about other calls and capabilities that this
// authentication service may implement.
func (service RemoteAuthService) Style() string {
	return service.ReportedStyle
}

// NullAuthService is an AuthService implementation that refuses all users and provides no optional
// capabilities. It's used as a default if no AuthService is provided and is useful to embed in
// test cases.
type NullAuthService struct{}

// Validate rejects all account-API key pairs.
func (service NullAuthService) Validate(accountName, apiKey string) (bool, error) {
	return false, nil
}

// Style informs any API consumers that no other authentication capabilities are present.
func (service NullAuthService) Style() string {
	return "null"
}

// Ensure that NullAuthService adheres to the AuthService interface.

var _ AuthService = NullAuthService{}
