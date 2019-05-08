FROM golang:1.12.5-stretch
ARG JFROG_UNAME
ARG JFROG_PWORD

RUN mkdir -p /go/src/github.com/hashicorp/nomad && \
apt-get update && \
apt-get install -y build-essential

COPY . /go/src/github.com/hashicorp/nomad

WORKDIR /go/src/github.com/hashicorp/nomad
RUN make bootstrap && \
make prerelease && \
make release 

WORKDIR /go/src/github.com/hashicorp/nomad/pkg

RUN ls