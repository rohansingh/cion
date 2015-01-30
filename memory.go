package cion

import (
	"fmt"
	"io"
	"os"
)

// InMemoryJobStore is a mock JobStore that only stores jobs in memory and writes logs directly
// to stdout. It is only meant to be used for testing.
type InMemoryJobStore struct {
	jobCounter       uint64
	jobCounterByRepo map[string]uint64
	jobs             map[uint64]*Job
}

func NewInMemoryJobStore() *InMemoryJobStore {
	return &InMemoryJobStore{
		jobCounter:       0,
		jobCounterByRepo: make(map[string]uint64),
		jobs:             make(map[uint64]*Job),
	}
}

var (
	s JobStore = NewInMemoryJobStore()
)

func (s *InMemoryJobStore) GetByNumber(owner, repo string, number uint64) (*Job, error) {
	jobs, err := s.List(owner, repo)
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

func (s *InMemoryJobStore) ListOwners() ([]string, error) {
	m := make(map[string]bool)

	for _, j := range s.jobs {
		m[j.Owner] = true
	}

	l := make([]string, 0, len(m))
	for k, _ := range m {
		l = append(l, k)
	}

	return l, nil
}

func (s *InMemoryJobStore) ListRepos(owner string) ([]string, error) {
	m := make(map[string]bool)

	for _, j := range s.jobs {
		if j.Owner == owner {
			m[j.Repo] = true
		}
	}

	l := make([]string, 0, len(m))
	for k, _ := range m {
		l = append(l, k)
	}

	return l, nil
}

func (s *InMemoryJobStore) List(owner, repo string) ([]*Job, error) {
	var l []*Job

	for _, j := range s.jobs {
		if j.Owner == owner && j.Repo == repo {
			l = append(l, j)
		}
	}

	return l, nil
}

func (s *InMemoryJobStore) Save(j *Job) error {
	if j.Number != 0 {
		// the job is in memory, nothing to persist
		return nil
	}

	// obviously not thread-safe
	s.jobCounter++
	id := s.jobCounter

	k := j.Owner + ":" + j.Repo
	s.jobCounterByRepo[k]++
	number := s.jobCounterByRepo[k]

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

func (wl WriterLogger) WriteTo(w io.Writer) (int64, error) {
	// doesn't support reading
	return 0, nil
}

func (wl WriterLogger) WriteStep(name string) error {
	s := fmt.Sprintln("CION:", name)
	_, err := wl.Write([]byte(s))
	return err
}
