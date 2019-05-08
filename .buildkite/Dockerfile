FROM golang:1.12.5-stretch
ARG JFROG_UNAME
ARG JFROG_PWORD

ENV GOPATH=/Source

RUN mkdir -p /Source/src/github.com/hashicorp/nomad && \
apt-get update && \
apt-get install build-essential

COPY . /Source/src/github.com/hashicorp/nomad

WORKDIR /Source/src/github.com/hashicorp/nomad
RUN make bootstrap && \
make prerelease && \
make release 

WORKDIR /Source/src/github.com/hashicorp/nomad/pkg

RUN ls