package cion

import (
	"encoding/json"
	"github.com/zenazn/goji/web"
	"log"
	"net/http"
	"strconv"
)

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

func ListOwnersHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	l, err := js.ListOwners()
	if err != nil {
		log.Println("error getting owners list:", err)
	}

	b, _ := json.MarshalIndent(l, "", "\t")
	w.Write(b)
}

func ListReposHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	l, err := js.ListRepos(c.URLParams["owner"])
	if err != nil {
		log.Println("error getting repos list:", err)
	}

	b, _ := json.MarshalIndent(l, "", "\t")
	w.Write(b)
}

func ListBranchesHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	l, err := js.ListBranches(c.URLParams["owner"], c.URLParams["repo"])
	if err != nil {
		log.Println("error getting branch list:", err)
	}

	b, _ := json.MarshalIndent(l, "", "\t")
	w.Write(b)
}
