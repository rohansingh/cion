package cion

import (
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
	"log"
	"net/http"
	"regexp"
)

var (
	e   Executor
	js  JobStore
	ghc string
	ghs string
)

func Run(dockerEndpoint, dockerCertPath, cionDbPath, ghClientID, ghSecret string) {
	var err error

	e, err = NewDockerExecutor(dockerEndpoint, dockerCertPath)
	if err != nil {
		log.Fatalf("error initializing executor: %v", err)
	}

	js, err = NewBoltJobStore(cionDbPath)
	if err != nil {
		log.Fatalf("error initializing job store: %v", err)
	}

	ghc = ghClientID
	ghs = ghSecret

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

	goji.Get("/*", http.FileServer(http.Dir("./public")))

	serve()
}

func serve() {
	goji.DefaultMux.Compile()
	http.Handle("/", goji.DefaultMux)

	b := bind.Sniff()
	if b == "" {
		b = ":8000"
	}

	l := bind.Socket(b)
	log.Println("Listening on", l.Addr())

	graceful.HandleSignals()
	if err := graceful.Serve(l, http.DefaultServeMux); err != nil {
		log.Fatal(err)
	}
	graceful.Wait()
}
