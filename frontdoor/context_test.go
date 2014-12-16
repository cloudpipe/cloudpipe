package main

import (
	"os"
	"testing"
)

func TestLoadFromEnvironment(t *testing.T) {
	c := Context{}

	os.Setenv("RHO_PORT", "1234")
	os.Setenv("RHO_LOGLEVEL", "debug")

	if err := c.Load(); err != nil {
		t.Errorf("Error loading configuration: %v", err)
	}

	if c.Settings.Port != 1234 {
		t.Errorf("Unexpected port: %d", c.Settings.Port)
	}

	if c.Settings.LogLevel != "debug" {
		t.Errorf("Unexpected log level: %s", c.Settings.LogLevel)
	}
}

func TestDefaultValues(t *testing.T) {
	c := Context{}

	os.Setenv("RHO_PORT", "")
	os.Setenv("RHO_LOGLEVEL", "")

	if err := c.Load(); err != nil {
		t.Errorf("Error loading configuration: %v", err)
	}

	if c.Settings.Port != 8000 {
		t.Errorf("Unexpected port: %d", c.Settings.Port)
	}

	if c.Settings.LogLevel != "info" {
		t.Errorf("Unexpected logging level: %s", c.Settings.LogLevel)
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
