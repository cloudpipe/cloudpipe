package main

import (
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// Account represents a user of the cluster.
type Account struct {
	Name  string `bson:"name"`
	Admin bool   `bson:"admin"`

	// TotalRuntime tracks the cumulative runtime of all jobs submitted on behalf of this account, in
	// nanoseconds.
	TotalRuntime int64 `bson:"total_runtime"`

	// TotalJobs tracks the number of jobs submitted on behalf of this account.
	TotalJobs int64 `bson:"total_jobs"`
}

// Authenticate reads authentication information from HTTP basic auth and attempts to locate a
// corresponding user account.
func Authenticate(c *Context, w http.ResponseWriter, r *http.Request) (*Account, error) {
	accountName, apiKey, ok := r.BasicAuth()
	if !ok {
		// Credentials not provided.
		err := &APIError{
			Code:    CodeCredentialsMissing,
			Message: "You must authenticate.",
			Hint:    "Try using multyvac.config.set_key(api_key='username', api_secret_key='API key', api_url='') before calling other multyvac methods.",
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
				Name:  accountName,
				Admin: true,
			}, nil
		}
	}

	err := &APIError{
		Code:    CodeCredentialsIncorrect,
		Message: fmt.Sprintf("Unable to authenticate account [%s]", accountName),
		Hint:    "Double-check the account name and API key you're providing to multyvac.config.set_key().",
		Retry:   false,
	}
	err.Report(http.StatusUnauthorized, w)
	return nil, err
}
