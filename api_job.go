package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"
)

// JobHandler dispatches API calls to /job based on request type.
func JobHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		JobListHandler(c, w, r)
	case "POST":
		JobSubmitHandler(c, w, r)
	default:
		APIError{
			Code:    CodeMethodNotSupported,
			Message: "Method not supported",
			Hint:    "Use GET or POST against this endpoint.",
			Retry:   false,
		}.Report(http.StatusMethodNotAllowed, w)
	}
}

// JobSubmitHandler enqueues a new job associated with the authenticated account.
func JobSubmitHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Jobs []Job `json:"jobs"`
	}

	type Response struct {
		JIDs []uint64 `json:"jids"`
	}

	account, err := Authenticate(c, w, r)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Authentication failure.")
		return
	}

	var req Request
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"account": account.Name,
		}).Error("Unable to parse JSON.")

		APIError{
			Code:    CodeInvalidJobJSON,
			Message: fmt.Sprintf("Unable to parse job payload as JSON: %v", err),
			Hint:    "Please supply valid JSON in your request.",
			Retry:   false,
		}.Report(http.StatusBadRequest, w)
		return
	}

	jids := make([]uint64, len(req.Jobs))
	for index, job := range req.Jobs {
		// Validate the job.
		if err := job.Validate(); err != nil {
			log.WithFields(log.Fields{
				"account": account.Name,
				"job":     job,
				"error":   err,
			}).Error("Invalid job submitted.")

			err.Report(http.StatusBadRequest, w)
			return
		}

		// Pack the job into a SubmittedJob and store it.
		submitted := SubmittedJob{
			Job:       job,
			CreatedAt: StoreTime(time.Now()),
			Status:    StatusQueued,
			Account:   account.Name,
		}
		jid, err := c.InsertJob(submitted)
		if err != nil {
			log.WithFields(log.Fields{
				"account": account.Name,
				"error":   err,
			}).Error("Unable to enqueue a submitted job.")

			APIError{
				Code:    CodeEnqueueFailure,
				Message: "Unable to enqueue your job.",
				Retry:   true,
			}.Report(http.StatusServiceUnavailable, w)
			return
		}

		jids[index] = jid
		log.WithFields(log.Fields{
			"jid":     jid,
			"job":     job,
			"account": account.Name,
		}).Info("Successfully submitted a job.")
	}

	response := Response{JIDs: jids}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// JobListHandler provides updated details about one or more jobs currently submitted to the
// cluster.
func JobListHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	account, err := Authenticate(c, w, r)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Authentication failure.")
		return
	}

	if err := r.ParseForm(); err != nil {
		APIError{
			Code:    CodeUnableToParseQuery,
			Message: fmt.Sprintf("Unable to parse query parameters: %v", err),
			Hint:    "You broke Go's URL parsing somehow! Make URLs that suck less.",
			Retry:   false,
		}.Log(account).Report(http.StatusBadRequest, w)
		return
	}

	q := JobQuery{AccountName: account.Name}
	if rawJIDs, ok := r.Form["jid"]; ok {
		jids := make([]uint64, len(rawJIDs))
		for i, rawJID := range rawJIDs {
			if jids[i], err = strconv.ParseUint(rawJID, 10, 64); err != nil {
				APIError{
					Code:    CodeUnableToParseQuery,
					Message: fmt.Sprintf("Unable to parse JID [%s]: %v", rawJID, err),
					Hint:    "Please only use valid JIDs.",
					Retry:   false,
				}.Log(account).Report(http.StatusBadRequest, w)
				return
			}
		}
		q.JIDs = jids
	}
	if names, ok := r.Form["name"]; ok {
		q.Names = names
	}
	if statuses, ok := r.Form["status"]; ok {
		q.Statuses = statuses
	}
	if rawLimit := r.FormValue("limit"); rawLimit != "" {
		limit, err := strconv.ParseInt(rawLimit, 10, 0)
		if err != nil {
			APIError{
				Code:    CodeUnableToParseQuery,
				Message: fmt.Sprintf("Unable to parse limit [%s]: %v", rawLimit, err),
				Hint:    "Please specify a valid integral limit.",
				Retry:   false,
			}.Log(account).Report(http.StatusBadRequest, w)
			return
		}

		if limit > 9999 {
			limit = 9999
		}
		if limit < 1 {
			APIError{
				Code:    CodeUnableToParseQuery,
				Message: fmt.Sprintf("Invalid negative or zero limit [%d]", limit),
				Hint:    "Please specify a valid, positive integral limit.",
				Retry:   false,
			}.Log(account).Report(http.StatusBadRequest, w)
			return
		}
		q.Limit = int(limit)
	} else {
		q.Limit = 1000
	}

	if rawBefore := r.FormValue("before"); rawBefore != "" {
		before, err := strconv.ParseUint(rawBefore, 10, 64)
		if err != nil {
			APIError{
				Code:    CodeUnableToParseQuery,
				Message: fmt.Sprintf(`Unable to parse Before bound [%s]: %v`, rawBefore, err),
				Hint:    "Please specify a valid integral JID as the lower bound.",
				Retry:   false,
			}.Log(account).Report(http.StatusBadRequest, w)
			return
		}
		q.Before = before
	}
	if rawAfter := r.FormValue("after"); rawAfter != "" {
		after, err := strconv.ParseUint(rawAfter, 10, 64)
		if err != nil {
			APIError{
				Code:    CodeUnableToParseQuery,
				Message: fmt.Sprintf(`Unable to parse After bound [%s]: %v`, rawAfter, err),
				Hint:    "Please specify a valid integral JID as the upper bound.",
				Retry:   false,
			}.Log(account).Report(http.StatusBadRequest, w)
			return
		}
		q.After = after
	}

	results, err := c.ListJobs(q)
	if err != nil {
		re := APIError{
			Code:    CodeListFailure,
			Message: fmt.Sprintf("Unable to list jobs: %v", err),
			Hint:    "This is most likely a database problem.",
			Retry:   true,
		}
		re.Log(account).Report(http.StatusServiceUnavailable, w)
		return
	}

	var response struct {
		Jobs []SubmittedJob `json:"jobs"`
	}
	response.Jobs = results

	log.WithFields(log.Fields{
		"query":        q,
		"result count": len(results),
		"account":      account.Name,
	}).Debug("Successful job query.")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// JobKillHandler allows a user to prematurely terminate a running job.
func JobKillHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	account, err := Authenticate(c, w, r)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Authentication failure.")
		return
	}

	if err = r.ParseForm(); err != nil {
		APIError{
			Code:    CodeInvalidJobForm,
			Message: fmt.Sprintf("Unable to parse Job: Kill payload as a POST body: %v", err),
			Hint:    "Please use valid form encoding in your request.",
			Retry:   false,
		}.Log(account).Report(http.StatusBadRequest, w)
		return
	}

	jidstr := r.PostFormValue("jid")
	jid, err := strconv.ParseUint(jidstr, 10, 64)
	if err != nil {
		APIError{
			Code:    CodeInvalidJobForm,
			Message: fmt.Sprintf("Unable to parse Job: Kill payload as a valid JID: %v", err),
			Hint:    "Please provide a valid integer job ID to Job: Kill.",
			Retry:   false,
		}.Log(account).Report(http.StatusBadRequest, w)
		return
	}

	sudo := r.PostFormValue("sudo") == "true"

	query := JobQuery{JIDs: []uint64{jid}}
	if !sudo {
		query.AccountName = account.Name
	}

	jobs, err := c.ListJobs(query)
	if err != nil {
		APIError{
			Code:    CodeListFailure,
			Message: "Unable to list jobs.",
			Hint:    "This is probably a storage error on our end.",
			Retry:   true,
		}.Log(account).Report(http.StatusInternalServerError, w)
		return
	}

	if len(jobs) == 0 {
		APIError{
			Code:    CodeJobNotFound,
			Message: fmt.Sprintf("Unable to find a job with ID [%s].", jid),
			Hint:    "Make sure that the JID is still valid.",
			Retry:   false,
		}.Log(account).Report(http.StatusNotFound, w)
		return
	}
	if len(jobs) != 1 {
		APIError{
			Code: CodeWTF,
			Message: fmt.Sprintf(
				"Job query for JID [%s] on account [%s] returned [%d] results.",
				jid, account.Name, len(jobs),
			),
			Hint:  "Duplicate JID. No clue how that happened.",
			Retry: false,
		}.Log(account).Report(http.StatusInternalServerError, w)
		return
	}

	job := &jobs[0]

	job.KillRequested = true

	// If the container ID hasn't been assigned yet, the job most likely isn't running.
	// If it's already left StatusQueued, let the job runner handle the transition to
	// StatusKilled. Otherwise, set it to StatusKilled ourselves to remove it from the queue.
	if job.Status == StatusQueued {
		job.Status = StatusKilled
	}

	err = c.UpdateJob(job)
	if err != nil {
		APIError{
			Code:    CodeJobUpdateFailure,
			Message: fmt.Sprintf("Unable to request a job kill: %v", err),
			Hint:    "This is probably a storage error on our end.",
			Retry:   true,
		}.Log(account).Report(http.StatusInternalServerError, w)
		return
	}

	if job.ContainerID != "" {
		err = c.KillContainer(docker.KillContainerOptions{ID: job.ContainerID})
		if err != nil {
			APIError{
				Code:    CodeJobKillFailure,
				Message: fmt.Sprintf("Unable to kill a running job: %v", err),
				Hint:    "The container is misbehaving somehow.",
				Retry:   true,
			}.Log(account).Report(http.StatusInternalServerError, w)
			return
		}

		log.WithFields(log.Fields{
			"jid":     job.JID,
			"account": account.Name,
			"sudo":    sudo,
		}).Info("Running job killed.")
	} else {
		log.WithFields(log.Fields{
			"jid":     job.JID,
			"account": account.Name,
			"sudo":    sudo,
		}).Info("Job kill requested.")
	}

	OKResponse(w)
}

// JobKillAllHandler allows a user to terminate all jobs associated with their account.
func JobKillAllHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	//
}

// JobQueueStatsHandler allows a user to view statistics about the jobs that they have submitted.
func JobQueueStatsHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	//
}
