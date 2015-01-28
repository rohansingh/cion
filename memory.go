package cion

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"
)

// InMemoryJobStore is a mock JobStore that only stores jobs in memory and writes logs directly
// to stdout. It is only meant to be used for testing.
type InMemoryJobStore struct {
	jobCounter         *uint64
	jobCounterByBranch map[string]*uint64
	jobsByID           map[uint64]*Job
	jobsByNumber       map[string]map[uint64]*Job
}

func (s *InMemoryJobStore) GetByID(id uint64) (*Job, error) {
	return s.jobsByID[id], nil
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

	j.ID = id
	j.Number = number

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
	_, err := wl.Write([]byte(s))
	return err
}
