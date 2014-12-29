package main

import (
	"os"
	"testing"
)

func TestLoadFromEnvironment(t *testing.T) {
	c := Context{}

	os.Setenv("RHO_PORT", "1234")
	os.Setenv("RHO_LOGLEVEL", "debug")
	os.Setenv("RHO_MONGOURL", "server.example.com")
	os.Setenv("RHO_ADMINNAME", "fake")
	os.Setenv("RHO_ADMINKEY", "12345")
	os.Setenv("RHO_POLL", "5000")
	os.Setenv("RHO_IMAGE", "rgbkrk/inrhocloud")
	os.Setenv("RHO_DOCKERHOST", "tcp://1.2.3.4:4567/")
	os.Setenv("RHO_WEB", "true")
	os.Setenv("RHO_RUNNER", "true")

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

	if c.Image != "rgbkrk/inrhocloud" {
		t.Errorf("Unexpected image: [%s]", c.Image)
	}

	if c.AdminName != "fake" {
		t.Errorf("Unexpected administrator name: [%s]", c.AdminName)
	}

	if c.AdminKey != "12345" {
		t.Errorf("Unexpected administrator API key: [%s]", c.AdminKey)
	}

	if !c.Web {
		t.Error("Expected Web to be enabled.")
	}

	if !c.Runner {
		t.Error("Expected Runner to be enabled.")
	}
}

func TestDefaultValues(t *testing.T) {
	c := Context{}

	os.Setenv("RHO_PORT", "")
	os.Setenv("RHO_LOGLEVEL", "")
	os.Setenv("RHO_MONGOURL", "")
	os.Setenv("RHO_ADMINNAME", "")
	os.Setenv("RHO_ADMINKEY", "")
	os.Setenv("RHO_POLL", "")
	os.Setenv("RHO_DOCKERHOST", "")
	os.Setenv("DOCKER_HOST", "")
	os.Setenv("RHO_IMAGE", "")
	os.Setenv("RHO_WEB", "")
	os.Setenv("RHO_RUNNER", "")

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

	if c.Image != "rgbkrk/inrhocloud" {
		t.Errorf("Unexpected default image: [%s]", c.Image)
	}

	if !c.Web {
		t.Error("Expected Web to be enabled.")
	}
	if !c.Runner {
		t.Error("Expected Runner to be enabled.")
	}
}

func TestOnlyWeb(t *testing.T) {
	os.Setenv("RHO_WEB", "true")
	os.Setenv("RHO_RUNNER", "")

	c := Context{}
	if err := c.Load(); err != nil {
		t.Errorf("Error loading configuration: %v", err)
	}

	if !c.Web {
		t.Error("Expected Web to be enabled.")
	}
	if c.Runner {
		t.Error("Expected Runner to be disabled.")
	}
}

func TestOnlyRunner(t *testing.T) {
	os.Setenv("RHO_WEB", "")
	os.Setenv("RHO_RUNNER", "true")

	c := Context{}
	if err := c.Load(); err != nil {
		t.Errorf("Error loading configuration: %v", err)
	}

	if c.Web {
		t.Error("Expected Web to be disabled.")
	}
	if !c.Runner {
		t.Error("Expected Runner to be enabled.")
	}
}

func TestUseDockerHost(t *testing.T) {
	os.Setenv("RHO_DOCKERHOST", "")
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

	os.Setenv("RHO_LOGLEVEL", "Walrus")

	err := c.Load()
	if err == nil {
		t.Errorf("Expected an error when loading an invalid RHO_LOG_LEVEL.")
	}
}
