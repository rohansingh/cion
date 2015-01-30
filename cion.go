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
	api.Get("/:owner/:repo", ListJobsHandler)

	repo := web.New()
	repo.Use(middleware.SubRouter)

	api.Handle("/:owner/:repo/*", repo)
	repo.Post("/new", NewJobHandler)
	repo.Post(regexp.MustCompile("^/branch/(?P<branch>.+)/new"), NewJobHandler)
	repo.Get("/:number/log", GetLogHandler)
	repo.Get("/:number", GetJobHandler)
	repo.Get("/", ListJobsHandler)

	goji.Serve()
}
