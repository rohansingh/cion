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

	j := NewJob(owner, repo, branch, "")
	if err := js.Save(j); err != nil {
		log.Println("error saving job:", err)
	}

	jr := &JobRequest{
		Job:      j,
		Executor: e,
		Store:    js,
	}

	go jr.Run()

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.MarshalIndent(j, "", "\t")
	w.Write(b)
}

func GetJobHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]
	number, _ := strconv.ParseUint(c.URLParams["number"], 0, 64)

	j, err := js.GetByNumber(owner, repo, number)
	if err != nil {
		log.Println("error getting job:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.MarshalIndent(j, "", "\t")
	w.Write(b)
}

func GetLogHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]
	number, _ := strconv.ParseUint(c.URLParams["number"], 0, 64)

	j, err := js.GetByNumber(owner, repo, number)
	if err != nil {
		log.Println("error getting job:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := js.GetLogger(j).WriteTo(w); err != nil {
		log.Println("error getting job logs:", err)
	}
}

func ListJobsHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	owner := c.URLParams["owner"]
	repo := c.URLParams["repo"]

	l, err := js.List(owner, repo)
	if err != nil {
		log.Println("error getting job list:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.MarshalIndent(l, "", "\t")
	w.Write(b)
}

func ListOwnersHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	l, err := js.ListOwners()
	if err != nil {
		log.Println("error getting owners list:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.MarshalIndent(l, "", "\t")
	w.Write(b)
}

func ListReposHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	l, err := js.ListRepos(c.URLParams["owner"])
	if err != nil {
		log.Println("error getting repos list:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.MarshalIndent(l, "", "\t")
	w.Write(b)
}
