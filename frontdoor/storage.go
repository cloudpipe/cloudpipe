package main

import "gopkg.in/mgo.v2"

// Storage enumerates interactions with the storage engine, and allows us to interject in-memory
// substitutes for testing.
type Storage interface {
	Connect(*Context) error

	InsertJob(SubmittedJob) (uint, error)
}

// MongoStorage is a Storage implementation that connects to a real MongoDB cluster.
type MongoStorage struct {
	Database *mgo.Database
}

// Connect establishes a connection to the cluster.
func (storage *MongoStorage) Connect(c *Context) error {
	session, err := mgo.Dial(c.Settings.MongoURL)
	if err != nil {
		return err
	}
	storage.Database = session.DB("rho")
	return nil
}

// Job storage

// InsertJob appends a job to the queue and returns a newly allocated job ID.
func (storage *MongoStorage) InsertJob(job SubmittedJob) (uint, error) {
	if err := storage.Database.C("jobs").Insert(job); err != nil {
		return 0, err
	}

	return 0, nil
}

// NullStorage is a useful embeddable struct that can be used to mock selected storage calls without
// needing to stub out all of the ones you don't care about.
type NullStorage struct{}

// Connect always succeeds.
func (storage NullStorage) Connect(c *Context) error {
	return nil
}
