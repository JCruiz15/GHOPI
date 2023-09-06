#!/bin/bash

source .env

# Build docker image
docker build -t ghopi:latest .

# Run docker image with environment variables
docker run \
    -e GITHUB_CLIENTID=$GITHUB_CLIENTID \
    -e GITHUB_SECRETID=$GITHUB_SECRETID \
    -e OPENPROJECT_CLIENTID=$OPENPROJECT_CLIENTID \
    -e OPENPROJECT_SECRETID=$OPENPROJECT_SECRETID \
    -e PORT=$PORT \
    -p $PORT:$PORT \
    ghopi:latest