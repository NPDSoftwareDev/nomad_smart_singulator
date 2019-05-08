#!/bin/sh
while getopts ":n:s:b:t:" opt; do
  case $opt in
    s) SHA1="$OPTARG"
    ;;
    b) BRANCH="$OPTARG"
    ;;
    t) TAG="$OPTARG"
    ;;
    n) NET="$OPTARG"
    ;;
    \?) echo "Invalid option -$OPTARG" >&2
    ;;
  esac
done

docker build -t npdsoftwaredev/nomad_offline:${SHA1} -f .buildkite/Dockerfile --network=cluster_network --build-arg JFROG_UNAME=${JFROG_UNAME} --build-arg JFROG_PWORD=${JFROG_PWORD} .