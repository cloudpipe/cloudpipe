package main

import (
	"fmt"

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

// NewContext loads the active configuration and applies any immediate, global settings like the
// logging level.
func NewContext() (*Context, error) {
	c := &Context{}

	if err := c.Load(); err != nil {
		return c, err
	}

	level, err := log.ParseLevel(c.Settings.LogLevel)
	if err != nil {
		return c, err
	}
	log.SetLevel(level)

	log.WithFields(log.Fields{
		"port":          c.Settings.Port,
		"logging level": c.Settings.LogLevel,
	}).Info("Initializing with loaded settings.")

	return c, nil
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

// ListenAddr generates an address to bind the net/http server to based on the current settings.
func (c *Context) ListenAddr() string {
	return fmt.Sprintf(":%d", c.Settings.Port)
}
