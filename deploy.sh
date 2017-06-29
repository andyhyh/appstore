#!/bin/bash 
set -euo pipefail
TAG=$1
QUAY_REPO=quay.io/uninett/k8s-appstore-backend

docker build -t $QUAY_REPO:$TAG .
docker login -u="$DOCKER_USERNAME" -p="$DOCKER_PASSWORD" quay.io
docker push $QUAY_REPO:$TAG
