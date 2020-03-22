# MovieNight stream server

[![Build status](https://api.travis-ci.org/zorchenhimer/MovieNight.svg?branch=master)](https://travis-ci.org/zorchenhimer/MovieNight)

This is a single-instance streaming server with chat.  Originally written to
replace Rabbit as the platform for watching movies with a group of people
online.

## Build requirements

- Go 1.13 or newer
- GNU Make

### Older Go Versions

You can install a newer version of Go alongside your OS's distribution by
following the guide here: [https://golang.org/doc/install#extra_versions](https://golang.org/doc/install#extra_versions)

Once you have that setup add an enviromnent variable named `GO_VERSION` and
set it to the version you installed (eg, `1.14.1`).  The Makefile will now use
the newer version.

## Install

To just download and run:

```bash
$ git clone https://github.com/zorchenhimer/MovieNight
$ cd MovieNight
$ make
$ ./MovieNight
```

## Usage

Now you can use OBS to push a stream to the server.  Set the stream URL to

```text
rtmp://your.domain.host/live
```

and enter the stream key.

Now you can view the stream at

```text
http://your.domain.host:8089/
```

There is a video only version at

```text
http://your.domain.host:8089/video
```

and a chat only version at

```text
http://your.domain.host:8089/chat
```

The default listen port is `:8089`.  It can be changed by providing a new port
at startup:

```text
Usage of .\MovieNight.exe:
  -k string
        Stream key, to protect your stream
  -l string
        host:port of the MovieNight (default ":8089")
```
