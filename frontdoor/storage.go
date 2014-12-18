package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	log "github.com/Sirupsen/logrus"
)

// Storage enumerates interactions with the storage engine, and allows us to interject in-memory
// substitutes for testing.
type Storage interface {
	Bootstrap() error

	InsertJob(SubmittedJob) (uint64, error)
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

func (storage *MongoStorage) root() *mgo.Collection {
	return storage.Database.C("root")
}

// MongoRoot contains global metadata, counters and statistics used by various storage functions.
// Exactly one instance of MongoRoot should exist in the "root" collection.
type MongoRoot struct {
	JobID uint64 `bson:"job_id"`
}

// Bootstrap creates indices and metadata objects.
func (storage *MongoStorage) Bootstrap() error {
	initial := MongoRoot{}
	var existing MongoRoot

	info, err := storage.root().Find(bson.M{}).Apply(mgo.Change{
		Update: bson.M{"$setOnInsert": &initial},
		Upsert: true,
	}, &existing)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"updated": info.Updated,
		"removed": info.Removed,
	}).Debug("MongoRoot object initialized.")

	return nil
}

// Job storage

// InsertJob appends a job to the queue and returns a newly allocated job ID.
func (storage *MongoStorage) InsertJob(job SubmittedJob) (uint64, error) {
	// Assign the job a job ID.
	var root MongoRoot
	_, err := storage.root().Find(bson.M{}).Apply(mgo.Change{
		Update:    bson.M{"$inc": bson.M{"job_id": 1}},
		ReturnNew: true,
	}, &root)
	if err != nil {
		return 0, err
	}
	job.JID = root.JobID

	if err := storage.jobs().Insert(job); err != nil {
		return 0, err
	}

	return job.JID, nil
}

// NullStorage is a useful embeddable struct that can be used to mock selected storage calls without
// needing to stub out all of the ones you don't care about.
type NullStorage struct{}

// Bootstrap is a no-op.
func (storage *NullStorage) Bootstrap() error {
	return nil
}

// InsertJob is a no-op.
func (storage *NullStorage) InsertJob(job SubmittedJob) (uint64, error) {
	return 0, nil
}
