package cion

import (
	"fmt"
	"io"
	"os"
)

// InMemoryJobStore is a mock JobStore that only stores jobs in memory and writes logs directly
// to stdout. It is only meant to be used for testing.
type InMemoryJobStore struct {
	jobCounter         uint64
	jobCounterByBranch map[string]uint64
	jobs               map[uint64]*Job
}

func NewInMemoryJobStore() *InMemoryJobStore {
	return &InMemoryJobStore{
		jobCounter:         0,
		jobCounterByBranch: make(map[string]uint64),
		jobs:               make(map[uint64]*Job),
	}
}

func (s *InMemoryJobStore) GetByID(id uint64) (*Job, error) {
	return s.jobs[id], nil
}

func (s *InMemoryJobStore) GetByNumber(owner, repo, branch string, number uint64) (*Job, error) {
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
	var l []*Job

	for _, j := range s.jobs {
		if j.Owner == owner && j.Repo == repo && j.Branch == branch {
			l = append(l, j)
		}
	}

	return l, nil
}

func (s *InMemoryJobStore) Save(j *Job) error {
	if j.ID != 0 {
		// the job is in memory, nothing to persist
		return nil
	}

	// obviously not thread-safe
	s.jobCounter++
	id := s.jobCounter

	k := j.Owner + ":" + j.Repo + ":" + j.Branch
	s.jobCounterByBranch[k]++
	number := s.jobCounterByBranch[k]

	j.ID = id
	j.Number = number

	s.jobs[id] = j

	return nil
}

func (s *InMemoryJobStore) GetLogger(j *Job) JobLogger {
	return NewWriterLogger(os.Stdout)
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
	_, err := wl.Write([]byte(s))
	return err
}
