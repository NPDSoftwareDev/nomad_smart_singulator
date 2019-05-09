FROM golang:1.12.5-stretch
ARG JFROG_UNAME
ARG JFROG_PWORD

ENV PROTOC_ZIP=protoc-3.7.1-linux-x86_64.zip 

RUN mkdir -p /go/src/github.com/hashicorp/nomad && \
apt-get update && \
apt-get install -y curl unzip zip && \
curl -sL https://deb.nodesource.com/setup_8.x | bash - && \
apt-get install -y nodejs npm && \
npm install -g --unsafe-perm ember-cli yarn node-sass node-gyp

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

#Install build utilities
RUN apt-get install -y gcc \
g++ \
software-properties-common \
build-essential \
libc6-dev

#Install i386 dependencies
RUN dpkg --add-architecture i386 && \
apt-get install -y libc6-dev-i386 \
libpcre3-dev \
libc6-dev-i386 \
pkg-config 

# Install ARM build utilities
RUN apt-get install -y \
binutils-aarch64-linux-gnu \
binutils-arm-linux-gnueabihf \
gcc-aarch64-linux-gnu \
gcc-arm-linux-gnueabihf \
gcc-multilib-arm-linux-gnueabihf

# Install Windows build utilities
RUN apt-get install -y \
binutils-mingw-w64 \
gcc-mingw-w64

RUN make prerelease; exit 0

RUN make prerelease && \
make release 

WORKDIR /go/src/github.com/hashicorp/nomad/pkg

RUN ls
