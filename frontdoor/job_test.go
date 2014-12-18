package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// JobStorage is a fake Storage implementation that only provides job-relevant storage methods.
type JobStorage struct {
	NullStorage

	Submitted SubmittedJob
	Query     JobQuery
}

func (storage *JobStorage) InsertJob(job SubmittedJob) (uint64, error) {
	storage.Submitted = job

	return 42, nil
}

func (storage *JobStorage) ListJobs(query JobQuery) ([]SubmittedJob, error) {
	storage.Query = query

	j0 := SubmittedJob{
		Job: Job{Command: `echo "1"`},
		JID: 11,
	}
	j1 := SubmittedJob{
		Job: Job{Command: `echo "2"`},
		JID: 22,
	}
	j2 := SubmittedJob{
		Job: Job{Command: `echo "3"`},
		JID: 33,
	}

	return []SubmittedJob{j0, j1, j2}, nil
}

func TestJobHandlerBadRequest(t *testing.T) {
	r, err := http.NewRequest("PUT", "https://localhost/api/jobs", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	w := httptest.NewRecorder()
	c := &Context{}

	JobHandler(c, w, r)

	hasError(t, w, http.StatusMethodNotAllowed, RhoError{
		Code:    CodeMethodNotSupported,
		Message: "Method not supported",
		Retry:   false,
	})
}

func TestSubmitJob(t *testing.T) {
	body := strings.NewReader(`
	{
		"jobs": [{
			"cmd": "id",
			"name": "wat",
			"result_source": "stdout",
			"result_type": "binary"
		}]
	}
	`)
	r, err := http.NewRequest("POST", "https://localhost/api/jobs", body)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	r.SetBasicAuth("admin", "12345")
	w := httptest.NewRecorder()
	s := &JobStorage{}
	c := &Context{
		Settings: Settings{
			AdminName: "admin",
			AdminKey:  "12345",
		},
		Storage: s,
	}

	JobHandler(c, w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected HTTP status: [%d]", w.Code)
	}

	var response struct {
		JIDs []uint `json:"jids"`
	}
	out := w.Body.Bytes()
	if err := json.Unmarshal(out, &response); err != nil {
		t.Fatalf("Unable to parse response body as JSON: [%s]", string(out))
	}
	if len(response.JIDs) != 1 {
		t.Fatalf("Expected one JID, received [%d]", len(response.JIDs))
	}
	if response.JIDs[0] != 42 {
		t.Errorf("Expected to be assigned ID 42, got [%d]", response.JIDs[0])
	}

	if s.Submitted.Account != "admin" {
		t.Errorf("Expected submitted job to belong to admin, not [%s]", s.Submitted.Account)
	}
	if s.Submitted.Status != StatusQueued {
		t.Errorf("Expected submitted job to be in state queued, not [%s]", s.Submitted.Status)
	}

	if s.Submitted.CreatedAt == 0 {
		t.Error("Expected the job's CreatedAt time to be populated.")
	}
	if s.Submitted.StartedAt != 0 {
		t.Errorf("Expected the job's StartedAt time to be zero, but was [%s]", s.Submitted.StartedAt)
	}
	if s.Submitted.FinishedAt != 0 {
		t.Errorf("Expected the job's FinishedAt time to be zero, but was [%s]", s.Submitted.FinishedAt)
	}
}

func TestSubmitJobBadResultSource(t *testing.T) {
	body := strings.NewReader(`
	{
		"jobs": [{
			"cmd": "id",
			"name": "wat",
			"result_source": "magic",
			"result_type": "binary"
		}]
	}
	`)
	r, err := http.NewRequest("POST", "https://localhost/api/jobs", body)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	r.SetBasicAuth("admin", "12345")
	w := httptest.NewRecorder()
	c := &Context{
		Settings: Settings{
			AdminName: "admin",
			AdminKey:  "12345",
		},
	}

	JobHandler(c, w, r)

	hasError(t, w, http.StatusBadRequest, RhoError{
		Code:    CodeInvalidResultSource,
		Message: "Invalid result source [magic]",
		Retry:   false,
	})
}

func TestSubmitJobBadResultType(t *testing.T) {
	body := strings.NewReader(`
	{
		"jobs": [{
			"cmd": "id",
			"name": "wat",
			"result_source": "stdout",
			"result_type": "elsewhere"
		}]
	}
	`)
	r, err := http.NewRequest("POST", "https://localhost/api/jobs", body)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	r.SetBasicAuth("admin", "12345")
	w := httptest.NewRecorder()
	c := &Context{
		Settings: Settings{
			AdminName: "admin",
			AdminKey:  "12345",
		},
	}

	JobHandler(c, w, r)

	hasError(t, w, http.StatusBadRequest, RhoError{
		Code:    CodeInvalidResultType,
		Message: "Invalid result type [elsewhere]",
		Retry:   false,
	})
}

func TestListJobsAll(t *testing.T) {
	r, err := http.NewRequest("GET", "https://localhost/api/jobs", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	r.SetBasicAuth("admin", "12345")
	w := httptest.NewRecorder()
	s := &JobStorage{}
	c := &Context{
		Settings: Settings{
			AdminName: "admin",
			AdminKey:  "12345",
		},
		Storage: s,
	}

	JobHandler(c, w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected HTTP status: [%d]", w.Code)
	}

	var response struct {
		Jobs []SubmittedJob `json:"jobs"`
	}
	out := w.Body.Bytes()
	if err := json.Unmarshal(out, &response); err != nil {
		t.Fatalf("Unable to parse response body as JSON: [%s]", string(out))
	}
	t.Logf("Response body:\n%s", out)

	if len(response.Jobs) != 3 {
		t.Fatalf("Unexpected number of jobs returned: [%d]", len(response.Jobs))
	}
	if cmd0 := response.Jobs[0].Command; cmd0 != `echo "1"` {
		t.Errorf(`Expected first job to have command 'echo "1"', had [%s]`, cmd0)
	}
	if cmd1 := response.Jobs[1].Command; cmd1 != `echo "2"` {
		t.Errorf(`Expected second job to have command 'echo "2"', had [%s]`, cmd1)
	}
	if cmd2 := response.Jobs[2].Command; cmd2 != `echo "3"` {
		t.Errorf(`Expected third job to have command 'echo "3"', had [%s]`, cmd2)
	}
}
