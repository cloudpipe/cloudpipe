package main

import (
	"fmt"
	"os"
	"os/user"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/kelseyhightower/envconfig"
	docker "github.com/smashwilson/go-dockerclient"
)

// Context provides shared state among individual route handlers.
type Context struct {
	Settings
	Storage
	Docker
}

// Settings contains configuration options loaded from the environment.
type Settings struct {
	Port         int
	LogLevel     string
	MongoURL     string
	AdminName    string
	AdminKey     string
	DockerHost   string
	DockerTLS    bool
	DockerCACert string
	DockerCert   string
	DockerKey    string
	Image        string
	Poll         int
	AuthService  string
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
		"port":               c.Port,
		"logging level":      c.LogLevel,
		"mongo URL":          c.MongoURL,
		"admin account":      c.AdminName,
		"docker host":        c.DockerHost,
		"docker TLS enabled": c.DockerTLS,
		"docker CA cert":     c.DockerCACert,
		"docker cert":        c.DockerCert,
		"docker key":         c.DockerKey,
		"default layer":      c.Image,
		"polling interval":   c.Poll,
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

	// Connect to Docker.

	if c.DockerTLS {
		c.Docker, err = docker.NewTLSClient(c.DockerHost, c.DockerCert, c.DockerKey, c.DockerCACert)
		if err != nil {
			log.WithFields(log.Fields{
				"docker host":    c.DockerHost,
				"docker cert":    c.DockerCert,
				"docker key":     c.DockerKey,
				"docker CA cert": c.DockerCACert,
			}).Fatal("Unable to connect to Docker with TLS.")
			return c, err
		}
	} else {
		c.Docker, err = docker.NewClient(c.DockerHost)
		if err != nil {
			log.WithFields(log.Fields{
				"docker host": c.DockerHost,
				"error":       err,
			}).Fatal("Unable to connect to Docker.")
			return c, err
		}
	}

	return c, nil
}

// Load configuration settings from the environment, apply defaults, and validate them.
func (c *Context) Load() error {
	if err := envconfig.Process("PIPE", &c.Settings); err != nil {
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

	if c.Poll == 0 {
		c.Poll = 500
	}

	if c.DockerHost == "" {
		if host := os.Getenv("DOCKER_HOST"); host != "" {
			c.DockerHost = host
		} else {
			c.DockerHost = "unix:///var/run/docker.sock"
		}
	}

	certRoot := os.Getenv("DOCKER_CERT_PATH")
	if certRoot == "" {
		user, err := user.Current()
		if err != nil {
			return fmt.Errorf("Unable to read the current OS user: %v", err)
		}

		certRoot = path.Join(user.HomeDir, ".docker")
	}

	if c.DockerCACert == "" {
		c.DockerCACert = path.Join(certRoot, "ca.pem")
	}

	if c.DockerCert == "" {
		c.DockerCert = path.Join(certRoot, "cert.pem")
	}

	if c.DockerKey == "" {
		c.DockerKey = path.Join(certRoot, "key.pem")
	}

	if c.Image == "" {
		c.Image = "cloudpipe/runner-py2"
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
