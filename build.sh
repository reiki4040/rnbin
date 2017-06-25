#!/bin/bash
VERSION=$(git describe --tags)
HASH=$(git rev-parse --verify HEAD)
GOVERSION=$(go version)

ARCHIVE_INCLUDES_FILES="LICENSE README.md"

function usage() {
  cat <<_EOB
rnbin build script.

  - build rnbin binary.
  - create release archive and show sha256

[Options]
  -a: create archive for release
  -g: run glide up when build
  -s: show current build version for check
  -q: quiet mode

_EOB
}

function show_build_version() {
  echo $VERSION
}

quiet=""
function msg() {
  test -z "$quiet" && echo $*
}

function err_exit() {
  echo $* >&2
  exit 1
}

function build() {
  local dest_dir=$1
  local os=$2
  if [ -z "$os" ]; then
    os="darwin"
  fi

  if [ -n "$glideup" ]; then
    msg "run glide up..."
    if [ -n "$quiet" ]; then
      glide -q up
	else
      glide up
	fi
  fi

  msg "start build rnbin..."
  GOOS="$os" GOARCH="amd64" go build -o "$dest_dir/rnbin" -ldflags "-X main.version=$VERSION -X main.hash=$HASH -X \"main.goversion=$GOVERSION\""
  msg "finished build rnbin."
}

function create_archive() {
  local work_dir="work"
  local dest_dir="archives"
  local current=$(pwd)
  if [ -z "$current" ]; then
    exit 1
  fi

  mkdir -p $current/$dest_dir

  for os in "darwin" "linux"; do
    msg "start $os/amd64 build and create archive file."

    rnbin_prefix="rnbin-$VERSION-$os-amd64"
    archive_dir="$current/$work_dir/$rnbin_prefix"
    mkdir -p "$archive_dir"

    # build
    build "$archive_dir" "$os"

    # something
    for f in $ARCHIVE_INCLUDES_FILES
    do
      cp -a $current/$f $archive_dir/
    done

    msg "creating archive..."
    cd $current/$work_dir

    local taropt="czvf"
    if [ -n "$quiet" ]; then
    taropt="czf"
    fi
    tar $taropt "$rnbin_prefix".tar.gz "./$rnbin_prefix"

    mv "$rnbin_prefix".tar.gz $current/$dest_dir/
    shasum -a 256 "$current/$dest_dir/$rnbin_prefix.tar.gz"
    msg "finished $os/amd64 build and create archive file."
    echo ""
  done
}

mode="build"
glideup=""
while getopts ashqu OPT
do
  case $OPT in
    a) mode="archive"
       ;;
    s) show_build_version
       exit 0
       ;;
    u) glideup="1"
       ;;
    h) usage
       exit 0
       ;;
    q) quiet=1
       ;;
    *) echo "unknown option."
       usage
       exit 1
       ;;
  esac
done
shift $((OPTIND - 1))

# run build or archive
case $mode in
  "build")
    build $(pwd)
    ;;
  "archive")
    create_archive
    ;;
  *)
    echo "unknown mode"
    usage
    exit 1
    ;;
esac
