package cion

import (
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
	"log"
	"regexp"
)

var (
	e  Executor
	js JobStore
)

func Run(dockerEndpoint, dockerCertPath, cionDbPath string) {
	var err error

	e, err = NewDockerExecutor(dockerEndpoint, dockerCertPath)
	if err != nil {
		log.Fatalf("error initializing executor: %v", err)
	}

	js, err = NewBoltJobStore(cionDbPath)
	if err != nil {
		log.Fatalf("error initializing job store: %v", err)
	}

	api := web.New()
	api.Use(middleware.SubRouter)
	goji.Handle("/api/*", api)

	api.Get("/", ListOwnersHandler)
	api.Get("/:owner", ListReposHandler)
	api.Get("/:owner/:repo", ListBranchesHandler)

	repo := web.New()
	repo.Use(middleware.SubRouter)

	api.Handle("/:owner/:repo/*", repo)
	repo.Post("/new", NewJobHandler)
	repo.Post(regexp.MustCompile("^/commit/(?P<sha>.+)/new"), NewJobHandler)
	repo.Post(regexp.MustCompile("^/branch/(?P<branch>.+)/new"), NewJobHandler)
	repo.Get(regexp.MustCompile("^/branch/(?P<branch>.+)/(?P<number>[0-9]+)/log"), GetLogHandler)
	repo.Get(regexp.MustCompile("^/branch/(?P<branch>.+)/(?P<number>[0-9]+)"), GetJobHandler)
	repo.Get(regexp.MustCompile("^/branch/(?P<branch>.+)"), ListJobsHandler)

	goji.Serve()
}
