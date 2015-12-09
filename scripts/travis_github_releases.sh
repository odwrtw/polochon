#!/bin/bash

export BUILD_DATE=$(date -u "+%Y-%m-%d %H:%M:%S UTC")
export GIT_TAG=latest
export GITHUB_PROJECT=https://$GITHUBTOKEN@github.com/xunien/polochon

# Only add the GIT_TAG on master to avoid pushing the tag
# each time travis builds the project
if [ $TRAVIS_BRANCH == "master" ] && [ $TRAVIS_PULL_REQUEST == "false" ]
then
    git config --global user.email "builds@travis-ci.com"
    git config --global user.name "Travis CI"

    echo "Get the release of the $GIT_TAG from the github API"
    GITHUB_RELEASE_URL=`curl -H "Authorization: token $GITHUBTOKEN" https://api.github.com/repos/xunien/polochon/releases/tags/$GIT_TAG | grep -m 1 '"url"' | awk -F '"' '{ print $4 }'`

    if [ $GITHUB_RELEASE_URL != "" ]
    then
        echo "Deleting the old '$GIT_TAG' release with the URL $GITHUB_RELEASE_URL"
        curl -H "Authorization: token $GITHUBTOKEN" -X DELETE $GITHUB_RELEASE_URL
    fi

    echo "Deleting the current '$GIT_TAG' tag"
    git tag -d $GIT_TAG
    git push $GITHUB_PROJECT :refs/tags/$GIT_TAG

    echo "Pushing the $GIT_TAG to github"
    git tag $GIT_TAG -a -m "Generated tag from TravisCI build $TRAVIS_BUILD_NUMBER ($BUILD_DATE)"
    git push $GITHUB_PROJECT $GIT_TAG
else
    echo "No need to create the '$GIT_TAG' tag on this branch"
fi
