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

// JobStatus describes the current status of a submitted job.
type JobStatus struct {
	name      string
	completed bool
}

func (s JobStatus) String() string {
	return s.name
}

// IsFinished returns true if the current status indicates that the job has completed execution,
// successfully or otherwise.
func (s JobStatus) IsFinished() bool {
	return s.completed
}

// MarshalJSON encodes a JobStatus as a JSON string.
func (s JobStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.name)
}

var (
	// StatusWaiting indicates that a job has been submitted, but has not yet entered the queue.
	StatusWaiting = JobStatus{name: "waiting"}

	// StatusQueued indicates that a job has been placed into the execution queue.
	StatusQueued = JobStatus{name: "queued"}

	// StatusProcessing indicates that the job is running.
	StatusProcessing = JobStatus{name: "processing"}

	// StatusDone indicates that the job has completed successfully.
	StatusDone = JobStatus{name: "done"}

	// StatusError indicates that the job threw some kind of exception or otherwise returned a non-zero
	// exit code.
	StatusError = JobStatus{name: "error"}

	// StatusKilled indicates that the user requested that the job be terminated.
	StatusKilled = JobStatus{name: "killed"}

	// StatusStalled indicates that the job has gotten stuck (usually fetching dependencies).
	StatusStalled = JobStatus{name: "stalled"}
)

// Collected contains various metrics about the running job.
type Collected struct {
	CPUTimeUser     uint64 `json:"cputime_user,omitempty"`
	CPUTimeSystem   uint64 `json:"cputime_system,omitempty"`
	MemoryFailCount uint64 `json:"memory_failcnt,omitempty"`
	MemoryMaxUsage  uint64 `json:"memory_max_usage,omitempty"`
}

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

// SubmittedJob is a Job that has already been submitted.
type SubmittedJob struct {
	Job

	CreatedAt  JSONTime `json:"created_at"`
	StartedAt  JSONTime `json:"started_at,omitempty"`
	FinishedAt JSONTime `json:"finished_at,omitempty"`

	Status        JobStatus `json:"status"`
	Result        string    `json:"result"`
	ReturnCode    string    `json:"return_code"`
	Runtime       uint64    `json:"runtime"`
	QueueDelay    uint64    `json:"queue_delay"`
	OverheadDelay uint64    `json:"overhead_delay"`
	Stderr        string    `json:"stderr"`
	Stdout        string    `json:"stdout"`

	Collected Collected `json:"collected,omitempty"`
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
