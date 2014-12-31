package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"strings"
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

// DescribeStream returns "stdout" or "stderr" to indicate which stream this collector is consuming.
func (c OutputCollector) DescribeStream() string {
	if c.isStdout {
		return "stdout"
	}
	return "stderr"
}

// Write appends bytes to the selected stream and updates the SubmittedJob.
func (c OutputCollector) Write(p []byte) (int, error) {
	log.WithFields(log.Fields{
		"length": len(p),
		"bytes":  string(p),
		"stream": c.DescribeStream(),
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
		log.WithFields(log.Fields{"error": err}).Error("Unable to claim a job.")
		return
	}
	if job == nil {
		// Nothing to claim.
		return
	}
	if err := job.Validate(); err != nil {
		fields := log.Fields{
			"jid":     job.JID,
			"account": job.Account,
			"error":   err,
		}

		log.WithFields(fields).Error("Invalid job in queue.")

		job.Status = StatusError
		if err := c.UpdateJob(job); err != nil {
			fields["error"] = err
			log.WithFields(fields).Error("Unable to update job status.")
		}

		return
	}

	go Execute(c, client, job)
}

// Execute launches a container to process the submitted job. It passes any provided stdin data
// to the container and consumes stdout and stderr, updating Mongo as it runs. Once completed, it
// acquires the job's result from its configured source and marks the job as finished.
func Execute(c *Context, client *docker.Client, job *SubmittedJob) {
	defaultFields := log.Fields{
		"jid":     job.JID,
		"account": job.Account,
	}

	// Logging utility messages.
	debug := func(message string) {
		log.WithFields(defaultFields).Debug(message)
	}
	reportErr := func(message string, err error) {
		fs := log.Fields{}
		for k, v := range defaultFields {
			fs[k] = v
		}
		fs["err"] = err
		log.WithFields(fs).Error(message)
	}
	checkErr := func(message string, err error) bool {
		if err == nil {
			debug(fmt.Sprintf("%s: ok", message))
			return false
		}

		reportErr(fmt.Sprintf("%s: ERROR", message), err)
		return true
	}

	// Update the job model in mongo, reporting any errors along the way.
	updateJob := func(message string) bool {
		if err := c.UpdateJob(job); err != nil {
			reportErr(fmt.Sprintf("Unable to update the job's %s.", message), err)
			return false
		}
		return true
	}

	log.WithFields(defaultFields).Info("Launching a job.")

	job.StartedAt = StoreTime(time.Now())
	updateJob("start timestamp")

	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Name: job.ContainerName(),
		Config: &docker.Config{
			Image:     c.Image,
			Cmd:       []string{"/bin/bash", "-c", job.Command},
			OpenStdin: true,
			StdinOnce: true,
		},
	})
	if checkErr("Created the job's container", err) {
		job.Status = StatusError
		updateJob("status")
		return
	}

	// Include container information in this job's logging messages.
	defaultFields["container id"] = container.ID
	defaultFields["container name"] = container.Name

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

	go func() {
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
		checkErr("Attached to the container", err)
	}()

	// Start the created container.
	err = client.StartContainer(container.ID, &docker.HostConfig{})
	if checkErr("Started the container", err) {
		job.Status = StatusError
		updateJob("status")
		return
	}

	status, err := client.WaitContainer(container.ID)
	if checkErr("Waited for the container to complete", err) {
		job.Status = StatusError
		updateJob("status")
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

	// Extract the result from the job.
	if job.ResultSource == "stdout" {
		job.Result = []byte(job.Stdout)
		debug("Acquired job result from stdout: ok")
	} else if strings.HasPrefix(job.ResultSource, "file:") {
		resultPath := job.ResultSource[len("file:"):len(job.ResultSource)]

		var resultBuffer bytes.Buffer
		err = client.CopyFromContainer(docker.CopyFromContainerOptions{
			Container:    container.ID,
			Resource:     resultPath,
			OutputStream: &resultBuffer,
		})
		if checkErr(fmt.Sprintf("Acquired the job's result from the file [%s]", resultPath), err) {
			job.Status = StatusError
		} else {
			// CopyFromContainer returns the file contents as a tarball.
			var content bytes.Buffer
			r := bytes.NewReader(resultBuffer.Bytes())
			tr := tar.NewReader(r)

			for {
				_, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					reportErr("Read tar-encoded content: ERROR", err)
					job.Status = StatusError
					break
				}

				if _, err = io.Copy(&content, tr); err != nil {
					reportErr("Copy decoded content: ERROR", err)
					job.Status = StatusError
					break
				}
			}

			job.Result = content.Bytes()
		}
	}

	err = client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
	checkErr("Removed the container", err)

	updateJob("status and final result")
	log.WithFields(log.Fields{"jid": job.JID}).Info("Job complete.")
}
