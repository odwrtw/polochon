#!/bin/sh

set -e

PKG_NAME=github.com/odwrtw/polochon
BASE_PATH=$(git rev-parse --show-toplevel)
BUILD_DIR="$BASE_PATH/builds"
BIN_NAME=polochon

[ -d "$BUILD_DIR" ] || mkdir "$BUILD_DIR"

_log() {
	printf "$(tput setaf 5)-->$(tput setaf 2) %s$(tput setaf 7)\n" "$@"
}

_clean() {
	for file in "$BUILD_DIR"/*; do
		path=$(readlink -f "$file")
		[ -f "$path" ] || continue
		rm "$file"
		_log "Removing $file"
	done
}

_checksum() {
	for file in "$BUILD_DIR"/*; do
		path=$(readlink -f "$file")
		[ -x "$path" ] || continue
		_log "Creating checksum file for $file"
		sha256sum "$file" > "$file.sha256sum"
	done
}

_build() {
	_clean
	to_build="$(go env GOOS) $(go env GOARCH)"
	[ "$2" = "all" ] && to_build="
		linux amd64
		linux arm
		darwin amd64
	"

	echo "$to_build" | while read -r os arch; do
		[ "$os" ] || continue
		_log "Building $BIN_NAME for $os/$arch"
		(
			GOOS=$os \
			GOARCH=$arch \
			go build \
				-o "$BUILD_DIR/${BIN_NAME}_${os}_${arch}" \
				"$BASE_PATH/app/."
		)
	done

	_checksum
}

_coverage() {
	_log "Generating cover profiles..."
    coverprofile="$BUILD_DIR/code.coverage"
	echo "mode: set" > "$coverprofile"
	for pkg in $(go list "$BASE_PATH/..."); do
		path="$BASE_PATH/${pkg##$PKG_NAME}"
		coverfile="$path/cover.coverprofile"
		go test -coverprofile="$coverfile" "$path"
		[ "$(wc -l < "$coverfile")" = "1" ] && continue
		grep -v "mode" "$coverfile" >> "$coverprofile"
	done
	_log "Cleaning up cover profiles..."
	find "$BASE_PATH" -name '*.coverprofile' -delete

	[ "$TRAVIS" = "true" ] || return 0

	_log "Downloading goveralls..."
	goveralls_bin="$GOPATH/bin/goveralls"
	[ -x "$goveralls_bin" ] || \
		GO111MODULES=off go get -u -v github.com/mattn/goveralls

	_log "Uploading coverprofile..."
	"$goveralls_bin" -coverprofile="$coverprofile" -service=travis-ci

	_log "Cleaning up..."
	rm "$coverprofile"
}

_test() {
	go test -cover "$BASE_PATH/..."
}

_usage() {
	echo "$(basename "$0") usage:"
	echo "		build [all] - Build [for all achitecture]"
	echo "		clean       - Clean the builds"
	echo "		test        - Test the packages"
	echo "		coverage    - Generate *.coverprofile"
	exit 1
}

case "$1" in
	build)
		_build "$@"
		;;
	clean)
		_clean
		;;
	test)
		_test
		;;
	coverage)
		_coverage
		;;
	*)
		_usage
		;;
esac
