package main

import (
	"encoding/json"
	"net/http"
)

// JobLayer associates a Layer with a Job.
type JobLayer struct {
	Name string `json:"name"`
}

// JobVolume associates one or more Volumes with a Job.
type JobVolume struct {
	Name string `json:"name"`
}

// ResultSource describes a mechanism for providing a Job's result back to the client. This can be
// either the singleton constant StdoutResult or a FileResult with a path.
type ResultSource interface {
	json.Marshaler

	IsResultSource()
}

type stdoutResult struct{}

func (r stdoutResult) MarshalJSON() ([]byte, error) {
	return []byte(`"stdout"`), nil
}

func (r stdoutResult) IsResultSource() {}

// StdoutResult is a singleton ResultSource that indicates that a Job will return its result to
// the client over stdout.
var StdoutResult = stdoutResult{}

// FileResult is a ResultSource that indicates that a Job will return its result to the client by
// placing it in a file at a certain path within its container.
type FileResult struct {
	Path string
}

// MarshalJSON converts a FileResult into its JSON representation as the string "file:<path>".
func (r FileResult) MarshalJSON() ([]byte, error) {
	return json.Marshal("file:" + r.Path)
}

// IsResultSource is a marker method for the ResultSource interface.
func (r FileResult) IsResultSource() {}

// ResultType indicates how a Job's output should be interpreted by the client. Must be one of
// BinaryResult or PickleResult.
type ResultType struct {
	name string
}

func (s ResultType) String() string {
	return s.name
}

var (
	// BinaryResult indicates that the client should not attempt to interpret the result payload, but
	// provide it as raw bytes.
	BinaryResult = ResultType{name: "binary"}

	// PickleResult indicates that the result contains pickled Python objects.
	PickleResult = ResultType{name: "pickle"}
)

// Job is a user-submitted compute task to be executed in an appropriate Docker container.
type Job struct {
	Command      string            `json:"cmd"`
	Name         *string           `json:"name,omitempty"`
	Core         string            `json:"core"`
	Multicore    int               `json:"multicore"`
	Restartable  bool              `json:"restartable"`
	Tags         map[string]string `json:"tags"`
	Layers       []JobLayer        `json:"layer"`
	Volumes      []JobVolume       `json:"vol"`
	Environment  map[string]string `json:"env"`
	ResultSource ResultSource      `json:"result_source"`
	ResultType   ResultType        `json:"result_type"`
	MaxRuntime   int               `json:"max_runtime"`
	Stdin        []byte            `json:"stdin"`

	Profile   *string `json:"profile,omitempty"`
	DependsOn *string `json:"depends_on,omitempty"`
}

// JobHandler dispatches API calls to /job based on request type.
func JobHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	//
}

// JobSubmitHandler enqueues a new job associated with the authenticated account.
func JobSubmitHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	//
}

// JobListHandler provides updated details about one or more jobs currently submitted to the
// cluster.
func JobListHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	//
}

// JobKillHandler allows a user to prematurely terminate a running job.
func JobKillHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	//
}

// JobKillAllHandler allows a user to terminate all jobs associated with their account.
func JobKillAllHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	//
}

// JobQueueStatsHandler allows a user to view statistics about the jobs that they have submitted.
func JobQueueStatsHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	//
}
