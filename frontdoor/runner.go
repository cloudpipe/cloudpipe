package main

import (
	"bytes"
	"time"

	log "github.com/Sirupsen/logrus"
	docker "github.com/smashwilson/go-dockerclient"
)

// OutputCollector is an io.Writer that accumulates output from a specified stream in an attached
// Docker container and appends it to the appropriate field within a SubmittedJob.
type OutputCollector struct {
	context  *Context
	job      *SubmittedJob
	isStdout bool
}

// Write appends bytes to the selected stream and updates the SubmittedJob.
func (c OutputCollector) Write(p []byte) (int, error) {
	var stream string
	if c.isStdout {
		stream = "stdout"
	} else {
		stream = "stderr"
	}
	log.WithFields(log.Fields{
		"length": len(p),
		"bytes":  string(p),
		"stream": stream,
	}).Debug("Received output from a job")

	if c.isStdout {
		c.job.Stdout += string(p)
	} else {
		c.job.Stderr += string(p)
	}

	if err := c.context.UpdateJob(c.job); err != nil {
		return 0, err
	}

	return len(p), nil
}

// Runner is the main entry point for the job runner goroutine.
func Runner(c *Context) {
	var client *docker.Client
	var err error

	if c.DockerTLS {
		client, err = docker.NewTLSClient(c.DockerHost, c.DockerCert, c.DockerKey, c.DockerCACert)
		if err != nil {
			log.WithFields(log.Fields{
				"docker host":    c.DockerHost,
				"docker cert":    c.DockerCert,
				"docker key":     c.DockerKey,
				"docker CA cert": c.DockerCACert,
			}).Fatal("Unable to connect to Docker with TLS.")
			return
		}
	} else {
		client, err = docker.NewClient(c.DockerHost)
		if err != nil {
			log.WithFields(log.Fields{
				"docker host": c.DockerHost,
				"error":       err,
			}).Fatal("Unable to connect to Docker.")
			return
		}
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
			Cmd:       []string{"sh", "-c", job.Command},
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
	stdout := OutputCollector{
		context:  c,
		job:      job,
		isStdout: true,
	}
	stderr := OutputCollector{
		context:  c,
		job:      job,
		isStdout: false,
	}

	log.Debug("About to attach.")
	err = client.AttachToContainer(docker.AttachToContainerOptions{
		Container:    container.ID,
		Stream:       true,
		InputStream:  stdin,
		OutputStream: stdout,
		ErrorStream:  stderr,
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
	log.WithFields(log.Fields{
		"jid":            job.JID,
		"account":        job.Account,
		"container id":   container.ID,
		"container name": container.Name,
	}).Debug("Attached to the job's container.")

	log.Debug("Waiting for job completion.")
	status, err := client.WaitContainer(container.ID)
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
