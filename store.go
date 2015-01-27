package cion

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"
)

// JobStore write and reads jobs and job logs from persistent storage.
type JobStore interface {
	// GetByID gets a job by its unique ID.
	GetByID(id uint) (*Job, error)

	// GetByNumber gets a job for the given owner/repo/branch by its number.
	GetByNumber(owner, repo, branch string, number uint) (*Job, error)

	// Lists gets all the jobs for the given owner/repo/branch.
	List(owner, repo, branch string) ([]*Job, error)

	// Save persists a job to storage. If the job doesn't have an ID yet, it is assigned a
	// unique ID and the next incrementing job number for the owner/repo/branch.
	Save(j *Job) error

	// GetLogger gets the JobLogger to write logs for a job.
	GetLogger(j *Job) JobLogger
}

// JobLogger provides an io.Writer interface for writing build logs for a job.
type JobLogger interface {
	io.Writer

	// WriteStep writes a transition to a new build step to the log. All subsequent writes are
	// assumed to be part of the new build step, until another new step is written.
	WriteStep(name string) error
}

// InMemoryJobStore is a mock JobStore that only stores jobs in memory and writes logs directly
// to stdout. It is only meant to be used for testing.
type InMemoryJobStore struct {
	jobCounter         *uint64
	jobCounterByBranch map[string]*uint64
	jobsByID           map[uint64]*Job
	jobsByNumber       map[string]map[uint64]*Job
}

func (s *InMemoryJobStore) GetByID(id uint) (*Job, error) {
	return s.jobsByID[uint64(id)], nil
}

func (s *InMemoryJobStore) GetByNumber(owner, repo, branch string, number uint) (*Job, error) {
	jobs, err := s.List(owner, repo, branch)
	if err != nil {
		return nil, err
	}

	for _, j := range jobs {
		if j.Number == number {
			return j, nil
		}
	}

	return nil, nil
}

func (s *InMemoryJobStore) List(owner, repo, branch string) ([]*Job, error) {
	jbn := s.jobsByNumber[getKey(owner, repo, branch)]
	l := make([]*Job, len(jbn))

	for _, j := range jbn {
		l = append(l, j)
	}

	return l, nil
}

func (s *InMemoryJobStore) Save(j *Job) error {
	if j.ID != 0 {
		// the job is in memory, nothing to persist
		return nil
	}

	id := atomic.AddUint64(s.jobCounter, 1)
	number := atomic.AddUint64(s.jobCounterByBranch[getKey(j.Owner, j.Repo, j.Branch)], 1)

	j.ID = uint(id)
	j.Number = uint(number)

	s.jobsByID[id] = j
	s.jobsByNumber[getKey(j.Owner, j.Repo, j.Branch)][number] = j

	return nil
}

func (s *InMemoryJobStore) GetLogger(j *Job) JobLogger {
	return NewWriterLogger(os.Stdout)
}

func getKey(owner, repo, branch string) string {
	return owner + ":" + repo + ":" + branch
}

type WriterLogger struct {
	w io.Writer
}

func NewWriterLogger(w io.Writer) WriterLogger {
	return WriterLogger{w: w}
}

func (wl WriterLogger) Write(p []byte) (int, error) {
	return wl.w.Write(p)
}

func (wl WriterLogger) WriteStep(name string) error {
	s := fmt.Sprintln("CION:", name)
	_, err := io.WriteString(wl.w, s)
	return err
}
