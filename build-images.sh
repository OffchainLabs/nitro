#!/bin/bash
set -e

function build_and_push_image () {
  local PLATFORM=$1
  local ARCH=$2
  local DOCKER_ORG=$3
  local IMAGE_NAME=$4
  local DOCKERFILE_NAME=${5:-Dockerfile}

  TAG=$DOCKER_ORG/$IMAGE_NAME:$ARCH

  echo "Building for platform $PLATFORM, pushing to $TAG"

  docker buildx build . --pull \
      --platform $PLATFORM \
      --file $DOCKERFILE_NAME \
      --tag $TAG \
      --load

  echo "Publishing..."
  docker push $TAG
}

function create_and_push_manifest() {
  local DOCKER_ORG=$1
  local IMAGE_NAME=$2

  echo "Publishing manifest..."
  docker manifest create $DOCKER_ORG/$IMAGE_NAME:latest \
    --amend $DOCKER_ORG/$IMAGE_NAME:amd64 \
    --amend $DOCKER_ORG/$IMAGE_NAME:arm64v8
  docker manifest push --purge $DOCKER_ORG/$IMAGE_NAME:latest
}

# Build testnode-scripts image
pushd testnode-scripts
build_and_push_image "linux/amd64" "amd64" "tmigone" "nitro-testnode-scripts"
build_and_push_image "linux/arm64/v8" "arm64v8" "tmigone" "nitro-testnode-scripts"
create_and_push_manifest "tmigone" "nitro-testnode-scripts"
popd

# Build network-gen image
pushd testnode-tokenbridge
build_and_push_image "linux/amd64" "amd64" "tmigone" "nitro-testnode-tokenbridge"
build_and_push_image "linux/arm64/v8" "arm64v8" "tmigone" "nitro-testnode-tokenbridge"
create_and_push_manifest "tmigone" "nitro-testnode-tokenbridge"
popd
