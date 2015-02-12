package cion

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/fsouza/go-dockerclient"
	"io"
	"path/filepath"
)

// DockerExecutor is an Executor that runs against a single Docker host.
type DockerExecutor struct {
	client *docker.Client
}

func NewDockerExecutor(endpoint, certPath string) (*DockerExecutor, error) {
	var c *docker.Client
	var err error

	if certPath == "" {
		c, err = docker.NewClient(endpoint)
	} else {
		cert := filepath.Join(certPath, "cert.pem")
		key := filepath.Join(certPath, "key.pem")
		ca := filepath.Join(certPath, "ca.pem")

		c, err = docker.NewTLSClient(endpoint, cert, key, ca)
	}

	if err != nil {
		return nil, err
	}

	return &DockerExecutor{client: c}, nil
}

func (e DockerExecutor) Run(opts RunContainerOpts) (string, error) {
	vols := make(map[string]struct{}, len(opts.Volumes))
	for _, v := range opts.Volumes {
		vols[v] = struct{}{}
	}

	if !opts.LocalImage {
		pio := docker.PullImageOptions{Repository: opts.Image}
		if err := e.client.PullImage(pio, docker.AuthConfiguration{}); err != nil {
			return "", err
		}
	}

	ep := make(map[docker.Port]struct{}, len(opts.Ports))
	for _, p := range opts.Ports {
		ep[docker.Port(p)] = struct{}{}
	}

	cco := docker.CreateContainerOptions{
		Name: uuid.New(),
		Config: &docker.Config{
			Image:        opts.Image,
			Cmd:          opts.Cmd,
			Env:          opts.Env,
			ExposedPorts: ep,
			Volumes:      vols,
			WorkingDir:   opts.WorkingDir,
		},
	}

	hc := docker.HostConfig{
		Links:           opts.Links,
		Privileged:      opts.Privileged,
		VolumesFrom:     opts.VolumesFrom,
		PublishAllPorts: true,
	}

	c, err := e.client.CreateContainer(cco)
	if err != nil {
		return "", err
	}

	if err := e.client.StartContainer(c.ID, &hc); err != nil {
		return "", err
	}

	return c.Name, nil
}

func (e DockerExecutor) Attach(id string, stdout io.Writer, stderr io.Writer) error {
	opts := docker.AttachToContainerOptions{
		Container: id,
		Logs:      true,
		Stream:    true,

		Stdout: (stdout != nil),
		Stderr: (stderr != nil),

		OutputStream: stdout,
		ErrorStream:  stderr,
	}

	return e.client.AttachToContainer(opts)
}

func (e DockerExecutor) Wait(id string) (int, error) {
	return e.client.WaitContainer(id)
}

func (e DockerExecutor) Kill(id string) error {
	return e.client.KillContainer(docker.KillContainerOptions{ID: id})
}

func (e DockerExecutor) Build(input io.Reader, output io.Writer) (string, error) {
	name := uuid.New()
	opts := docker.BuildImageOptions{
		Name:         name,
		InputStream:  input,
		OutputStream: output,
	}

	if err := e.client.BuildImage(opts); err != nil {
		return "", err
	} else {
		return name, nil
	}
}
