package cion

import "io"

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
