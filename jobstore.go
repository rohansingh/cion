package cion

import "io"

// JobStore write and reads jobs and job logs from persistent storage.
type JobStore interface {
	// GetByNumber gets a job for the given owner/repo/branch by its number.
	GetByNumber(owner, repo string, number uint64) (*Job, error)

	// ListOwners returns a list of all the repo owners that have jobs.
	ListOwners() ([]string, error)

	// ListRepos returns a list of all the repos for a given owner.
	ListRepos(owner string) ([]string, error)

	// List gets all the jobs for the given owner/repo.
	List(owner, repo string) ([]*Job, error)

	// Save persists a job to storage. If the job doesn't have a number yet, it is assigned the
	// next incrementing job number for the repo.
	Save(j *Job) error

	// GetLogger gets the JobLogger to write logs for a job.
	GetLogger(j *Job) JobLogger
}

// JobLogger provides an io.Writer interface for writing build logs for a job.
type JobLogger interface {
	io.Writer
	io.WriterTo

	// WriteStep writes a transition to a new build step to the log. All subsequent writes are
	// assumed to be part of the new build step, until another new step is written.
	WriteStep(name string) error
}
