cion
===
A commit-to-deploy system based on Docker containers.


Build
---
Install Go and Node, set your `GOPATH`, and then run:

    $ go get github.com/rohansingh/cion
    $ cd $GOPATH/src/github.com/rohansingh/cion/public
    $ npm install -g gulp
    $ gulp

This will download cion to `$GOPATH/src`, build it, and install the executable to `$GOPATH/bin`.

The final steps are necessary to compile the resource bundle for the UI. This should be automated
at some point.

Run
---
Currently cion will only run from its source directory:

    $ cd $GOPATH/src/github.com/rohansingh/cion
    $ go install ./... && cion

Note that only one instance can run at a time. Any new instances will just wait around trying to
acquire a lock on the database file.

API Examples
---

    # kick off a build of the rohan/cion branch of https://github.com/spotify/docker-client
    curl -X POST http://localhost:8000/api/spotify/docker-client/branch/rohan/cion/new

    # kick off a build of the master branch of cion
    curl -X POST http://localhost:8000/api/rohansingh/cion/new

    # get a list of all jobs for spotify/docker-client
    curl -X GET http://localhost:8000/api/spotify/docker-client

    # get a particular job by number
    curl -X GET http://localhost:8000/api/spotify/docker-client/1

    # get the logs for a particular job by number
    curl -X GET http://localhost:8000/api/spotify/docker-client/1/log

User Guide
===

Build Steps
---
These are the logical steps needed to go from commit to deploy:

1. Pull down repo and parse configuration. Based on configuration, choose which images will be used for each of the following steps.

2. Run each service container. These containers provide services that are necessary for the build (for example, Docker).

2. Run the build container. Perhaps in the future, build and tests will be separated and we will also run a separate test container.

3. Run the deployment container.

We have a single system that tracks the output and state of each of these steps. It becomes the reference source for which builds were successful and what was deployed where.

Configuration Spec
---

All configuration is specified in `.cion.yml` in the project. Here's an example:

```yaml
build:
  image: rohan/my-build-image

release:
  image: rohan/my-release-image
  cmd: some-optional-command

services:
  docker:
    image: jpetazzo/dind
    privileged: true
    env: # any environment variables for the service container
      - PORT=2375
    ports: # list af ports to expose from the service container
      - 2375/tcp

  some_service:
    image: rohan/some-other-image
```

The specified build and release containers are run for the build and release steps of the build, respectively.

Job Runner
---

For each commit, we generate a single job. The job runner pulls the project from Git, parses the `.cion.yml` configuration, and runs the service, build, and release containers.

### Working directory container

Prior to running the service, build, and release containers, the job runner actually launches a data container to contain the working directory for the build. This container is also linked to the build and release containers.

Service Containers
---

Each service container is spawned at the beginning of the build, and is linked into the build and release containers via [Docker links](https://docs.docker.com/userguide/dockerlinks/).

For example, this is roughly how the build container is run based on the sample configuration above:

 ```bash
 $ docker run --link docker:docker --link some_service:some_service rohan/my-build-image
 ```

Build and test containers can use these environment variables to determine how to connect to the service containers.

Build Container
---

Environment variables that are passed to the build container:

* `BUILD_DIR`<br />
  The path to the working directory that contains the current commit.

* `ARTIFACTS_DIR`<br />
  The path to the artifacts directory.

* Additional environment variables from the user's config.

* Environment variables generated by [Docker links](https://docs.docker.com/userguide/dockerlinks/) for any service containers.

The expectation is that the build container will build the project and place generated artifacts in the `ARTIFACTS_DIR`.

Release Container
---

Environment variables that are passed to the release container:

* `ARTIFACTS_DIR`<br />
  The path to the artifacts directory.

* Additional environment variables from the user's config.

* Environment variables generated by [Docker links](https://docs.docker.com/userguide/dockerlinks/) for any service containers.

The expectation is that the release container will release the project and write the status to stdout/stderr.
