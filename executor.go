package cion

import "io"

// An Executor runs Docker containers against a Docker host or Docker-like cluster.
type Executor interface {
	// Run starts a Docker container and returns the container name if successful.
	Run(opts RunContainerOpts) (string, error)

	// Attach attaches to a container and writes stdout/stderr to the provided writers.
	Attach(id string, stdout io.Writer, stderr io.Writer) error

	// Wait blocks until a container exits, and returns its exit code.
	Wait(id string) (int, error)

	// Kill kills a container dead.
	Kill(id string) error

	// Build builds a Docker image and returns the image name if successful.
	Build(input io.Reader, output io.Writer) (string, error)
}

// RunContainerOpts are options for running a new container.
type RunContainerOpts struct {
	// Image is the name of the Docker image to run.
	Image string

	// Cmd is the command to run in the container.
	Cmd []string

	// Env is a list of environment variables for the container, in the form "KEY=value".
	Env []string

	// Volumes is a list of volumes to create in the container.
	Volumes []string

	// Links is a list of existing containers that should be linked into the new container,
	// in the form "container_nname:alias".
	Links []string

	// VolumesFrom is a list of existing containers whose volumes should be mounted in the new
	// container, in the form "container_name[:ro|:rw]".
	VolumesFrom []string

	// Privileged specifies whether the container has extended privileges.
	Privileged bool

	// WorkingDir is the working directory in the container.
	WorkingDir string

	// Ports is a list of container ports to expose, in the format <port>/<tcp|udp>.
	Ports []string

	// LocalImage specifies whether the image was built locally (so we shouldn't try to pull it
	// from a remote repo).
	LocalImage bool
}
