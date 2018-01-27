# Build Stage
FROM golang:alpine AS build-env

RUN set -x \
  && apk --update add git \
  && go get -u github.com/golang/dep/cmd/dep

ADD . /go/src/github.com/9seconds/topographer

RUN set -x \
  && cd /go/src/github.com/9seconds/topographer \
  && dep ensure \
  && go build -o topographer


# Package stage
FROM alpine:3.7
MAINTAINER Sergey Arkhipov <nineseconds@yandex.ru>

RUN set -x \
  && apk add --no-cache --update ca-certificates

COPY --from=build-env /go/src/github.com/9seconds/topographer/topographer /topographer
COPY --from=build-env /go/src/github.com/9seconds/topographer/example.config.toml /config.toml

ENTRYPOINT ["/topographer"]
CMD ["/config.toml"]
