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
	for {
		Claim(c)

		time.Sleep(time.Duration(c.Poll) * time.Millisecond)
	}
}

// Claim acquires the oldest single pending job and launches a goroutine to execute its command in
// a new container.
func Claim(c *Context) {
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

	go Execute(c, job)
}

// Execute launches a container to process the submitted job. It passes any provided stdin data
// to the container and consumes stdout and stderr, updating Mongo as it runs. Once completed, it
// acquires the job's result from its configured source and marks the job as finished.
func Execute(c *Context, job *SubmittedJob) {
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

	// Update the job model in Mongo, reporting any errors along the way.
	// This also updates our job model with any changes from Mongo, such as the kill request flag.
	updateJob := func(message string) bool {
		if err := c.UpdateJob(job); err != nil {
			reportErr(fmt.Sprintf("Unable to update the job's %s.", message), err)
			return false
		}
		return true
	}

	log.WithFields(defaultFields).Info("Launching a job.")

	job.StartedAt = StoreTime(time.Now())
	job.QueueDelay = job.StartedAt.AsTime().Sub(job.CreatedAt.AsTime()).Nanoseconds()

	container, err := c.CreateContainer(docker.CreateContainerOptions{
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

	// Record the job's container ID.
	job.ContainerID = container.ID
	if !updateJob("start timestamp and container id") {
		return
	}

	// Include container information in this job's logging messages.
	defaultFields["container id"] = container.ID
	defaultFields["container name"] = container.Name

	// Was a kill requested between the time the job was claimed, and the time the container was
	// created? If so: transition the job to StatusKilled and jump ahead to removing the container
	// we just created. If not: continue with job execution normally.

	// If a kill was requested before the job was claimed, it would have been removed from the queue.
	// If a kill is requested after the container was created, it will have the containerID that we
	// just sent and be able to kill the running container.

	if job.KillRequested {
		job.Status = StatusKilled
	} else {
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
			err = c.AttachToContainer(docker.AttachToContainerOptions{
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
		err = c.StartContainer(container.ID, &docker.HostConfig{})
		if checkErr("Started the container", err) {
			job.Status = StatusError
			updateJob("status")
			return
		}

		// Measure the container-launch overhead here.
		overhead := time.Now()
		job.OverheadDelay = overhead.Sub(job.StartedAt.AsTime()).Nanoseconds()
		updateJob("overhead delay")

		status, err := c.WaitContainer(container.ID)
		if checkErr("Waited for the container to complete", err) {
			job.Status = StatusError
			updateJob("status")
			return
		}

		job.FinishedAt = StoreTime(time.Now())
		job.Runtime = job.FinishedAt.AsTime().Sub(overhead).Nanoseconds()
		if status == 0 {
			// Successful termination.
			job.Status = StatusDone
		} else {
			// Something went wrong.

			// See if a kill was explicitly requested. If so, transition to StatusKilled. Otherwise,
			// transition to StatusError.
			killed, err := c.JobKillRequested(job.JID)
			if err != nil {
				reportErr("Check the job kill status: ERROR", err)
				return
			}

			if killed {
				job.Status = StatusKilled
			} else {
				job.Status = StatusError
			}
		}

		// Extract the result from the job.
		if job.ResultSource == "stdout" {
			job.Result = []byte(job.Stdout)
			debug("Acquired job result from stdout: ok")
		} else if strings.HasPrefix(job.ResultSource, "file:") {
			resultPath := job.ResultSource[len("file:"):len(job.ResultSource)]

			var resultBuffer bytes.Buffer
			err = c.CopyFromContainer(docker.CopyFromContainerOptions{
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

		// Job execution has completed successfully.
	}

	err = c.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
	checkErr("Removed the container", err)

	err = c.UpdateAccountUsage(job.Account, job.Runtime)
	if err != nil {
		reportErr("Update account usage: ERROR", err)
		return
	}
	updateJob("status and final result")

	log.WithFields(log.Fields{
		"jid":      job.JID,
		"account":  job.Account,
		"status":   job.Status,
		"runtime":  job.Runtime,
		"overhead": job.OverheadDelay,
		"queue":    job.QueueDelay,
	}).Info("Job complete.")
}
