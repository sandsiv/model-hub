#!/bin/bash

set -e

REPO="europe-docker.pkg.dev/sandsiv-infrastructure/vochub"
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null)

COMMITS_AFTER_TAG=$(git rev-list ${LATEST_TAG}..HEAD --count)

if [ "$COMMITS_AFTER_TAG" -gt 0 ]; then
    echo "Found $COMMITS_AFTER_TAG commits after $LATEST_TAG"
    echo "Exiting"
    exit 1
fi

docker buildx create --name builder-no-cuda
docker buildx build --builder builder-no-cuda --platform linux/amd64,linux/arm64 -t ${REPO}/model-hub:${LATEST_TAG}-no-cuda --file Dockerfile-no-cuda --push .
docker buildx rm builder-no-cuda
