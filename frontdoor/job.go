package main

import "net/http"

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
