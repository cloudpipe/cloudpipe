package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
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
	IsResultSource()
}

type stdoutResult struct{}

func (r stdoutResult) IsResultSource() {}

// StdoutResult is a singleton ResultSource that indicates that a Job will return its result to
// the client over stdout.
var StdoutResult = stdoutResult{}

// FileResult is a ResultSource that indicates that a Job will return its result to the client by
// placing it in a file at a certain path within its container.
type FileResult struct {
	Path string
}

func (r FileResult) String() string {
	return "file:" + r.Path
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
	ResultSource ResultSource      `json:"-"`
	ResultType   ResultType        `json:"-"`
	MaxRuntime   int               `json:"max_runtime"`
	Stdin        []byte            `json:"stdin"`

	Profile   *bool   `json:"profile,omitempty"`
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
	switch r.Method {
	case "GET":
		JobListHandler(c, w, r)
	case "POST":
		JobSubmitHandler(c, w, r)
	default:
		RhoError{
			Code:    "3",
			Message: "Method not supported",
			Hint:    "Use GET or POST against this endpoint.",
			Retry:   false,
		}.Report(http.StatusMethodNotAllowed, w)
	}
}

// JobSubmitHandler enqueues a new job associated with the authenticated account.
func JobSubmitHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	type RequestJob struct {
		Job

		RawResultSource string `json:"result_source"`
		RawResultType   string `json:"result_type"`
	}

	type Request struct {
		Jobs []RequestJob `json:"jobs"`
	}

	type Response struct {
		JIDs []uint `json:"jids"`
	}

	account, err := Authenticate(c, w, r)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Authentication failure.")
		return
	}

	// body, err := ioutil.ReadAll(r.Body)
	// log.WithFields(log.Fields{
	// 	"body": string(body),
	// }).Info("Request body")

	var req Request
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"account": account.Name,
		}).Error("Unable to parse JSON.")

		RhoError{
			Code:    "5",
			Message: "Unable to parse job payload as JSON.",
			Hint:    "Please supply valid JSON in your request.",
			Retry:   false,
		}.Report(http.StatusBadRequest, w)
		return
	}

	jids := make([]uint, len(req.Jobs))
	for index, rjob := range req.Jobs {
		job := rjob.Job

		// Interpret the deferred fields.
		if rjob.RawResultSource == "stdout" {
			job.ResultSource = StdoutResult
		} else if strings.HasPrefix(rjob.RawResultSource, "file:") {
			path := rjob.RawResultSource[len("file:") : len(rjob.RawResultSource)-1]
			job.ResultSource = FileResult{Path: path}
		} else {
			log.WithFields(log.Fields{
				"account":       account.Name,
				"result_source": rjob.RawResultSource,
			}).Error("Invalid result_source in a submitted job.")

			RhoError{
				Code:    "6",
				Message: "Invalid result_source.",
				Hint:    `"result_source" must be either "stdout" or "file:{path}".`,
				Retry:   false,
			}.Report(http.StatusBadRequest, w)
			return
		}

		switch rjob.RawResultType {
		case BinaryResult.name:
			job.ResultType = BinaryResult
		case PickleResult.name:
			job.ResultType = PickleResult
		default:
			log.WithFields(log.Fields{
				"account":     account.Name,
				"result_type": rjob.RawResultType,
			}).Error("Invalid result_type in a submitted job.")

			RhoError{
				Code:    "7",
				Message: "Invalid result_type.",
				Hint:    `"result_type" must be either "binary" or "pickle".`,
				Retry:   false,
			}.Report(http.StatusBadRequest, w)
			return
		}

		jids[index] = uint(index)
		log.WithFields(log.Fields{
			"job":     job,
			"account": account.Name,
		}).Info("Succesfully submitted a job.")
	}

	response := Response{JIDs: jids}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// JobListHandler provides updated details about one or more jobs currently submitted to the
// cluster.
func JobListHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `[]`)
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
