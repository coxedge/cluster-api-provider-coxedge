#!/usr/bin/env bash

# build-and-push-image.sh - CI script for building and publishing cluster-api-provider-cox-controller Docker image.
#
# Parameters:
# - IMAGE_REGISTRY  Registry to publish the Docker image. By default '514845858982.dkr.ecr.us-west-1.amazonaws.com' is used.
# - IMAGE_NAME      Name to use for this image. By default 'cluster-api-provider-cox-controller' is used.
# - IMAGE_TAG       Tag to use for the image. By default '$PF9_VERSION-$BUILD_NUMBER' is used.
# - IMAGE_REGISTRY      URL (without scheme) pointing to the ECR registry. The
#                   script will try to authenticate for the specified ECR with
#                   the credentials in the environment and override
#                   IMAGE_REGISTRY with IMAGE_REGISTRY.
# - DRY_RUN         If non-empty, no Docker image will be published.
# - CONTAINER_TAG   Location of the container_tag file (used as an artifact in TeamCity)
# - DOCKER_USERNAME Username to login to Dockerhub.
# - DOCKER_PASSWORD Password to login to Dockerhub.
#
# Examples:
# - `USE_SYSTEM_GO=1 IMAGE_REGISTRY=docker.io IMAGE_NAME=coxedge/cluster-api-provider-cox-controller IMAGE_TAG=latest ./build-and-push-image.sh`: To test the script locally without gimme and push to Docker

set -o nounset
set -o errexit
set -o pipefail

project_root=$(realpath "$(dirname $0)/..")
build_dir=${project_root}/build
CONTAINER_TAG=${CONTAINER_TAG:-${build_dir}/container-tag}
CONTAINER_FULL_TAG=${CONTAINER_FULL_TAG:-${build_dir}/container-full-tag}
GO_VERSION=${GO_VERSION:-1.18}

BUILD_NUMBER=${BUILD_NUMBER:-000}
PF9_VERSION=${PF9_VERSION:-5.5.0}

# TODO add IMAGE_ORGANIZATION once ECR is set up with it
IMAGE_REGISTRY=${IMAGE_REGISTRY:-"514845858982.dkr.ecr.us-west-1.amazonaws.com"}
IMAGE_NAME=${IMAGE_NAME:-"cluster-api-cox-controller"}
IMAGE_TAG=${IMAGE_TAG:-${PF9_VERSION}-${BUILD_NUMBER}}
IMAGE_NAME_TAG=${IMAGE_NAME}:${IMAGE_TAG}
IMAGE_REGISTRY_NAME_TAG=${IMAGE_REGISTRY}/${IMAGE_NAME_TAG}

MANIFEST_BUILD_PATH=${build_dir}/manifest.yaml

main() {
  # Move to the project directory
  pushd "${project_root}"
  trap on_exit EXIT

  if [ -n "${BASH_DEBUG:-}" ]; then
      set -x
      PS4='${BASH_SOURCE}.${LINENO}+ '
  fi

  info "Verifying prerequisites"
  which aws > /dev/null || (echo "error: missing required command 'aws'" && exit 1)
  which docker > /dev/null || (echo "error: missing required command 'docker'" && exit 1)
  # note: go and/or gimme are checked in configure_go

  info "Preparing build environment"
  mkdir -p "${build_dir}"

  info "Configure Docker registry and create image repository if not present"
  configure_docker_registry "${IMAGE_NAME}"

  info "Configure go"
  configure_go

  info "Build Kubernetes manifest"
  make manifest-build IMG="${IMAGE_REGISTRY_NAME_TAG}" MANIFEST_BUILD_PATH="${MANIFEST_BUILD_PATH}"

  info "Verifying Kubernetes manifest"
  grep -q "image: ${IMAGE_REGISTRY_NAME_TAG}" "${MANIFEST_BUILD_PATH}"

  info "Build Docker image"
  make docker-build IMG="${IMAGE_REGISTRY_NAME_TAG}"

  info "Verifying code generation"
  make verify-generate

  info "Push Docker image"
  if [ -z "${DRY_RUN:-}" ] ; then
    make docker-push IMG="${IMAGE_REGISTRY_NAME_TAG}"
  else
    echo "DRY_RUN is set; not publishing the image"
  fi

  # TODO publish to latest if it is at the HEAD of the main branch

  info "Publish artifacts"
  mkdir -p "$(dirname "${CONTAINER_TAG}")" "$(dirname "${CONTAINER_FULL_TAG}")"
  echo -n "${IMAGE_TAG}" > "${CONTAINER_TAG}"
  echo -n "${IMAGE_REGISTRY_NAME_TAG}" > "${CONTAINER_FULL_TAG}"
  echo "Stored image tag in ${CONTAINER_TAG}:"
  cat "${CONTAINER_TAG}" && echo ""
  echo "Stored image full tag in ${CONTAINER_FULL_TAG}:"
  cat "${CONTAINER_FULL_TAG}" && echo ""
}

on_exit() {
  ret=$?
  info "-------cleanup--------"
  make docker-clean IMG="${IMAGE_REGISTRY_NAME_TAG}" || true
  popd
  exit ${ret}
}

configure_docker_registry() {
  repository=$1
  if [ "${IMAGE_REGISTRY}" = "docker.io" ]; then
    if [ -n "${DOCKER_PASSWORD:-}" ] ; then
      echo -n "${DOCKER_PASSWORD}" | docker login --username "${DOCKER_USERNAME}" --password-stdin "${IMAGE_REGISTRY}"
    else
      echo "Using default docker registry"
    fi
  else
    # Otherwise use AWS ECR
    aws ecr get-login-password | docker login --username AWS --password-stdin "${IMAGE_REGISTRY}"

    # Ensure that the repository exists
    aws ecr create-repository --repository-name "${repository}" || true
  fi
  echo "Configured registry '${IMAGE_REGISTRY}' for '${repository}'"
}

configure_go() {
  if [ -n "${USE_SYSTEM_GO:-}" ] ; then
    echo "\$USE_SYSTEM_GO set, using system go instead of gimme"
    return 0
  else
    which gimme > /dev/null || (echo "error: missing required command 'gimme'" && exit 1)
    eval "$(GIMME_GO_VERSION=${GO_VERSION} gimme)"
  fi
  which go
  go version
}

RED='\033[1;31m'
YELLOW='\033[1;33m'
NC='\033[0m'
info() { echo -e >&2 "${YELLOW}[INFO] $@${NC}" ; }
fatal() { echo >&2 "${RED}[FATAL] $@${NC}" ; exit 1 ; }

main $@
