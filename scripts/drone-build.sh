#!/bin/bash

docker login -u $DOCKER_USERNAME -p $DOCKER_PASSWORD harbor.coxedgecomputing.com
export REGISTRY=harbor.coxedgecomputing.com
if [ -z "$DRONE_TAG" ];
then
  export DRONE_TAG=$DRONE_BUILD_NUMBER
fi
export IMAGE_NAME=coxedge/cluster-api-cox-controller:${DRONE_TAG}
make docker-build && make docker-push