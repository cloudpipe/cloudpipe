package main

import (
	"os"
	"os/user"
	"path"
	"testing"
)

func TestLoadFromEnvironment(t *testing.T) {
	c := Context{}

	os.Setenv("PIPE_PORT", "1234")
	os.Setenv("PIPE_LOGLEVEL", "debug")
	os.Setenv("PIPE_MONGOURL", "server.example.com")
	os.Setenv("PIPE_ADMINNAME", "fake")
	os.Setenv("PIPE_ADMINKEY", "12345")
	os.Setenv("PIPE_POLL", "5000")
	os.Setenv("PIPE_IMAGE", "cloudpipe/runner-py2")
	os.Setenv("PIPE_DOCKERHOST", "tcp://1.2.3.4:4567/")
	os.Setenv("PIPE_DOCKERTLS", "true")
	os.Setenv("PIPE_DOCKERCACERT", "/lockbox/ca.pem")
	os.Setenv("PIPE_DOCKERCERT", "/lockbox/cert.pem")
	os.Setenv("PIPE_DOCKERKEY", "/lockbox/key.pem")

	if err := c.Load(); err != nil {
		t.Errorf("Error loading configuration: %v", err)
	}

	if c.Port != 1234 {
		t.Errorf("Unexpected port: [%d]", c.Port)
	}

	if c.LogLevel != "debug" {
		t.Errorf("Unexpected log level: [%s]", c.LogLevel)
	}

	if c.MongoURL != "server.example.com" {
		t.Errorf("Unexpected MongoDB URL: [%s]", c.MongoURL)
	}

	if c.Poll != 5000 {
		t.Errorf("Unexpected polling interval: [%d]", c.Poll)
	}

	if c.DockerHost != "tcp://1.2.3.4:4567/" {
		t.Errorf("Unexpected docker host: [%s]", c.DockerHost)
	}

	if !c.DockerTLS {
		t.Errorf("Expected docker TLS to be enabled.")
	}

	if c.DockerCACert != "/lockbox/ca.pem" {
		t.Errorf("Unexpected docker CA cert: [%s]", c.DockerCACert)
	}

	if c.DockerCert != "/lockbox/cert.pem" {
		t.Errorf("Unexpected docker cert: [%s]", c.DockerCert)
	}

	if c.DockerKey != "/lockbox/key.pem" {
		t.Errorf("Unexpected docker key: [%s]", c.DockerKey)
	}

	if c.Image != "cloudpipe/runner-py2" {
		t.Errorf("Unexpected image: [%s]", c.Image)
	}

	if c.AdminName != "fake" {
		t.Errorf("Unexpected administrator name: [%s]", c.AdminName)
	}

	if c.AdminKey != "12345" {
		t.Errorf("Unexpected administrator API key: [%s]", c.AdminKey)
	}
}

func TestDefaultValues(t *testing.T) {
	c := Context{}

	os.Setenv("PIPE_PORT", "")
	os.Setenv("PIPE_LOGLEVEL", "")
	os.Setenv("PIPE_MONGOURL", "")
	os.Setenv("PIPE_ADMINNAME", "")
	os.Setenv("PIPE_ADMINKEY", "")
	os.Setenv("PIPE_POLL", "")
	os.Setenv("PIPE_DOCKERHOST", "")
	os.Setenv("DOCKER_HOST", "")
	os.Setenv("PIPE_DOCKERTLS", "")
	os.Setenv("PIPE_DOCKERCACERT", "")
	os.Setenv("PIPE_DOCKERCERT", "")
	os.Setenv("PIPE_DOCKERKEY", "")
	os.Setenv("DOCKER_TLS_VERIFY", "")
	os.Setenv("DOCKER_CERT_PATH", "")
	os.Setenv("PIPE_IMAGE", "")

	u, err := user.Current()
	if err != nil {
		t.Errorf("Unable to identify current user: %v", err)
	}
	home := u.HomeDir

	if err := c.Load(); err != nil {
		t.Errorf("Error loading configuration: %v", err)
	}

	if c.Port != 8000 {
		t.Errorf("Unexpected port: [%d]", c.Port)
	}

	if c.LogLevel != "info" {
		t.Errorf("Unexpected logging level: [%s]", c.LogLevel)
	}

	if c.MongoURL != "mongo" {
		t.Errorf("Unexpected MongoDB connection URL: [%s]", c.MongoURL)
	}

	if c.Poll != 500 {
		t.Errorf("Unexpected polling interval: [%d]", c.Poll)
	}

	if c.DockerHost != "unix:///var/run/docker.sock" {
		t.Errorf("Unexpected docker host: [%s]", c.DockerHost)
	}

	if c.DockerTLS {
		t.Errorf("Expected docker TLS to be disabled.")
	}

	if c.DockerCACert != path.Join(home, ".docker", "ca.pem") {
		t.Errorf("Unexpected docker CA cert: [%s]", c.DockerCACert)
	}

	if c.DockerCert != path.Join(home, ".docker", "cert.pem") {
		t.Errorf("Unexpected docker cert: [%s]", c.DockerCert)
	}

	if c.DockerKey != path.Join(home, ".docker", "key.pem") {
		t.Errorf("Unexpected docker key: [%s]", c.DockerKey)
	}

	if c.Image != "cloudpipe/runner-py2" {
		t.Errorf("Unexpected default image: [%s]", c.Image)
	}
}

func TestUseDockerHost(t *testing.T) {
	os.Setenv("PIPE_DOCKERHOST", "")
	os.Setenv("DOCKER_HOST", "tcp://1.2.3.4:4567/")

	c := Context{}
	if err := c.Load(); err != nil {
		t.Errorf("Error loading configuration: %v", err)
	}

	if c.DockerHost != "tcp://1.2.3.4:4567/" {
		t.Errorf("Unexpected docker host: [%s]", c.DockerHost)
	}
}

func TestAddressString(t *testing.T) {
	c := Context{
		Settings: Settings{Port: 1234},
	}

	if c.ListenAddr() != ":1234" {
		t.Errorf("Unexpected listen address: %s", c.ListenAddr())
	}
}

func TestValidateLogLevel(t *testing.T) {
	c := Context{}

	os.Setenv("PIPE_LOGLEVEL", "Walrus")

	err := c.Load()
	if err == nil {
		t.Errorf("Expected an error when loading an invalid PIPE_LOG_LEVEL.")
	}
}
