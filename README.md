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

2. Run the build container. Perhaps in the future, build and tests will be separated and we will also run a separate test container.

3. Run the deployment container.

We have a single system that tracks the output and state of each of these steps. It becomes the reference source for which builds were successful and what was deployed where.

Configuration Spec
---

All configuration is specified in `.cion.yml` in the project. Here's an example:

```yaml
build:
  - image: rohan/my-build-image
  - env: # additional environment variables for the build container
      key1: val1
      key2: val2

release:
  - image: rohan/my-release-image
  - env: # additional environment variables for the release container
      key3: val3
      key4: val4
```

Job Container
---

For each commit, we generate a single job. All work for the commit is done inside this single Docker container.

The job container runs a Docker-in-Docker container, and each build step is still executed in its own container. However, the working directory can live in the main container and be bind-mounted into each subcontainer.

### Docker-in-Docker details

This is roughly how we run the job container:

```bash
$ docker run -v /var/run/docker.sock:/var/run/docker.sock
```

There are two things to note here:

1. The job container needs access to the host's Docker endpoint. It uses this to build and launch a Docker-in-Docker (DinD) container. The endpoint for the DinD instance is passed to the build container, to allow it to safely build and push Docker images.

2. The job container also uses the host's Docker endpoint to launch the build and release containers.

Note that the build and release containers are *not* run using Docker-in-Docker, they are first-class containers on the host machine. However, they are not privileged containers, and the Docker instance used by the build container only exists for the lifetime of the build.

Build Container
---

Environment variables that are passed to the build container:

* `BUILD_DIR`<br />
  The path to the working directory that contains the current commit.

* `ARTIFACTS_DIR`<br />
  The path to the artifacts directory.

* `DOCKER_HOST`<br />
  Endpoint for a Docker instance that the build container can use to build or push Docker images.

* Additional environment variables from the user's config.

The expectation is that the build container will build the project and place generated artifacts in the `ARTIFACTS_DIR`.

Release Container
---

Environment variables that are passed to the release container:

* `ARTIFACTS_DIR`<br />
  The path to the artifacts directory.

* Additional environment variables from the user's config.

The expectation is that the release container will release the project and write the status to stdout/stderr.
