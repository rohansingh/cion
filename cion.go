package cion

import (
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
	"log"
	"net/http"
	"regexp"
)

var (
	e  Executor
	js JobStore
)

func Run(dockerEndpoint, dockerCertPath string) {
	var err error

	e, err = NewDockerExecutor(dockerEndpoint, dockerCertPath)
	if err != nil {
		log.Fatalf("error initializing executor: %v", err)
	}

	js = &InMemoryJobStore{}

	repo := web.New()
	goji.Handle("/:owner/:repo/*", repo)

	repo.Use(middleware.SubRouter)

	repo.Post("/new", NewJobHandler)
	repo.Post(regexp.MustCompile("^/branch/(?P<branch>.+)/new"), NewJobHandler)
	repo.Post(regexp.MustCompile("^/commit/(?P<sha>.+)/new"), NewJobHandler)

	goji.Serve()
}

func NewJobHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]
	branch := c.URLParams["branch"]
	sha := c.URLParams["sha"]

	j := NewJob(owner, repo, branch, sha)
	jr := &JobRequest{
		Job:      &j,
		Executor: e,
		Store:    js,
	}

	go jr.Run()
}
