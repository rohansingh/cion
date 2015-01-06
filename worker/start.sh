#!/bin/bash
set -e

DIND_IMAGE=$(cat /dev/urandom | tr -cd 'a-f0-9' | head -c 32)
DIND_NAME=$(cat /dev/urandom | tr -cd 'a-f0-9' | head -c 32)
DIND_ALIAS=dind

docker build -q -t $DIND_IMAGE dind

DIND_ID=$(docker run --name $DIND_NAME --privileged -d -e PORT=2375 -p 2375 $DIND_IMAGE)
trap "docker kill $DIND" SIGINT SIGTERM

DIND_PORT=$(docker port $DIND_ID 2375)
DIND_DOCKER_HOST=tcp://$DIND_ALIAS:$DIND_PORT

# TODO: parse configuration and run the build/deploy containers
