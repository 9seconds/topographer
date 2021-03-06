Topographer
===========

[![CI](https://github.com/9seconds/topographer/workflows/CI/badge.svg?branch=master)](https://github.com/9seconds/topographer/actions)
[![Code Coverage](https://codecov.io/gh/9seconds/topographer/branch/master/graph/badge.svg?token=CQSrBMbxh9)](https://codecov.io/gh/9seconds/topographer)
[![Go Reference](https://pkg.go.dev/badge/github.com/9seconds/topographer.svg)](https://pkg.go.dev/github.com/9seconds/topographer)

Fast and lenient self-hosted IP geolocation service.

Sometimes you need to detect regions and cities of different IPs. There
are a bunch of databases and free services available but you need to
have a code which works with these database or deal with limitations of
these services. For example, ipinfo.io or freegeoip.net limit your queries.

Sometimes you need your own service which responds with country/city
and does not have such limitations. Most of such services are based on
free versions of geolocation databases so it makes sense to have a free
self-hosted service which you can simply plug into your infrastructure.

Also, if you ever deal with IP geolocation you may know it is awfully
imprecise. There are many situations when one database detects one
city, another - slightly different location. Also, if you deal with
non-residential IPs you may know that a lot of hosters and clouds have a
weird route setup so you may have differences even in countries!

Just look at this example: https://bgpview.io/ip/191.96.13.80 Which
country does this IP belong?

This service goes in slightly different way: it uses a couple of
databases, collects their results, combine and consolidate results and
return a final one.

It queries all providers (or a limited set of them), picks the most
popular country. Within this country group, it picks the most popular
city and returns this tuple as a result.


Building
========

Building is trivial.

1. Install [Golang](https://golang.org/doc/install);
2. Run `go get github.com/9seconds/topographer`

or if you want to build from sources:

```shell
$ git clone https://github.com/9seconds/topographer
$ cd topographer
$ go build
```

or simple build Docker container

```shell
$ docker build -t topographer .
```


Installing
==========

Installation is simple as

```shell
$ go get github.com/9seconds/topographer
```

(but if you want, you can find prebuilt binaries on [releases page](https://github.com/9seconds/topographer/releases))

We also have Docker images in both DockerHub and Github Container Registry:

```shell
$ docker pull nineseconds/topographer
```

and

```shell
$ docker pull ghcr.io/9seconds/topographer:master
```

Running the application
=======================

A binary has a single cli flag: `-config`.

```shell
$ topographer -config /path/to/config.hjson
```

or if you run with docker, just put config as `/config.hjson` there:

```shell
$ docker run -v /path/to/local/config.hjson:/config.hjson -p 8000:80 nineseconds/topographer
```

API
===

Please see [OpenAPI specification](https://github.com/9seconds/topographer/blob/master/openapi.yaml).
