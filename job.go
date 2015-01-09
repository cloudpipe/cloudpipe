package main

import (
	"fmt"
	"strings"
)

// JobLayer associates a Layer with a Job.
type JobLayer struct {
	Name string `json:"name" bson:"name"`
}

// JobVolume associates one or more Volumes with a Job.
type JobVolume struct {
	Name string `json:"name" bson:"name"`
}

const (
	// ResultBinary indicates that the client should not attempt to interpret the result payload, but
	// provide it as raw bytes.
	ResultBinary = "binary"

	// ResultPickle indicates that the result contains pickled Python objects.
	ResultPickle = "pickle"

	// StatusWaiting indicates that a job has been submitted, but has not yet entered the queue.
	StatusWaiting = "waiting"

	// StatusQueued indicates that a job has been placed into the execution queue.
	StatusQueued = "queued"

	// StatusProcessing indicates that the job is running.
	StatusProcessing = "processing"

	// StatusDone indicates that the job has completed successfully.
	StatusDone = "done"

	// StatusError indicates that the job threw some kind of exception or otherwise returned a non-zero
	// exit code.
	StatusError = "error"

	// StatusKilled indicates that the user requested that the job be terminated.
	StatusKilled = "killed"

	// StatusStalled indicates that the job has gotten stuck (usually fetching dependencies).
	StatusStalled = "stalled"
)

var (
	validResultType = map[string]bool{ResultBinary: true, ResultPickle: true}

	validStatus = map[string]bool{
		StatusWaiting:    true,
		StatusQueued:     true,
		StatusProcessing: true,
		StatusDone:       true,
		StatusError:      true,
		StatusKilled:     true,
		StatusStalled:    true,
	}

	completedStatus = map[string]bool{
		StatusDone:    true,
		StatusError:   true,
		StatusKilled:  true,
		StatusStalled: true,
	}
)

// Collected contains various metrics about the running job.
type Collected struct {
	CPUTimeUser     uint64 `json:"cputime_user,omitempty" bson:"cputime_user,omitempty"`
	CPUTimeSystem   uint64 `json:"cputime_system,omitempty" bson:"cputime_system,omitempty"`
	MemoryFailCount uint64 `json:"memory_failcnt,omitempty" bson:"memory_failcnt,omitempty"`
	MemoryMaxUsage  uint64 `json:"memory_max_usage,omitempty" bson:"memory_max_usage,omitempty"`
}

// Job is a user-submitted compute task to be executed in an appropriate Docker container.
type Job struct {
	Command      string            `json:"cmd" bson:"cmd"`
	Name         *string           `json:"name,omitempty" bson:"name,omitempty"`
	Core         string            `json:"core" bson:"core"`
	Multicore    int               `json:"multicore" bson:"multicore"`
	Restartable  bool              `json:"restartable" bson:"restartable"`
	Tags         map[string]string `json:"tags" bson:"tags"`
	Layers       []JobLayer        `json:"layer" bson:"layer"`
	Volumes      []JobVolume       `json:"vol" bson:"vol"`
	Environment  map[string]string `json:"env" bson:"env"`
	ResultSource string            `json:"result_source" bson:"result_source"`
	ResultType   string            `json:"result_type" bson:"result_type"`
	MaxRuntime   int               `json:"max_runtime" bson:"max_runtime"`
	Stdin        []byte            `json:"stdin" bson:"stdin"`

	Profile   *bool   `json:"profile,omitempty" bson:"profile,omitempty"`
	DependsOn *string `json:"depends_on,omitempty" bson:"depends_on,omitempty"`
}

// Validate ensures that all required fields have non-zero values, and that enum-like fields have
// acceptable values.
func (j Job) Validate() *APIError {
	// Command is required.
	if j.Command == "" {
		return &APIError{
			Code:    CodeMissingCommand,
			Message: "All jobs must specify a command to execute.",
			Hint:    `Specify a command to execute as a "cmd" element in your job.`,
		}
	}

	// ResultSource
	if j.ResultSource != "stdout" && !strings.HasPrefix(j.ResultSource, "file:") {
		return &APIError{
			Code:    CodeInvalidResultSource,
			Message: fmt.Sprintf("Invalid result source [%s]", j.ResultSource),
			Hint:    `The "result_source" must be either "stdout" or "file:{path}".`,
		}
	}

	// ResultType
	if _, ok := validResultType[j.ResultType]; !ok {
		accepted := make([]string, 0, len(validResultType))
		for tp := range validResultType {
			accepted = append(accepted, tp)
		}

		return &APIError{
			Code:    CodeInvalidResultType,
			Message: fmt.Sprintf("Invalid result type [%s]", j.ResultType),
			Hint:    fmt.Sprintf(`The "result_type" must be one of the following: %s`, strings.Join(accepted, ", ")),
		}
	}

	return nil
}

// SubmittedJob is a Job that has already been submitted.
type SubmittedJob struct {
	Job

	CreatedAt  StoredTime `json:"created_at" bson:"created_at"`
	StartedAt  StoredTime `json:"started_at,omitempty" bson:"started_at"`
	FinishedAt StoredTime `json:"finished_at,omitempty" bson:"finished_at"`

	Status        string `json:"status" bson:"status"`
	Result        []byte `json:"result" bson:"result"`
	ReturnCode    string `json:"return_code" bson:"return_code"`
	Runtime       int64  `json:"runtime" bson:"runtime"`
	QueueDelay    int64  `json:"queue_delay" bson:"queue_delay"`
	OverheadDelay int64  `json:"overhead_delay" bson:"overhead_delay"`
	Stderr        string `json:"stderr" bson:"stderr"`
	Stdout        string `json:"stdout" bson:"stdout"`

	Collected Collected `json:"collected,omitempty" bson:"collected,omitempty"`

	JID           uint64 `json:"jid" bson:"_id"`
	Account       string `json:"-" bson:"account"`
	ContainerID   string `json:"-" bson:"container_id,omitempty"`
	KillRequested bool   `json:"-" bson:"kill_requested,omitempty"`
}

// ContainerName derives a name for the Docker container used to execute this job.
func (j SubmittedJob) ContainerName() string {
	var nameFragment string
	if j.Name != nil {
		nameFragment = *j.Name
	} else {
		nameFragment = "unnamed"
	}

	return fmt.Sprintf("job_%d_%s", j.JID, nameFragment)
}
