package cion

import (
	"fmt"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
	"log"
	"net/http"
	"regexp"
)

type Config struct {
	Executor       Executor
	JobStore       JobStore
	GitHubClientID string
	GitHubSecret   string
}

func Configure(dockerEndpoint, dockerCertPath, cionDbPath, ghClientID, ghSecret string) Config {
	var err error
	c := Config{}

	c.Executor, err = NewDockerExecutor(dockerEndpoint, dockerCertPath)
	if err != nil {
		log.Fatalf("error initializing executor: %v", err)
	}

	if cionDbPath == "" {
		c.JobStore = NewInMemoryJobStore()
	} else {
		c.JobStore, err = NewBoltJobStore(cionDbPath)
		if err != nil {
			log.Fatalf("error initializing job store: %v", err)
		}
	}

	c.GitHubClientID = ghClientID
	c.GitHubSecret = ghSecret

	return c
}

func ConfigureLocal(dockerEndpoint, dockerCertPath, ghClientID, ghSecret string) Config {
	return Configure(dockerEndpoint, dockerCertPath, "", ghClientID, ghSecret)
}

func RunLocal(path string, conf Config) {
	j := &Job{LocalPath: path}

	jr := &JobRequest{
		Job:            j,
		Executor:       conf.Executor,
		Store:          conf.JobStore,
		GitHubClientID: conf.GitHubClientID,
		GitHubSecret:   conf.GitHubSecret,
	}

	jr.Run()

	fmt.Println("---")
	if j.Success {
		fmt.Println("CION: job succeeded")
	} else {
		fmt.Println("CION: job failed")
	}
}

func Run(conf Config) {
	goji.Use(middleware.EnvInit)
	goji.Use(func(c *web.C, h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Env["config"] = conf
			h.ServeHTTP(w, r)
		})
	})

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
