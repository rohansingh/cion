FROM ubuntu:14.04
MAINTAINER Rohan Singh <rohan@washington.edu> (@rohansingh)

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y apt-transport-https ca-certificates iptables

RUN echo deb https://get.docker.io/ubuntu docker main > /etc/apt/sources.list.d/docker.list \
  && apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 36A1D7869245C8950F966E92D8576A8BA88D21E9 \
  && apt-get update \
  && apt-get install -y lxc-docker

ADD dind /dind

VOLUME /var/lib/docker
ENTRYPOINT ["/dind"]
