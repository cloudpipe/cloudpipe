package main

import (
	"encoding/json"
	"fmt"
	"net/http"

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

	http.HandleFunc("/api/job", BindContext(c, JobHandler))
	http.HandleFunc("/api/job/kill", BindContext(c, JobKillHandler))
	http.HandleFunc("/api/job/kill_all", BindContext(c, JobKillAllHandler))
	http.HandleFunc("/api/job/queue_stats", BindContext(c, JobQueueStatsHandler))

	log.Info("Commence primary ignition.")
	http.ListenAndServe(c.ListenAddr(), nil)

	fmt.Println("I exist")
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
