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

	repo := web.New()
	repo.Use(middleware.SubRouter)

	goji.Handle("/:owner/:repo/*", repo)
	repo.Post("/new", NewJobHandler)
	repo.Post(regexp.MustCompile("^/commit/(?P<sha>.+)/new"), NewJobHandler)
	repo.Post(regexp.MustCompile("^/branch/(?P<branch>.+)/new"), NewJobHandler)
	repo.Get(regexp.MustCompile("^/branch/(?P<branch>.+)/(?P<number>[0-9]+)/log"), GetLogHandler)
	repo.Get(regexp.MustCompile("^/branch/(?P<branch>.+)/(?P<number>[0-9]+)"), GetJobHandler)
	repo.Get(regexp.MustCompile("^/branch/(?P<branch>.+)"), ListJobsHandler)

	goji.Serve()
}

func NewJobHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]
	branch := c.URLParams["branch"]
	sha := c.URLParams["sha"]

	j := NewJob(owner, repo, branch, sha)
	if err := js.Save(j); err != nil {
		log.Println("error saving job:", err)
	}

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

	j, err := js.GetByNumber(owner, repo, branch, number)
	if err != nil {
		log.Println("error getting job:", err)
	}

	b, _ := json.MarshalIndent(j, "", "\t")
	w.Write(b)
}

func GetLogHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]
	branch := c.URLParams["branch"]
	number, _ := strconv.ParseUint(c.URLParams["number"], 0, 64)

	j, err := js.GetByNumber(owner, repo, branch, number)
	if err != nil {
		log.Println("error getting job:", err)
	}

	if _, err := js.GetLogger(j).WriteTo(w); err != nil {
		log.Println("error getting job logs:", err)
	}
}

func ListJobsHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]
	branch := c.URLParams["branch"]

	l, err := js.List(owner, repo, branch)
	if err != nil {
		log.Println("error getting job list:", err)
	}

	b, _ := json.MarshalIndent(l, "", "\t")
	w.Write(b)
}
