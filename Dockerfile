# This Dockerfile builds topographer image based on Alpine Linux.
# This is minimal image as possible
#
# To run this service you need to:
#     1. Mount config to /config.toml within a container
#     2. Propagate TOPOGRAPHER_IP2LOCATION_DOWNLOAD_TOKEN environment
#        variable with ip2location token to the container (you can get
#        this token after registration at ip2location.com)
#     3. Map exposed 80 port to any port you like
#
# Please visit https://github.com/9seconds/topographer for the details.

###############################################################################
# BUILD STAGE

FROM golang:alpine AS build-env

RUN set -x \
  && apk --update add git make

ADD . /go/src/github.com/9seconds/topographer

RUN set -x \
  && cd /go/src/github.com/9seconds/topographer \
  && make deps \
  && make


###############################################################################
# PACKAGE STAGE

FROM alpine:3.7
LABEL maintainer="Sergey Arkhipov <nineseconds@yandex.ru>" version="0.0.1"

ENTRYPOINT ["/topographer"]
CMD ["-b", "0.0.0.0", "-p", "80", "/config.toml"]
EXPOSE 80

RUN set -x \
  && apk add --no-cache --update ca-certificates

COPY --from=build-env /go/src/github.com/9seconds/topographer/topographer /topographer
COPY --from=build-env /go/src/github.com/9seconds/topographer/example.config.toml /config.toml
