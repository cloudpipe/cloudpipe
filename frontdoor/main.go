package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

func main() {
	c, err := NewContext()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to load application context.")
		return
	}

	if c.Web {
		http.HandleFunc("/api/job", BindContext(c, JobHandler))
		http.HandleFunc("/api/job/kill", BindContext(c, JobKillHandler))
		http.HandleFunc("/api/job/kill_all", BindContext(c, JobKillAllHandler))
		http.HandleFunc("/api/job/queue_stats", BindContext(c, JobQueueStatsHandler))

		log.Info("Commence primary ignition.")
		http.ListenAndServe(c.ListenAddr(), nil)
	}

	if c.Runner {
		log.Info("Launching job runner.")
		go Runner(c)
	}
}

// ContextHandler is an HTTP HandlerFunc that accepts an additional parameter containing the
// server context.
type ContextHandler func(c *Context, w http.ResponseWriter, r *http.Request)

// BindContext returns an http.HandlerFunc that binds a ContextHandler to a specific Context.
func BindContext(c *Context, handler ContextHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { handler(c, w, r) }
}

// RhoError stores information that may be returned in an error response from the API.
type RhoError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	Retry   bool   `json:"retry,omitempty"`
}

// Report serializes an error report as JSON to an open ResponseWriter.
func (e RhoError) Report(status int, w http.ResponseWriter) error {
	var outer struct {
		Error RhoError `json:"error"`
	}
	outer.Error = e

	b, err := json.Marshal(outer)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Unable to serialize API error.")
		fmt.Fprintf(w, "Er, there was an error serializing the error. Talk to your administrator, please.")
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(b)
	return err
}

// Log logs a RhoError at the ERROR level.
func (e RhoError) Log(account *Account) RhoError {
	f := log.Fields{"error": e}
	if account != nil {
		f["account"] = account.Name
	}

	log.WithFields(f).Error(e.Message)
	return e
}

func (e *RhoError) Error() string {
	return e.Message
}

// StoredTime is a Time that can be parsed from strings in incoming JSON data, but can also be
// stored gracefully in BSON.
type StoredTime int64

const (
	timeFormat   = `2006-01-02 15:04:05.000`
	quotedFormat = `"` + timeFormat + `"`
)

// StoreTime stores a Go time.Time object as a StoredTime.
func StoreTime(t time.Time) StoredTime {
	return StoredTime(t.UTC().UnixNano())
}

// AsTime converts a StoredTime back to a Go time.Time.
func (t *StoredTime) AsTime() time.Time {
	return time.Unix(0, int64(*t)).UTC()
}

func (t *StoredTime) String() string {
	return t.AsTime().Format(timeFormat)
}

// MarshalJSON encodes a JSONTime as a UTC timestamp string.
func (t *StoredTime) MarshalJSON() ([]byte, error) {
	return []byte(t.AsTime().Format(quotedFormat)), nil
}

// UnmarshalJSON decodes a UTC timestamp string into a time.
func (t *StoredTime) UnmarshalJSON(input []byte) error {
	parsed, err := time.Parse(quotedFormat, string(input))
	*t = StoredTime(parsed.UTC().UnixNano())
	return err
}
