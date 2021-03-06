#!/bin/bash

#--------------------------------------------------------------#
# build RNBin on golang docker image.
# 1. create golang and glide enabled docker image.
# 2. mount local RNBin directory on container.
# 3. build RNBin with build.sh on container.
# 4. stored binary to mounted directory.
# 5. got RNBin binary that compiled docker image go version.
#--------------------------------------------------------------#

function usage() {
  cat <<_EOB
RNBin build script with docker.

  - build RNBin binary.
  - create release archive and show sha256 (for homebrew formula)

[Options]
  -a: create archive for release
  -g: run glide up

_EOB
}

function build() {
  local opt=""
  if [ $mode = "archive" ]; then
    opt="-a"
  fi

  if [ -n "$glideup" ]; then
    opt="$opt -g"
  fi

  # build golang + glide image
  docker build -t myglide:latest .

  # run RNBin build with docker
  docker run --rm \
		  -v $GOPATH/src/github.com/reiki4040/rnbin:/go/src/github.com/reiki4040/rnbin \
		  -w /go/src/github.com/reiki4040/rnbin \
		  myglide:latest bash build.sh $opt
}

mode="build"
glideup=
while getopts agh OPT
do
  case $OPT in
    a) mode="archive"
       ;;
    g) glideup="1"
       ;;
    h) usage
       exit 0
       ;;
    *) echo "unknown option."
       usage
       exit 1
       ;;
  esac
done
shift $((OPTIND - 1))

build
