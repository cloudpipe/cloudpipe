package main

import (
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
		err := &RhoError{
			Code:    CodeCredentialsMissing,
			Message: "You must authenticate.",
			Hint:    "Try using multivac.config.set_key(api_key='username', api_secret_key='API key', api_url='') before calling other multivac methods.",
			Retry:   false,
		}
		err.Report(http.StatusUnauthorized, w)
		return nil, err
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

	err := &RhoError{
		Code:    CodeCredentialsIncorrect,
		Message: fmt.Sprintf("Unable to authenticate account [%s]", accountName),
		Hint:    "Double-check the account name and API key you're providing to multivac.config.set_key().",
		Retry:   false,
	}
	err.Report(http.StatusUnauthorized, w)
	return nil, err
}
