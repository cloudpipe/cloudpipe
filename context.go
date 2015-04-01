package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/kelseyhightower/envconfig"
)

// Context provides shared state among individual route handlers.
type Context struct {
	// Configuration settings from the environment.
	Settings

	// Service facades.
	Storage
	Docker

	// Shared clients.
	HTTPS       *http.Client
	AuthService AuthService
}

// Settings contains configuration options loaded from the environment.
type Settings struct {
	Port         int
	LogLevel     string
	LogColors    bool
	MongoURL     string
	AdminName    string
	AdminKey     string
	DockerHost   string
	DockerTLS    bool
	CACert       string
	Cert         string
	Key          string
	DefaultImage string
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

	// Configure the logging level and formatter.

	level, err := log.ParseLevel(c.LogLevel)
	if err != nil {
		return c, err
	}
	log.SetLevel(level)

	log.SetFormatter(&log.TextFormatter{
		ForceColors: c.LogColors,
	})

	// Summarize the loaded settings.

	log.WithFields(log.Fields{
		"port":               c.Port,
		"logging level":      c.LogLevel,
		"log with color":     c.LogColors,
		"mongo URL":          c.MongoURL,
		"admin account":      c.AdminName,
		"docker host":        c.DockerHost,
		"docker TLS enabled": c.DockerTLS,
		"CA cert":            c.CACert,
		"cert":               c.Cert,
		"key":                c.Key,
		"default image":      c.DefaultImage,
		"polling interval":   c.Poll,
		"auth service":       c.Settings.AuthService,
	}).Info("Initializing with loaded settings.")

	// Configure a HTTP(S) client to use the provided TLS credentials.

	caCertPool := x509.NewCertPool()

	caCertPEM, err := ioutil.ReadFile(c.CACert)
	if err != nil {
		log.Debug("Hint: if you're running in dev mode, try running script/genkeys first.")
		return nil, fmt.Errorf("unable to load CA certificate: %v", err)
	}
	caCertPool.AppendCertsFromPEM(caCertPEM)

	keypair, err := tls.LoadX509KeyPair(c.Cert, c.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to load TLS keypair: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       []tls.Certificate{keypair},
		MinVersion:         tls.VersionTLS10,
		InsecureSkipVerify: false,
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	c.HTTPS = &http.Client{Transport: transport}

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
		c.Docker, err = docker.NewTLSClient(c.DockerHost, c.Cert, c.Key, c.CACert)
		if err != nil {
			log.WithFields(log.Fields{
				"docker host": c.DockerHost,
			}).Fatal("Unable to connect to Docker with TLS.")
			return c, err
		}
	} else {
		c.Docker, err = docker.NewClient(c.DockerHost)
		if err != nil {
			log.WithFields(log.Fields{
				"docker host": c.DockerHost,
				"error":       err,
			}).Error("Unable to connect to Docker.")
			return c, err
		}
	}

	// Initialize an appropriate authentication service.
	c.AuthService, err = ConnectToAuthService(c, c.Settings.AuthService)
	if err != nil {
		log.WithFields(log.Fields{
			"auth service url": c.Settings.AuthService,
			"error":            err,
		}).Error("Unable to connect to authentication service.")
		return c, err
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
		certRoot = "/certificates"
	}

	if c.CACert == "" {
		c.CACert = path.Join(certRoot, "ca.pem")
	}

	if c.Cert == "" {
		c.Cert = path.Join(certRoot, "cloudpipe-cert.pem")
	}

	if c.Key == "" {
		c.Key = path.Join(certRoot, "cloudpipe-key.pem")
	}

	if c.DefaultImage == "" {
		c.DefaultImage = "cloudpipe/runner-py2"
	}

	if c.Settings.AuthService == "" {
		c.Settings.AuthService = "https://authstore:9001/v1"
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
