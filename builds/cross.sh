#!/bin/bash

echo '
  linux amd64
  linux arm 6
  linux arm 7
' | {
  while read os arch armv; do
    [ -n "$os" ] || continue

    {
        export VERSION=`git rev-parse --short HEAD`
        export GIMME_OS="$os"
        export GIMME_TYPE="source"
        export GOOS="$os"
        export GIMME_ARCH="$arch"
        export GOARCH="$arch"
        export GOARM="$armv"
        eval "$(gimme 1.5)"

        echo "Building for $os $arch$armv" >&2
        binname="polochon_${os}_${arch}${armv}"
        time go build -a -v -o "$binname" -ldflags "-X main.minversion=$VERSION" server/*.go || {
          echo "Unable to build for $os $arch$armv" >&2
          exit 1
        }
        file $binname
    } &

    [ -z "`git tag -l | grep latest`" ] && break
  done
}

ls -1

wait
exit 0
