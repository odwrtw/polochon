#!/bin/sh
# vim: set noexpandtab tabstop=4 shiftwidth=4 softtabstop=4 :

set -e

TAG_DATE=$(date -u "+%Y-%m-%d %H:%M:%S UTC")
GIT_TAG=latest
GITHUB_PROJECT=https://${GITHUB_TOKEN}@github.com/odwrtw/polochon
RELEASE_DESC="Generated from TravisCI build $TRAVIS_BUILD_NUMBER ($TAG_DATE)"

log() {
	# shellcheck disable=SC1117
	printf "\e[36m-->\e[39m \e[35m%s\e[39m\n" "$@"
}

tag() {
	log "Calling tag()"

	git config --global user.email "builds@travis-ci.com"
	git config --global user.name "Travis CI"

	git fetch "$GITHUB_PROJECT" --tags

	if [ -n "$(git tag -l "$GIT_TAG")" ]; then
		log "Deleting existing tag $GIT_TAG..."
		git tag -d "$GIT_TAG"
		git push "$GITHUB_PROJECT" ":refs/tags/$GIT_TAG"
	fi

	log "Creating tag $GIT_TAG..."
	git tag "$GIT_TAG" -a -m "$RELEASE_DESC"
	git push "$GITHUB_PROJECT" "$GIT_TAG"
}

release() {
	log "Calling release()"

	log "Installing github-release..."
	go get github.com/aktau/github-release

	if [ -z "$(github-release info -t "$GIT_TAG" 2>&1 >/dev/null)" ]; then
		log "Deleting existing release with tag $GIT_TAG..."
		github-release delete --tag "$GIT_TAG"
	fi

	log "Creating fresh release $GIT_TAG..."
	github-release release \
		--tag  "$GIT_TAG" \
		--name "$GIT_TAG" \
		--description "$RELEASE_DESC"

	for bin in builds/polochon_*; do
		log "Pushing $bin to release $GIT_TAG..."
		github-release upload \
			--tag  "$GIT_TAG" \
			--name "$bin" \
			--file "$bin"
	done
}

trigger_docker_build() {
	log "Triggering docker build"

	curl -H "Content-Type: application/json" --data '{"build": true}' \
		-X POST "https://registry.hub.docker.com/u/odwrtw/polochon/trigger/${DOCKER_BUILD_TOKEN}/"

	log "Docker build triggered"
}

if [ "$TRAVIS_BRANCH" = "master" ] && [ "$TRAVIS_PULL_REQUEST" = "false" ]; then
	tag
	release
	trigger_docker_build
else
	log "No need to create the $GIT_TAG tag on this branch"
fi
