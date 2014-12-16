package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kelseyhightower/envconfig"
)

// Context provides shared state among individual route handlers.
type Context struct {
	Settings Settings
}

// Settings contains configuration options loaded from the environment.
type Settings struct {
	Port     int
	LogLevel string
}

// Load configuration settings from the environment, apply defaults, and validate them.
func (c *Context) Load() error {
	if err := envconfig.Process("RHO", &c.Settings); err != nil {
		return err
	}

	if c.Settings.Port == 0 {
		c.Settings.Port = 8000
	}

	if c.Settings.LogLevel == "" {
		c.Settings.LogLevel = "info"
	}

	if _, err := log.ParseLevel(c.Settings.LogLevel); err != nil {
		return err
	}

	return nil
}
