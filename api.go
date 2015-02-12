package cion

import (
	"encoding/json"
	"github.com/zenazn/goji/web"
	"log"
	"net/http"
	"strconv"
)

func NewJobHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	config := c.Env["config"].(Config)

	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]
	branch := c.URLParams["branch"]

	j := NewJob(owner, repo, branch, "")
	if err := config.JobStore.Save(j); err != nil {
		log.Println("error saving job:", err)
	}

	jr := &JobRequest{
		Job:            j,
		Executor:       config.Executor,
		Store:          config.JobStore,
		GitHubClientID: config.GitHubClientID,
		GitHubSecret:   config.GitHubSecret,
	}

	go jr.Run()

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.MarshalIndent(j, "", "\t")
	w.Write(b)
}

func GetJobHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	config := c.Env["config"].(Config)

	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]
	number, _ := strconv.ParseUint(c.URLParams["number"], 0, 64)

	j, err := config.JobStore.GetByNumber(owner, repo, number)
	if err != nil {
		log.Println("error getting job:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.MarshalIndent(j, "", "\t")
	w.Write(b)
}

func GetLogHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	config := c.Env["config"].(Config)

	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]
	number, _ := strconv.ParseUint(c.URLParams["number"], 0, 64)

	j, err := config.JobStore.GetByNumber(owner, repo, number)
	if err != nil {
		log.Println("error getting job:", err)
	}

	w.Header().Set("Content-Type", "text/plain")
	if _, err := config.JobStore.GetLogger(j).WriteTo(w); err != nil {
		log.Println("error getting job logs:", err)
	}
}

func ListJobsHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	config := c.Env["config"].(Config)

	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]

	l, err := config.JobStore.List(owner, repo)
	if err != nil {
		log.Println("error getting job list:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.MarshalIndent(l, "", "\t")
	w.Write(b)
}

func ListOwnersHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	config := c.Env["config"].(Config)
	log.Println("hello")
	log.Println(config)

	l, err := config.JobStore.ListOwners()
	if err != nil {
		log.Println("error getting owners list:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.MarshalIndent(l, "", "\t")
	w.Write(b)
}

func ListReposHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	config := c.Env["config"].(Config)

	l, err := config.JobStore.ListRepos(c.URLParams["owner"])
	if err != nil {
		log.Println("error getting repos list:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.MarshalIndent(l, "", "\t")
	w.Write(b)
}
