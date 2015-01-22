package main

import (
	"crypto/sha1"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/fsouza/go-dockerclient"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

const WorkdirDockerfile string = `FROM scratch
MAINTAINER Rohan Singh <rohan@washington.edu> (@rohansingh)

VOLUME /cion/build
VOLUME /cion/artifacts

COPY ./* /cion/build`

type ContainerConfig struct {
	Image      string
	Command    string `yaml:"cmd"`
	Env        map[string]string
	Privileged bool
}

type JobConfig struct {
	Build    ContainerConfig
	Release  ContainerConfig
	Services map[string]ContainerConfig
}

func main() {
	app := cli.NewApp()
	app.Name = "run_job"
	app.Usage = "Run a cion job"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "name",
			Usage:  "name of the project",
			EnvVar: "CION_PROJECT",
		},
		cli.StringFlag{
			Name:   "repo",
			Usage:  "git repo for the project",
			EnvVar: "CION_REPO",
		},
		cli.StringFlag{
			Name:   "refspec",
			Usage:  "refspec to build",
			EnvVar: "CION_REFSPEC",
		},
		cli.StringFlag{
			Name:   "docker",
			Value:  "unix:///var/run/docker.sock",
			Usage:  "docker host endpoint",
			EnvVar: "DOCKER_HOST",
		},
		cli.StringFlag{
			Name:  "workdir",
			Value: "/cion/workdir",
			Usage: "path to the workdir image",
		},
	}

	app.Action = Run
	app.Run(os.Args)
}

func Run(c *cli.Context) {
	dockerClient, err := docker.NewClient(c.String("docker"))
	if err != nil {
		log.Fatalf("error creating Docker client: %v", err)
	}

	workdir, repo, refspec := c.String("workdir"), c.String("repo"), c.String("refspec")

	if err := FetchSources(workdir, repo, refspec); err != nil {
		log.Fatalf("error fetching sources: %v", err)
	}

	jobConfig, err := ReadJobConfig(workdir)
	if err != nil {
		log.Fatalf("error reading job config: %v", err)
	}

	workdirContainer, err := StartWorkdir(workdir, repo, refspec, dockerClient)
	if err != nil {
		log.Fatalf("error initializing workdir: %v", err)
	} else {
		defer dockerClient.KillContainer(
			docker.KillContainerOptions{ID: workdirContainer.ID},
		)
	}

	_, err = jobConfig.StartServiceContainers()
	if err != nil {
		log.Fatalf("error starting service containers: %v", err)
	}
}

// FetchSources fetches sources for the project from Git and places them into workdir.
func FetchSources(workdir string, repo string, refspec string) error {
	cloneCmd := exec.Command(
		"git",
		"clone",
		"--depth=50",
		repo,
		workdir,
	)
	cloneCmd.Stdout, cloneCmd.Stderr = os.Stdout, os.Stderr

	if err := cloneCmd.Run(); err != nil {
		return err
	}

	fetchCmd := exec.Command(
		"git",
		"fetch",
		"origin",
		refspec,
	)
	fetchCmd.Dir = workdir
	fetchCmd.Stdout, fetchCmd.Stderr = os.Stdout, os.Stderr

	if err := fetchCmd.Run(); err != nil {
		return err
	}

	return nil
}

// StartWorkdir builds and starts a Docker container for the working directory, populated with the
// sources from the workdir. If successful, it returns the running docker.Container.
func StartWorkdir(workdir string, repo string, refspec string,
	dockerClient *docker.Client) (*docker.Container, error) {
	imageName := fmt.Sprintf("%x", sha1.Sum([]byte(repo+" "+refspec)))

	buildOpts := docker.BuildImageOptions{
		Name:        imageName,
		InputStream: strings.NewReader(WorkdirDockerfile),
		ContextDir:  workdir,
	}
	if err := dockerClient.BuildImage(buildOpts); err != nil {
		return nil, err
	}

	createOpts := docker.CreateContainerOptions{
		Config: &docker.Config{Image: imageName},
	}

	container, err := dockerClient.CreateContainer(createOpts)
	if err != nil {
		return nil, err
	}

	hc := docker.HostConfig{NetworkMode: "none"}
	if err := dockerClient.StartContainer(container.ID, &hc); err != nil {
		return nil, err
	}

	return container, nil
}

// ReadJobConfig reads the .cion.yml config in the workdir and returns a JobConfig.
func ReadJobConfig(workdir string) (*JobConfig, error) {
	file, err := ioutil.ReadFile(path.Join(workdir, ".cion.yml"))
	if err != nil {
		return nil, err
	}

	var jc *JobConfig
	if err := yaml.Unmarshal(file, jc); err != nil {
		return nil, err
	}

	return jc, nil
}

func (jc *JobConfig) StartServiceContainers() ([]docker.Container, error) {
	started = make([]docker.Container, 0, len(jc.Services))

	for name, cc := range jc.Services {
		createOpts := docker.CreateContainerOptions{
			Config: &docker.Config{Image: cc.Image},
		}
	}
}
