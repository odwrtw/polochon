#!/usr/bin/env bash

set -o errexit
set -o pipefail
# FIXME: /home/travis/build.sh: line 53: assert: unbound variable
# set -o nounset

readonly SCRIPT_NAME=$(basename $0)
readonly TAG_DATE=$(date -u "+%Y-%m-%d %H:%M:%S UTC")
readonly GIT_TAG=latest
readonly GITHUB_PROJECT=https://${GITHUB_TOKEN}@github.com/odwrtw/polochon
readonly RELEASE_DESC="Generated from TravisCI build ${TRAVIS_BUILD_NUMBER} (${TAG_DATE})"

log() {
    echo -e "\e[36m-->\e[39m \e[35m$@\e[39m"
    logger -p user.notice -t $SCRIPT_NAME "$@"
}

tag() {
    log "Calling tag()"

    git config --global user.email "builds@travis-ci.com"
    git config --global user.name "Travis CI"

    git fetch ${GITHUB_PROJECT} --tags

    if [ -n "$(git tag -l ${GIT_TAG})" ]
    then
        log "Deleting existing tag ${GIT_TAG}..."
        git tag -d ${GIT_TAG}
        git push ${GITHUB_PROJECT} :refs/tags/${GIT_TAG}
    fi

    log "Creating tag ${GIT_TAG}..."
    git tag ${GIT_TAG} -a -m "${RELEASE_DESC}"
    git push ${GITHUB_PROJECT} ${GIT_TAG}
}

release() {
    log "Calling release()"

    log "Installing github-release..."
    go get github.com/aktau/github-release

    if [ -z "$(github-release info -t ${GIT_TAG} 2>&1 >/dev/null)" ]
    then
        log "Deleting existing release with tag ${GIT_TAG}..."
        github-release delete --tag ${GIT_TAG}
    fi

    log "Creating fresh release ${GIT_TAG}..."
    github-release release \
        --tag  ${GIT_TAG} \
        --name ${GIT_TAG} \
        --description "${RELEASE_DESC}"

    cd builds
    for BIN_NAME in $(ls polochon_*)
    do
        log "Pushing ${BIN_NAME} to release ${GIT_TAG}..."
        github-release upload \
            --tag  ${GIT_TAG} \
            --name ${BIN_NAME} \
            --file ${BIN_NAME}
    done
}

if [ ${TRAVIS_BRANCH} == "master" ] && [ ${TRAVIS_PULL_REQUEST} == "false" ]
then
    tag
    release
else
    log "No need to create the ${GIT_TAG} tag on this branch"
fi
