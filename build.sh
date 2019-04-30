#!/bin/sh
docker build . -f Dockerfile.builder -t builder
docker run -v "$(pwd)":/vol builder
