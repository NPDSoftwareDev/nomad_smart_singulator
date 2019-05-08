FROM golang:1.12.5-stretch
ARG JFROG_UNAME
ARG JFROG_PWORD

ENV GOROOT=/go \
GOPATH=/Source \
PATH=$PATH:$GOROOT/bin:$GOPATH/bin

RUN mkdir -p /Source/src/github.com/hashicorp/nomad && \
apt-get update && \
apt-get install -y build-essential

COPY . /Source/src/github.com/hashicorp/nomad

WORKDIR /Source/src/github.com/hashicorp/nomad
RUN make bootstrap && \
make prerelease && \
make release 

WORKDIR /Source/src/github.com/hashicorp/nomad/pkg

RUN ls