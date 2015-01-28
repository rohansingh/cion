package cion

import (
	"encoding/json"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
	"log"
	"net/http"
	"regexp"
	"strconv"
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

	js = NewInMemoryJobStore()

	repo := web.New()
	repo.Use(middleware.SubRouter)

	goji.Handle("/:owner/:repo/*", repo)
	repo.Post("/new", NewJobHandler)
	repo.Post(regexp.MustCompile("^/commit/(?P<sha>.+)/new"), NewJobHandler)
	repo.Post(regexp.MustCompile("^/branch/(?P<branch>.+)/new"), NewJobHandler)
	repo.Get(regexp.MustCompile("^/branch/(?P<branch>.+)/(?P<number>[0-9]+)"), GetJobHandler)

	goji.Serve()
}

func NewJobHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]
	branch := c.URLParams["branch"]
	sha := c.URLParams["sha"]

	j := NewJob(owner, repo, branch, sha)
	js.Save(j)

	jr := &JobRequest{
		Job:      j,
		Executor: e,
		Store:    js,
	}

	go jr.Run()

	b, _ := json.MarshalIndent(j, "", "\t")
	w.Write(b)
}

func GetJobHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]
	branch := c.URLParams["branch"]
	number, _ := strconv.ParseUint(c.URLParams["number"], 0, 64)

	j, _ := js.GetByNumber(owner, repo, branch, number)

	b, _ := json.MarshalIndent(j, "", "\t")
	w.Write(b)
}
