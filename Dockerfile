# This Dockerfile builds topographer image based on Alpine Linux.
# This is minimal image as possible
#
# Please visit https://github.com/9seconds/topographer for the details.

###############################################################################
# BUILD STAGE

FROM golang:1.15-alpine AS build-env

ENV CGO_ENABLED=0

RUN set -x \
  && apk --update add \
    ca-certificates \
    git \
    make

ADD . /app
WORKDIR /app

RUN set -x \
  && make clean \
  && git submodule update --init \
  && make -j 4 static-build


###############################################################################
# PACKAGE STAGE

FROM scratch

ENTRYPOINT ["/topographer"]
CMD ["-config", "/config.hjson"]
EXPOSE 80

COPY --from=build-env \
    /etc/ssl/certs/ca-certificates.crt \
    /etc/ssl/certs/ca-certificates.crt
COPY --from=build-env \
    /app/topographer \
    /topographer
COPY --from=build-env \
    /app/example.config.hjson \
    /config.hjson
