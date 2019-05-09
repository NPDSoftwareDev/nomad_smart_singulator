FROM golang:1.12.5-stretch
ARG JFROG_UNAME
ARG JFROG_PWORD

ENV PROTOC_ZIP=protoc-3.7.1-linux-x86_64.zip 

RUN mkdir -p /go/src/github.com/hashicorp/nomad && \
apt-get update && \
apt-get install -y build-essential curl unzip software-properties-common&& \
curl -sL https://deb.nodesource.com/setup_10.x | bash - && \
apt-get install -y nodejs npm && \
npm install -g ember-cli yarn

COPY . /go/src/github.com/hashicorp/nomad

WORKDIR /go/src/github.com/hashicorp/nomad
RUN go get -u github.com/golang/protobuf/protoc-gen-go && \
go get -u github.com/golang/protobuf/proto && \
make bootstrap && \
curl -OL https://github.com/google/protobuf/releases/download/v3.7.1/$PROTOC_ZIP && \
unzip -o $PROTOC_ZIP -d protoc3 && \
mv protoc3/bin/* /usr/local/bin/ && \
mv protoc3/include/* /usr/local/include/ && \
rm -f $PROTOC_ZIP

RUN make prerelease; exit 0

RUN make prerelease && \
make release 

WORKDIR /go/src/github.com/hashicorp/nomad/pkg

RUN ls
