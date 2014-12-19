package main

import (
	"bytes"
	"strings"
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
			"jid":     job.JID,
			"account": job.Account,
			"error":   err,
		}).Error("Unable to update the job's start timestamp.")
		return
	}

	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Name: job.ContainerName(),
		Config: &docker.Config{
			Image:     c.Image,
			Cmd:       strings.Split(job.Command, " "),
			OpenStdin: true,
		},
	})
	if err != nil {
		log.WithFields(log.Fields{
			"jid":     job.JID,
			"account": job.Account,
			"error":   err,
		}).Error("Unable to create the job's container.")
		return
	}
	log.WithFields(log.Fields{
		"jid":            job.JID,
		"account":        job.Account,
		"container id":   container.ID,
		"container name": container.Name,
	}).Debug("Container created successfully.")

	// Start the created container.
	if err := client.StartContainer(container.ID, &docker.HostConfig{}); err != nil {
		log.WithFields(log.Fields{
			"jid":            job.JID,
			"account":        job.Account,
			"container id":   container.ID,
			"container name": container.Name,
			"error":          err,
		}).Error("Unable to start the job's container.")
		return
	}
	log.WithFields(log.Fields{
		"jid":            job.JID,
		"account":        job.Account,
		"container id":   container.ID,
		"container name": container.Name,
	}).Debug("Container started successfully.")

	// Prepare the input and output streams.
	stdin := bytes.NewReader(job.Stdin)
	var stdout, stderr bytes.Buffer

	complete := make(chan struct{})

	log.Debug("About to attach.")
	err = client.AttachToContainer(docker.AttachToContainerOptions{
		Container:    container.ID,
		InputStream:  stdin,
		OutputStream: &stdout,
		ErrorStream:  &stderr,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"jid":            job.JID,
			"account":        job.Account,
			"container id":   container.ID,
			"container name": container.Name,
			"error":          err,
		}).Error("Unable to attach to the job's container.")
		return
	}
	log.Debug("Waiting for attachment to succeed.")

	log.WithFields(log.Fields{
		"jid":            job.JID,
		"account":        job.Account,
		"container id":   container.ID,
		"container name": container.Name,
	}).Debug("Attached to the job's container.")

	go func() {
		log.WithFields(log.Fields{"jid": job.JID}).Debug("Polling container I/O.")
	IOLOOP:
		for {
			select {
			case <-time.After(100 * time.Millisecond):
				log.WithFields(log.Fields{"jid": job.JID}).Debug("Reading output so far.")
				nout, nerr := stdout.String(), stderr.String()

				log.WithFields(log.Fields{
					"jid":  job.JID,
					"nout": nout,
					"nerr": nerr,
				}).Debug("Read bytes from stdout and/or stderr.")

				job.Stdout += nout
				job.Stderr += nerr

				if len(nout) > 0 || len(nerr) > 0 {
					if err := c.UpdateJob(job); err != nil {
						log.WithFields(log.Fields{
							"jid":            job.JID,
							"account":        job.Account,
							"container id":   container.ID,
							"container name": container.Name,
							"error":          err,
						}).Warn("Unable to update the job's stdout and stderr.")
					}
				}
			case <-complete:
				log.WithFields(log.Fields{"jid": job.JID}).Debug("Complete signal received.")
				break IOLOOP
			}
		}
		log.WithFields(log.Fields{"jid": job.JID}).Debug("Polling loop complete.")
	}()

	log.Debug("Waiting for job completion.")
	status, err := client.WaitContainer(container.ID)
	log.Debug("Signalling completion.")
	complete <- struct{}{}
	if err != nil {
		log.WithFields(log.Fields{
			"jid":            job.JID,
			"account":        job.Account,
			"container id":   container.ID,
			"container name": container.Name,
			"error":          err,
		}).Error("Unable to wait for the container to terminate.")
		return
	}

	job.FinishedAt = StoreTime(time.Now())
	if status == 0 {
		// Successful termination.
		job.Status = StatusDone
	} else {
		// Something went wrong.
		job.Status = StatusError
	}

	if err := c.UpdateJob(job); err != nil {
		log.WithFields(log.Fields{
			"jid":   job.JID,
			"error": err,
		}).Error("Unable to update job status.")
		return
	}

	log.WithFields(log.Fields{
		"jid": job.JID,
	}).Info("Job complete.")
}
