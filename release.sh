#!/bin/bash

# Get the current date and time in YYYY-MM-DD-HH-MM-SS format
datetime=$(date +'%Y-%m-%d-%H-%M-%S')


# Build the Docker image with the 'latest' tag for the target platform (Intel x64)
docker buildx build --platform linux/amd64 -t epicmo/newapi:latest --push .

# Build the Docker image with a date-time tag for the target platform (Intel x64)
docker buildx build --platform linux/amd64 -t epicmo/newapi:$datetime --push .