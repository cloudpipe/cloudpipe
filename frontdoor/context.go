package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/kelseyhightower/envconfig"
)

// Context provides shared state among individual route handlers.
type Context struct {
	Settings
	Storage
}

// Settings contains configuration options loaded from the environment.
type Settings struct {
	Port      int
	LogLevel  string
	MongoURL  string
	AdminName string
	AdminKey  string
}

// NewContext loads the active configuration and applies any immediate, global settings like the
// logging level.
func NewContext() (*Context, error) {
	c := &Context{}

	if err := c.Load(); err != nil {
		return c, err
	}

	// Summarize the loaded settings.

	log.WithFields(log.Fields{
		"port":          c.Port,
		"logging level": c.LogLevel,
		"mongo URL":     c.MongoURL,
		"admin account": c.AdminName,
	}).Info("Initializing with loaded settings.")

	// Configure the logging level.

	level, err := log.ParseLevel(c.LogLevel)
	if err != nil {
		return c, err
	}
	log.SetLevel(level)

	// Connect to MongoDB.

	c.Storage, err = NewMongoStorage(c)
	if err != nil {
		return c, err
	}
	if err := c.Storage.Bootstrap(); err != nil {
		return c, err
	}

	return c, nil
}

// Load configuration settings from the environment, apply defaults, and validate them.
func (c *Context) Load() error {
	if err := envconfig.Process("RHO", &c.Settings); err != nil {
		return err
	}

	if c.Port == 0 {
		c.Port = 8000
	}

	if c.LogLevel == "" {
		c.LogLevel = "info"
	}

	if c.MongoURL == "" {
		c.MongoURL = "mongo"
	}

	if _, err := log.ParseLevel(c.LogLevel); err != nil {
		return err
	}

	return nil
}

// ListenAddr generates an address to bind the net/http server to based on the current settings.
func (c *Context) ListenAddr() string {
	return fmt.Sprintf(":%d", c.Port)
}
