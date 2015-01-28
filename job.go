package cion

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/google/go-github/github"
	"gopkg.in/yaml.v2"
	"io"
	"time"
)

const (
	// GitImage is the Docker image used for the working directory and fetching sources.
	GitImage = "radial/busyboxplus:git"
)

// JobRequest defines a job that needs to be run and the dependencies needed to run it.
type JobRequest struct {
	Job      *Job
	Executor Executor
	Store    JobStore
}

// Job represents the job data that should be persisted to a JobStore.
type Job struct {
	ID     uint64
	Number uint64

	Owner  string
	Repo   string
	Branch string
	SHA    string

	StartedAt *time.Time
	EndedAt   *time.Time

	Success bool
}

// JobConfig is the job configuration defined by .cion.yml.
type JobConfig struct {
	Build    ContainerConfig
	Release  ContainerConfig
	Services map[string]ContainerConfig
}

// ContainerConfig is a container configuration defined in .cion.yml.
type ContainerConfig struct {
	Image      string
	Cmd        []string
	Env        []string
	Privileged bool
}

func NewJob(owner, repo, branch, sha string) *Job {
	if branch == "" && sha == "" {
		branch = "master"
	}

	t := time.Now()
	return &Job{
		Owner:     owner,
		Repo:      repo,
		Branch:    branch,
		SHA:       sha,
		StartedAt: &t,
	}
}

// Run executes a JobRequest and logs the results to the JobStore.
func (r JobRequest) Run() {
	jl := r.Store.GetLogger(r.Job)

	if err := runJob(r.Job, r.Executor, r.Store, jl); err != nil {
		io.WriteString(jl, fmt.Sprintf("ERROR: %v", err))
	}
}

func runJob(j *Job, e Executor, s JobStore, jl JobLogger) error {
	jl.WriteStep("fetch sources")

	if j.SHA == "" {
		// figure out the latest commit sha for the branch
		gh := github.NewClient(nil)

		com, _, err := gh.Repositories.GetCommit(j.Owner, j.Repo, j.Branch)
		if err != nil {
			return err
		}

		j.SHA = *com.SHA
		s.Save(j)
	}

	wd, err := startWorkdirContainer(j.Owner, j.Repo, j.SHA, e, jl)
	if err != nil {
		return err
	}

	jl.WriteStep("parse job config")
	jc, err := parseJobConfig(wd, e, jl)
	if err != nil {
		return err
	}

	jl.WriteStep("start services")
	services, err := startServices(*jc, wd, e)
	for _, sc := range services {
		// ensure any started services are shut down when we're done
		defer e.Kill(sc)
	}
	if err != nil {
		return err
	}

	jl.WriteStep("build")
	if err := run(jc.Build, services, wd, e, jl); err != nil {
		return err
	}

	if jc.Release.Image != "" {
		jl.WriteStep("release")
		return run(jc.Release, nil, wd, e, jl)
	} else {
		return nil
	}
}

func startWorkdirContainer(owner, repo, sha string, e Executor, lw io.Writer) (string, error) {
	gh := github.NewClient(nil)
	r, _, err := gh.Repositories.Get(owner, repo)
	if err != nil {
		return "", err
	}

	// command to fetch sources and then read .cion.yml to stderr
	fetchCmd := []string{
		"sh", "-c",
		`git clone "$CLONE_URL" /cion/build && \
			cd /cion/build && \
			git checkout "$REFSPEC"`,
	}

	opts := RunContainerOpts{
		Image:   GitImage,
		Cmd:     fetchCmd,
		Volumes: []string{"/cion/build", "/cion/artifacts"},
		Env: []string{
			"CLONE_URL=" + *r.CloneURL,
			"REFSPEC=" + sha,
		},
	}

	wd, err := e.Run(opts)
	if err != nil {
		return "", err
	}

	// wait for the container to finish fetching sources
	err = e.Attach(wd, lw, lw)
	if err != nil {
		return "", err
	}

	return wd, nil
}

func parseJobConfig(wd string, e Executor, lw io.Writer) (*JobConfig, error) {
	opts := RunContainerOpts{
		Image:       GitImage,
		Cmd:         []string{"cat", "/cion/build/.cion.yml"},
		VolumesFrom: []string{wd},
	}

	c, err := e.Run(opts)
	if err != nil {
		return nil, err
	} else {
		defer e.Kill(c)
	}

	var stdout bytes.Buffer
	err = e.Attach(c, io.MultiWriter(&stdout, lw), lw)
	if err != nil {
		return nil, err
	}

	// if things worked out, the .cion.yml should have been read into stdout
	jc := &JobConfig{}
	if err := yaml.Unmarshal(stdout.Bytes(), jc); err != nil {
		return nil, err
	}

	if jc.Build.Image == "" {
		// this is really the only thing that's required in the JobConfig
		return nil, errors.New("no build image specified")
	}

	return jc, nil
}

func startServices(jc JobConfig, wd string, e Executor) (map[string]string, error) {
	started := make(map[string]string, len(jc.Services))

	for s, cc := range jc.Services {
		opts := RunContainerOpts{
			Image: cc.Image,
			Cmd:   cc.Cmd,
			Env:   cc.Env,
		}

		c, err := e.Run(opts)
		if err != nil {
			return started, err
		}

		started[s] = c
	}

	return started, nil
}

func run(cc ContainerConfig, services map[string]string, wd string,
	e Executor, lw io.Writer) error {
	links := make([]string, len(services))
	for s, sc := range services {
		links = append(links, s+":"+sc)
	}

	env := make([]string, 0, len(cc.Env)+2)
	copy(env, cc.Env)

	env = append(env, "BUILD_DIR=/cion/build")
	env = append(env, "ARTIFACTS_DIR=/cion/artifacts")

	opts := RunContainerOpts{
		Image:       cc.Image,
		Cmd:         cc.Cmd,
		Env:         env,
		Privileged:  cc.Privileged,
		Links:       links,
		VolumesFrom: []string{wd},
		WorkingDir:  "/cion/build",
	}

	c, err := e.Run(opts)
	if err != nil {
		return err
	}

	return e.Attach(c, lw, lw)
}
