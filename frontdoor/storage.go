package main

import "gopkg.in/mgo.v2"

// Storage enumerates interactions with the storage engine, and allows us to interject in-memory
// substitutes for testing.
type Storage interface {
	Bootstrap() error

	InsertJob(SubmittedJob) (uint, error)
}

// MongoStorage is a Storage implementation that connects to a real MongoDB cluster.
type MongoStorage struct {
	Database *mgo.Database
}

// NewMongoStorage establishes a connection to the MongoDB cluster.
func NewMongoStorage(c *Context) (*MongoStorage, error) {
	session, err := mgo.Dial(c.Settings.MongoURL)
	if err != nil {
		return nil, err
	}
	return &MongoStorage{Database: session.DB("rho")}, nil
}

func (storage *MongoStorage) jobs() *mgo.Collection {
	return storage.Database.C("jobs")
}

// Bootstrap creates indices and metadata objects.
func (storage *MongoStorage) Bootstrap() error {
	return nil
}

// Job storage

// InsertJob appends a job to the queue and returns a newly allocated job ID.
func (storage *MongoStorage) InsertJob(job SubmittedJob) (uint, error) {
	if err := storage.jobs().Insert(job); err != nil {
		return 0, err
	}

	return 0, nil
}

// NullStorage is a useful embeddable struct that can be used to mock selected storage calls without
// needing to stub out all of the ones you don't care about.
type NullStorage struct{}

// InsertJob is a no-op.
func (storage *NullStorage) InsertJob(job SubmittedJob) (uint, error) {
	return 0, nil
}
