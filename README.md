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

Container Specs
---

### Job container

For each commit, we generate a single job. All work for the commit is done inside this single Docker container.

The job container runs Docker-in-Docker, so each build step is still executed in its own container. However, the working directory can live in the main container and be bind-mounted into each subcontainer.

### Build container

Environment variables that are passed to the build container:

* `BUILD_DIR`<br />
  The path to the working directory that contains the current commit.

* `ARTIFACTS_DIR`<br />
  The path to the artifacts directory.

* Additional environment variables from the user's config.

The expectation is that the build container will build the project and place generated artifacts in the `ARTIFACTS_DIR`.

### Release container

Environment variables that are passed to the release container:

* `ARTIFACTS_DIR`<br />
  The path to the artifacts directory.

* Additional environment variables from the user's config.

The expectation is that the release container will release the project and write the status to stdout/stderr.
