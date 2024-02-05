#!/bin/bash

# Check for the presence of an argument (version)
if [ "$#" -ne 1 ]; then
    echo "Error: A version argument is required."
    exit 1
fi

# Save the version to a variable
VERSION=$1

# Create and use a new builder instance
docker buildx create --name mybuilder --use

# Build and push the image for model-hub
docker buildx build --platform linux/amd64,linux/arm64 -t devxpro/model-hub:$VERSION --push .
docker buildx build --platform linux/amd64,linux/arm64 -t devxpro/model-hub:latest --push .

# Build and push the image for model-hub-no-cuda
docker buildx build --platform linux/amd64,linux/arm64 -t devxpro/model-hub:${VERSION}-no-cuda --file Dockerfile-no-cuda --push .
docker buildx build --platform linux/amd64,linux/arm64 -t devxpro/model-hub:latest-no-cuda --file Dockerfile-no-cuda --push .

docker buildx use default
docker buildx rm mybuilder
