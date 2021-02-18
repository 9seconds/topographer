# This Dockerfile builds topographer image based on Alpine Linux.
# This is minimal image as possible
#
# Please visit https://github.com/9seconds/topographer for the details.

###############################################################################
# BUILD STAGE

FROM golang:1.15-alpine AS build-env

RUN set -x \
  && apk --update add git make

ADD . /go/src/github.com/9seconds/topographer
WORKDIR /go/src/github.com/9seconds/topographer

RUN set -x \
  && make clean \
  && git submodule update --init \
  && make -j 4 static-build


###############################################################################
# PACKAGE STAGE

FROM alpine:latest

ENTRYPOINT ["/topographer"]
CMD ["-config", "/config.hjson"]
EXPOSE 80

RUN set -x \
  && apk add --no-cache --update ca-certificates

COPY --from=build-env \
    /go/src/github.com/9seconds/topographer/topographer \
    /topographer
COPY --from=build-env \
    /go/src/github.com/9seconds/topographer/example.config.hjson \
    /config.hjson
