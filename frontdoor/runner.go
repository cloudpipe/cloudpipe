package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"
)

// Runner is the main entry point for the job runner goroutine.
func Runner(c *Context) {
	client, err := docker.NewClient(c.DockerHost)
	if err != nil {
		log.WithFields(log.Fields{
			"docker host": c.DockerHost,
			"error":       err,
		}).Fatal("Unable to connect to Docker.")
		return
	}

	for {
		select {
		case <-time.After(time.Duration(c.Poll) * time.Millisecond):
			Claim(c, client)
		}
	}
}

// Claim acquires the oldest single pending job and launches a goroutine to execute its command in
// a new container.
func Claim(c *Context, client *docker.Client) {
	job, err := c.ClaimJob()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Unable to claim a job.")
		return
	}
	if job == nil {
		// Nothing to claim.
		return
	}

	log.WithFields(log.Fields{
		"jid": job.JID,
	}).Info("Launching a new job.")

	go Execute(c, client, job)
}

// Execute launches a container to process the submitted job. It passes any provided stdin data
// to the container and consumes stdout and stderr, updating Mongo as it runs. Once completed, it
// acquires the job's result from its configured source and marks the job as finished.
func Execute(c *Context, client *docker.Client, job *SubmittedJob) {
	job.StartedAt = StoreTime(time.Now())
	if err := c.UpdateJob(job); err != nil {
		log.WithFields(log.Fields{
			"jid":   job.JID,
			"error": err,
		}).Error("Unable to update job start timestamp.")
		return
	}

	log.WithFields(log.Fields{
		"jid":     job.JID,
		"command": job.Command,
	}).Info("Hey look I'm executing a job!")

	job.FinishedAt = StoreTime(time.Now())
	job.Status = StatusDone
	if err := c.UpdateJob(job); err != nil {
		log.WithFields(log.Fields{
			"jid":   job.JID,
			"error": err,
		}).Error(`Unable to update job status to "done".`)
		return
	}
}
