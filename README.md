cion
===
A commit-to-deploy system based on Docker containers.

Job Queue
---
If we can abstract all work into a Docker container, then the whole commit-to-deploy system can be represented by a single queue of containers that need to be run.

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
  - image: rohan/my-build-image
  - env: # extra environment variables for the build container
      key1: val1
      key2: val2

release:
  - image: rohan/my-release-image
  - cmd: some-optional-command
  - env: # extra environment variables for the release container
      key3: val3
      key4: val4

services:
  docker:
  - image: tianon/dind
  - privileged: true

  some_service:
  - image: rohan/some-other-image
  - env: # any environment variables for the service container
      key6: val6
```

The specified build and release containers are run for the build and release steps of the build, respectively.

Job Runner
---

For each commit, we generate a single job. The job runner pulls the project from Git, parses the `.cion.yml` configuration, and runs the service, build, and release containers.

This is roughly how we execute the job runner:

```bash
$ docker run -v /var/run/docker.sock:/var/run/docker.sock cion-job-runner
```

We do this because the job runner needs access to the host's Docker endpoint to launch the other containers for the build.

### Working directory container

Prior to running the build, and release containers, the job runner actually launches a data container to contain the working directory for the build. This container is also linked to the build and release containers.

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
