package main

import (
	docker "github.com/smashwilson/go-dockerclient"
)

// Docker enumerates interactions with the Docker client, allowing us to use alternate
// implementations for testing.
type Docker interface {
	CreateContainer(docker.CreateContainerOptions) (*docker.Container, error)
	AttachToContainer(docker.AttachToContainerOptions) error
	StartContainer(string, *docker.HostConfig) error
	WaitContainer(string) (int, error)
	CopyFromContainer(docker.CopyFromContainerOptions) error
	RemoveContainer(docker.RemoveContainerOptions) error
}

// NullDocker is an embeddable struct that implements the full Docker interface as no-ops, allowing
// you to only implement the calls that you care about for a specific test.
type NullDocker struct{}

// CreateContainer is a no-op that always returns nil and no error.
func (n NullDocker) CreateContainer(docker.CreateContainerOptions) (*docker.Container, error) {
	return nil, nil
}

// AttachToContainer is a no-op.
func (n NullDocker) AttachToContainer(docker.AttachToContainerOptions) error {
	return nil
}

// StartContainer is a no-op.
func (n NullDocker) StartContainer(string, *docker.HostConfig) error {
	return nil
}

// WaitContainer is a no-op.
func (n NullDocker) WaitContainer(string) (int, error) {
	return 0, nil
}

// CopyFromContainer is a no-op.
func (n NullDocker) CopyFromContainer(docker.CopyFromContainerOptions) error {
	return nil
}

// RemoveContainer is a no-op.
func (n NullDocker) RemoveContainer(docker.RemoveContainerOptions) error {
	return nil
}

// Ensure that NullDocker adheres to the Docker interface.
var _ Docker = NullDocker{}
