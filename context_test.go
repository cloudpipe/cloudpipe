package main

import (
	"os"
	"testing"
)

func TestLoadFromEnvironment(t *testing.T) {
	c := Context{}

	os.Setenv("PIPE_PORT", "1234")
	os.Setenv("PIPE_LOGLEVEL", "debug")
	os.Setenv("PIPE_LOGCOLORS", "true")
	os.Setenv("PIPE_MONGOURL", "server.example.com")
	os.Setenv("PIPE_ADMINNAME", "fake")
	os.Setenv("PIPE_ADMINKEY", "12345")
	os.Setenv("PIPE_POLL", "5000")
	os.Setenv("PIPE_IMAGE", "cloudpipe/runner-py2")
	os.Setenv("PIPE_DOCKERHOST", "tcp://1.2.3.4:4567/")
	os.Setenv("PIPE_DOCKERTLS", "true")
	os.Setenv("PIPE_CACERT", "/lockbox/ca.pem")
	os.Setenv("PIPE_CERT", "/lockbox/cert.pem")
	os.Setenv("PIPE_KEY", "/lockbox/key.pem")
	os.Setenv("PIPE_AUTHSERVICE", "https://auth")

	if err := c.Load(); err != nil {
		t.Errorf("Error loading configuration: %v", err)
	}

	if c.Port != 1234 {
		t.Errorf("Unexpected port: [%d]", c.Port)
	}

	if c.LogLevel != "debug" {
		t.Errorf("Unexpected log level: [%s]", c.LogLevel)
	}

	if !c.LogColors {
		t.Error("Expected log coloring to be enabled")
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

	if c.CACert != "/lockbox/ca.pem" {
		t.Errorf("Unexpected docker CA cert: [%s]", c.CACert)
	}

	if c.Cert != "/lockbox/cert.pem" {
		t.Errorf("Unexpected docker cert: [%s]", c.Cert)
	}

	if c.Key != "/lockbox/key.pem" {
		t.Errorf("Unexpected docker key: [%s]", c.Key)
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

	if c.Settings.AuthService != "https://auth" {
		t.Errorf("Unexpected authentication service URL: [%s]", c.AuthService)
	}
}

func TestDefaultValues(t *testing.T) {
	c := Context{}

	os.Setenv("PIPE_PORT", "")
	os.Setenv("PIPE_LOGLEVEL", "")
	os.Setenv("PIPE_LOGCOLORS", "")
	os.Setenv("PIPE_MONGOURL", "")
	os.Setenv("PIPE_ADMINNAME", "")
	os.Setenv("PIPE_ADMINKEY", "")
	os.Setenv("PIPE_POLL", "")
	os.Setenv("PIPE_DOCKERHOST", "")
	os.Setenv("DOCKER_HOST", "")
	os.Setenv("PIPE_DOCKERTLS", "")
	os.Setenv("PIPE_CACERT", "")
	os.Setenv("PIPE_CERT", "")
	os.Setenv("PIPE_KEY", "")
	os.Setenv("DOCKER_TLS_VERIFY", "")
	os.Setenv("DOCKER_CERT_PATH", "")
	os.Setenv("PIPE_IMAGE", "")
	os.Setenv("PIPE_AUTHSERVICE", "")

	if err := c.Load(); err != nil {
		t.Errorf("Error loading configuration: %v", err)
	}

	if c.Port != 8000 {
		t.Errorf("Unexpected port: [%d]", c.Port)
	}

	if c.LogLevel != "info" {
		t.Errorf("Unexpected logging level: [%s]", c.LogLevel)
	}

	if c.LogColors {
		t.Error("Expected logging colors to be disabled by default")
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

	if c.CACert != "/certificates/ca.pem" {
		t.Errorf("Unexpected docker CA cert: [%s]", c.CACert)
	}

	if c.Cert != "/certificates/cloudpipe-cert.pem" {
		t.Errorf("Unexpected docker cert: [%s]", c.Cert)
	}

	if c.Key != "/certificates/cloudpipe-key.pem" {
		t.Errorf("Unexpected docker key: [%s]", c.Key)
	}

	if c.Image != "cloudpipe/runner-py2" {
		t.Errorf("Unexpected default image: [%s]", c.Image)
	}

	if c.Settings.AuthService != "https://authstore:9001/v1" {
		t.Errorf("Unexpected default auth service: [%s]", c.AuthService)
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
