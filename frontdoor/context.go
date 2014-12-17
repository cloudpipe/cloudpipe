package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/kelseyhightower/envconfig"
	mgo "gopkg.in/mgo.v2"
)

// Context provides shared state among individual route handlers.
type Context struct {
	Settings Settings
	Database *mgo.Database
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
		"port":          c.Settings.Port,
		"logging level": c.Settings.LogLevel,
		"mongo URL":     c.Settings.MongoURL,
		"admin account": c.Settings.AdminName,
	}).Info("Initializing with loaded settings.")

	// Configure the logging level.

	level, err := log.ParseLevel(c.Settings.LogLevel)
	if err != nil {
		return c, err
	}
	log.SetLevel(level)

	// Connect to MongoDB.

	session, err := mgo.Dial(c.Settings.MongoURL)
	if err != nil {
		return c, err
	}
	c.Database = session.DB("rho")

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

	if c.Settings.MongoURL == "" {
		c.Settings.MongoURL = "mongo"
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
