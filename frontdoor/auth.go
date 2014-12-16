package main

import (
	"errors"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// Account represents a user of the cluster.
type Account struct {
	Name   string
	APIKey string
	Admin  bool
}

// Authenticate reads authentication information from HTTP basic auth and attempts to locate a
// corresponding user account.
func Authenticate(c *Context, w http.ResponseWriter, r *http.Request) (*Account, error) {
	accountName, apiKey, ok := r.BasicAuth()
	if !ok {
		// Credentials not provided.
		RhoError{
			Code:    "1",
			Message: "You must authenticate.",
			Hint:    "Try using multivac.set_key(api_key='username', api_secret_key='API key', api_url='') before calling other multivac methods.",
			Retry:   false,
		}.Report(http.StatusUnauthorized, w)

		return nil, errors.New("Credentials not provided")
	}

	if c.Settings.AdminName != "" && c.Settings.AdminKey != "" {
		if accountName == c.Settings.AdminName && apiKey == c.Settings.AdminKey {
			log.WithFields(log.Fields{
				"account": accountName,
			}).Debug("Administrator authenticated.")

			return &Account{
				Name:   accountName,
				APIKey: apiKey,
				Admin:  true,
			}, nil
		}
	}

	RhoError{
		Code:    "2",
		Message: "Unable to authenticate.",
		Hint:    "Double-check the account name and API key you're providing to multivac.set_key().",
		Retry:   false,
	}.Report(http.StatusUnauthorized, w)
	return nil, fmt.Errorf("Authentication failure for account [%s]", accountName)
}
