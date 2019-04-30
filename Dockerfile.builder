FROM golang:stretch

RUN apt-get update && apt-get -y --no-install-recommends install libsystemd-dev

VOLUME /vol
WORKDIR /vol
ENTRYPOINT ["go", "build"]
