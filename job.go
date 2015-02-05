package cion

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/google/go-github/github"
	"gopkg.in/yaml.v2"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	// GitImage is the Docker image used for the working directory and fetching sources.
	GitImage = "radial/busyboxplus:git"

	// BuildDir is the path for project sources in the working directory container.
	BuildDir = "/cion/build"

	// ArtifactsDir is the path to the artifacts directory in the working directory container.
	ArtifactsDir = "/cion/artifacts"
)

// JobRequest defines a job that needs to be run and the dependencies needed to run it.
type JobRequest struct {
	Job      *Job
	Executor Executor
	Store    JobStore

	GitHubClientID string
	GitHubSecret   string
}

// Job represents the job data that should be persisted to a JobStore.
type Job struct {
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
	Ports      []string
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

	var c *http.Client
	if r.GitHubClientID != "" {
		t := &github.UnauthenticatedRateLimitedTransport{
			ClientID:     r.GitHubClientID,
			ClientSecret: r.GitHubSecret,
		}

		c = t.Client()
	}

	gh := github.NewClient(c)

	if err := runJob(r.Job, r.Executor, r.Store, jl, gh); err != nil {
		log.Println("job execution error:", err)
		io.WriteString(jl, fmt.Sprintf("ERROR: %v", err))
	} else {
		r.Job.Success = true
	}

	t := time.Now()
	r.Job.EndedAt = &t

	if err := r.Store.Save(r.Job); err != nil {
		log.Println("error saving completed job:", err)
	}
}

func runJob(j *Job, e Executor, s JobStore, jl JobLogger, gh *github.Client) error {
	jl.WriteStep("fetch sources")

	if j.SHA == "" {
		// figure out the latest commit sha for the branch
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
		return run(jc.Release, services, wd, e, jl)
	} else {
		return nil
	}
}

func startWorkdirContainer(owner, repo, sha string, e Executor, jl io.Writer) (string, error) {
	gh := github.NewClient(nil)
	r, _, err := gh.Repositories.Get(owner, repo)
	if err != nil {
		return "", err
	}

	// command to fetch sources and then read .cion.yml to stderr
	fetchCmd := []string{
		"sh", "-c",
		`git clone "$CLONE_URL" "$BUILD_DIR" && \
			cd "$BUILD_DIR" && \
			git checkout "$REFSPEC"`,
	}

	opts := RunContainerOpts{
		Image: GitImage,
		Cmd:   fetchCmd,
		Volumes: []string{
			BuildDir,
			ArtifactsDir,
		},
		Env: []string{
			"BUILD_DIR=" + BuildDir,
			"CLONE_URL=" + *r.CloneURL,
			"REFSPEC=" + sha,
		},
	}

	wd, err := e.Run(opts)
	if err != nil {
		return "", err
	}

	// wait for the container to finish fetching sources
	err = e.Attach(wd, jl, jl)
	if err != nil {
		return "", err
	}

	if r, err := e.Wait(wd); err != nil {
		return "", err
	} else if r != 0 {
		return "", errors.New("non-zero exit status when fetching sources")
	}

	return wd, nil
}

func parseJobConfig(wd string, e Executor, jl io.Writer) (*JobConfig, error) {
	opts := RunContainerOpts{
		Image:       GitImage,
		Cmd:         []string{"cat", ".cion.yml"},
		VolumesFrom: []string{wd},
		WorkingDir:  BuildDir,
	}

	c, err := e.Run(opts)
	if err != nil {
		return nil, err
	} else {
		defer e.Kill(c)
	}

	var stdout bytes.Buffer
	if err := e.Attach(c, io.MultiWriter(&stdout, jl), jl); err != nil {
		return nil, err
	}

	if r, err := e.Wait(c); err != nil {
		return nil, err
	} else if r != 0 {
		return nil, errors.New("unable to read job config file")
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
			Image:      cc.Image,
			Cmd:        cc.Cmd,
			Env:        cc.Env,
			Ports:      cc.Ports,
			Privileged: cc.Privileged,
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
	e Executor, jl io.Writer) error {
	links := make([]string, 0, len(services))
	for s, sc := range services {
		links = append(links, sc+":"+s)
	}

	env := make([]string, 0, len(cc.Env)+2)
	copy(env, cc.Env)

	env = append(env, "BUILD_DIR="+BuildDir)
	env = append(env, "ARTIFACTS_DIR="+ArtifactsDir)

	opts := RunContainerOpts{
		Image:       cc.Image,
		Cmd:         cc.Cmd,
		Ports:       cc.Ports,
		Privileged:  cc.Privileged,
		Env:         env,
		Links:       links,
		VolumesFrom: []string{wd},
		WorkingDir:  BuildDir,
	}

	c, err := e.Run(opts)
	if err != nil {
		return err
	}

	if err := e.Attach(c, jl, jl); err != nil {
		return err
	}

	if r, err := e.Wait(c); r != 0 {
		return errors.New("non-zero exit status from container")
	} else {
		return err
	}
}
