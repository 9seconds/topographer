Topographer
===========

[![Build Status](https://travis-ci.org/9seconds/topographer.svg?branch=master)](https://travis-ci.org/9seconds/topographer)

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

This service goes in slightly different way: it uses 4 databases, all
free, and uses simple voting system to detect correct country + city.
This is not the best approach but it is better than blindly trust to the
one certain database.

It uses simple weighted majority algorithm to detect such cases. It
basically means that each provider has its own weight (set in the
configuration file). When this tool collects responses from different
databases, it sums all weights for each country (each provider votes
with its weight) and the winner is the region which has bigger sum. City
is choosen by the winning country and the same way.

At the current moment, following databases are supported:

* [MaxMind](https://www.maxmind.com) GeoIP2 Lite database
* [DB-IP](https://db-ip.com) Country / City
* [Sypex](https://sypexgeo.net) City
* [IP2Location](https://www.ip2location.com) LITE DB1 and DB3
* [Software77](http://software77.net/geo-ip/) GeoIP database

So you basically need to build the container, propagate correct
configuration file to it and set token for ip2location (could be
obtained from their user profile page, after you register).


Building
========

Building is trivial.

1. Install [Golang](https://golang.org/doc/install);
2. Install [dep](https://golang.github.io/dep/docs/installation.html);
3. Run `dep ensure` to fetch dependencies;
4. Run `go build` to build the project.

or simple build Docker container

```shell
$ docker build -t topographer .
```

Also, there is an image on docker hub:
https://hub.docker.com/r/nineseconds/topographer/


Running the application
=======================

First, prepare configuration file. You can use
[example one](https://github.com/9seconds/topographer/blob/master/example.config.toml)
from the repository.

Here you can set weights for every provider, enable or disable it.
Also, you can set update interval (a duration string is a possibly
signed sequence of decimal numbers, each with optional fraction and a
unit suffix, such as `300ms`, `-1.5h` or `2h45m`. Valid time units are
`ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`), directory where to store
downloaded databases and precision (`city` or `country`).

Path to the application config has to be set as only as a single
argument for the topographer.

Also, you may want to bind topographer API to some host and port since
default host is 127.0.0.1. To do that, please use CLI flags.

So, in general, you should run you app as

```shell
$ topographer -b 0.0.0.0 -p 80 /path/to/the/config.toml
```

In case of docker, you can run application as following:

```shell
docker run \
  -e TOPOGRAPHER_IP2LOCATION_DOWNLOAD_TOKEN=111 \
  -v /path/to/local/config.toml:/config.toml:ro \
  -p 20000:80 \
  topographer
```


Some comments on databases
--------------------------

All databases are downloaded on the start of application and autoupdated
each `update_each` interval. All databases except of db-ip works as the
local files but db-ip unfortunately at the current moment has to be
loaded into the memory completely. Even if we do some tricks like
reusing structures in the radix tree, memory consumption can be huge.

So we recommend to have 512MB RAM available for country level
precision and 4GB RAM - for city level.


API
===

API is simple and straightforward:

- `GET /`: resolve my current IP
- `POST /`: resolve IPs posted in the body. Format is following:
  `{"ips": ["192.168.1.1", "10.10.0.10"], "providers": ["maxmind", "dbip"]}`.

  If you omit `providers` field or set it to empty array, topographer
  will use all possible databases (which are, of course, downloaded and
  ready and enabled in the config).
- `GET /info`: basic information on providers, last update etc.
