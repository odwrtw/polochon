#!/bin/bash

find . -name "*.coverprofile" -exec rm {} \;

for pkg in `find modules lib -maxdepth 1 -type d -links 2`
do
    go test -v ./${pkg} -coverprofile "${pkg}.coverprofile"
done

gover

$HOME/gopath/bin/goveralls -coverprofile=gover.coverprofile -service=travis-ci
