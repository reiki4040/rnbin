#!/bin/bash
# glide install

VERSION=0.0.1
HASH=$(git rev-parse --verify HEAD)
GOVERSION=$(go version | cut -d' ' -f3)

export GO15VENDOREXPERIMENT=1

mkdir -p bin
gox -output="bin/rnbin_{{.OS}}_{{.Arch}}" \
    -os="linux" -os="darwin" -arch="amd64" \
    -ldflags="-w -X main.version=$VERSION -X main.hash=$HASH -X \"main.goversion=$GOVERSION\""

echo "finished"


