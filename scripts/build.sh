#!/bin/sh
set -e

BASE_PATH=$(git rev-parse --show-toplevel)
BUILD_DIR="$BASE_PATH/builds"

mkdir -p "$BUILD_DIR"

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

_build_bin() {
	os=$1
	arch=$2
	app=$3
	name=$4

	_log "Building $name for $os/$arch"
	(
		GOOS=$os \
		GOARCH=$arch \
		CGO_ENABLED=0 \
		go build \
			-ldflags="-extldflags=-static" \
			-o "$BUILD_DIR/${name}_${os}_${arch}" \
			"$BASE_PATH/$app/."
	)
}

_build() {
	_clean
	to_build="$(go env GOOS) $(go env GOARCH)"
	[ "$2" = "all" ] && to_build="
		linux amd64
		linux arm
		darwin amd64
	"

	while read -r os arch; do
		[ -z "$os" ] && continue
		_build_bin "$os" "$arch" "app" "polochon"
		_build_bin "$os" "$arch" "cmd/polochonfs" "polochonfs"
	done <<-EOF
	$to_build
	EOF

	_checksum
}

_test() {
	go test -cover "$BASE_PATH/..."
}

_usage() {
	echo "$(basename "$0") usage:"
	echo "		build [all] - Build [for all achitecture]"
	echo "		clean       - Clean the builds"
	echo "		test        - Test the packages"
	exit 1
}

case "$1" in
	build) _build "$@" ;;
	clean) _clean      ;;
	test)  _test       ;;
	*)     _usage      ;;
esac
