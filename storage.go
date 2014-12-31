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
	ListJobs(JobQuery) ([]SubmittedJob, error)
	ClaimJob() (*SubmittedJob, error)
	UpdateJob(*SubmittedJob) error

	GetAccount(name string) (*Account, error)
	UpdateAccountUsage(name string, runtime int64) error
}

// JobQuery specifies (all optional) query parameters for fetching jobs.
type JobQuery struct {
	AccountName string

	JIDs     []uint64
	Names    []string
	Statuses []string

	Limit  int
	Before uint64
	After  uint64
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
	return &MongoStorage{Database: session.DB("pipe")}, nil
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

// ListJobs queries jobs that have been submitted to the cluster.
func (storage *MongoStorage) ListJobs(query JobQuery) ([]SubmittedJob, error) {
	q := bson.M{"account": query.AccountName}

	switch len(query.JIDs) {
	case 0:
		if query.Before != 0 {
			q["_id"] = bson.M{"$lt": query.Before}
		}

		if query.After != 0 {
			q["_id"] = bson.M{"$gte": query.After}
		}
	case 1:
		only := query.JIDs[0]
		if query.Before != 0 && only >= query.Before {
			return []SubmittedJob{}, nil
		}
		if query.After != 0 && only < query.After {
			return []SubmittedJob{}, nil
		}

		q["_id"] = query.JIDs[0]
	default:
		var filtered []uint64

		if query.Before != 0 || query.After != 0 {
			filtered = make([]uint64, 0, len(query.JIDs))
			for _, jid := range query.JIDs {
				if (query.Before == 0 || jid < query.Before) && (query.After == 0 || jid >= query.After) {
					filtered = append(filtered, jid)
				}
			}

			if len(filtered) == 0 {
				return []SubmittedJob{}, nil
			}
		} else {
			filtered = query.JIDs
		}

		q["_id"] = bson.M{"$in": filtered}
	}

	switch len(query.Names) {
	case 0:
	case 1:
		q["job.name"] = query.Names[0]
	default:
		q["job.name"] = bson.M{"$in": query.Names}
	}

	switch len(query.Statuses) {
	case 0:
	case 1:
		q["status"] = query.Statuses[0]
	default:
		q["status"] = bson.M{"$in": query.Statuses}
	}

	var result []SubmittedJob
	if err := storage.jobs().Find(q).Limit(query.Limit).All(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// ClaimJob atomically searches for the oldest pending SubmittedJob, marks it as StatusProcessing,
// and returns it. nil is returned if no SubmittedJobs are available.
func (storage *MongoStorage) ClaimJob() (*SubmittedJob, error) {
	var job SubmittedJob
	_, err := storage.jobs().Find(bson.M{"status": StatusQueued}).Sort("created_at").Apply(mgo.Change{
		Update:    bson.M{"$set": bson.M{"status": StatusProcessing}},
		ReturnNew: true,
	}, &job)

	if err == mgo.ErrNotFound {
		// No jobs in the queue.
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &job, nil
}

// UpdateJob updates the state of a job in the database to match any changes made to the model.
func (storage *MongoStorage) UpdateJob(job *SubmittedJob) error {
	var out SubmittedJob
	_, err := storage.jobs().FindId(job.JID).Apply(mgo.Change{
		Update: bson.M{"$set": job},
	}, &out)
	return err
}

// Account storage

// GetAccount loads an account by its unique account name.
func (storage *MongoStorage) GetAccount(name string) (*Account, error) {
	var out Account
	return &out, nil
}

// UpdateAccountUsage updates an account to take a new job into account.
func (storage *MongoStorage) UpdateAccountUsage(name string, runtime int64) error {
	return nil
}

// NullStorage is a useful embeddable struct that can be used to mock selected storage calls without
// needing to stub out all of the ones you don't care about.
type NullStorage struct{}

// Ensure that NullStorage adheres to the Storage interface.
var _ Storage = NullStorage{}

// Bootstrap is a no-op.
func (storage NullStorage) Bootstrap() error {
	return nil
}

// InsertJob is a no-op.
func (storage NullStorage) InsertJob(job SubmittedJob) (uint64, error) {
	return 0, nil
}

// ListJobs returns an empty collection.
func (storage NullStorage) ListJobs(query JobQuery) ([]SubmittedJob, error) {
	return []SubmittedJob{}, nil
}

// ClaimJob always returns nil.
func (storage NullStorage) ClaimJob() (*SubmittedJob, error) {
	return nil, nil
}

// UpdateJob is a no-op.
func (storage NullStorage) UpdateJob(job *SubmittedJob) error {
	return nil
}

// GetAccount returns a fake, zero-initialized Account.
func (storage NullStorage) GetAccount(name string) (*Account, error) {
	return &Account{Name: name}, nil
}

// UpdateAccountUsage is a no-op.
func (storage NullStorage) UpdateAccountUsage(name string, runtime int64) error {
	return nil
}
