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

	if err := c.Load(); err != nil {
		t.Errorf("Error loading configuration: %v", err)
	}

	if c.Settings.Port != 1234 {
		t.Errorf("Unexpected port: [%d]", c.Settings.Port)
	}

	if c.Settings.LogLevel != "debug" {
		t.Errorf("Unexpected log level: [%s]", c.Settings.LogLevel)
	}

	if c.Settings.MongoURL != "server.example.com" {
		t.Errorf("Unexpected MongoDB URL: [%s]", c.Settings.MongoURL)
	}

	if c.Settings.AdminName != "fake" {
		t.Errorf("Unexpected administrator name: [%s]", c.Settings.AdminName)
	}

	if c.Settings.AdminKey != "12345" {
		t.Errorf("Unexpected administrator API key: [%s]", c.Settings.AdminKey)
	}
}

func TestDefaultValues(t *testing.T) {
	c := Context{}

	os.Setenv("RHO_PORT", "")
	os.Setenv("RHO_LOGLEVEL", "")
	os.Setenv("RHO_MONGOURL", "")
	os.Setenv("RHO_ADMINNAME", "")
	os.Setenv("RHO_ADMINKEY", "")

	if err := c.Load(); err != nil {
		t.Errorf("Error loading configuration: %v", err)
	}

	if c.Settings.Port != 8000 {
		t.Errorf("Unexpected port: [%d]", c.Settings.Port)
	}

	if c.Settings.LogLevel != "info" {
		t.Errorf("Unexpected logging level: [%s]", c.Settings.LogLevel)
	}

	if c.Settings.MongoURL != "mongo" {
		t.Errorf("Unexpected MongoDB connection URL: [%s]", c.Settings.MongoURL)
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
