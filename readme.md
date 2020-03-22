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

## Docker
MovieNight provides a Dockerfile and a docker-compose file to run MovieNight using Docker.

### Dockerfile
#### Building the Container
Install Docker, clone the repository and build:

```shell
docker build -t movienight .
```

#### Running the Container
Run the image once it's built:

```shell
docker run -d -p 8089:8089 -p 1935:1935 [-v ./settings.json:/config/settings.json] movienight
```

Explanation:
- **-d** runs the container in the background.
- **-p 8089:8089** maps the MovieNight web interface to port 8089 on the server.
- **-p 1935:1935** maps the RTMP port for OBS to port 1935 (default RTMP port) on the server.
- **-v ./settings.json:/config/settings.json** maps the file *settings.json* into the container. [OPTIONAL]

### docker-compose
docker-compose will automatically build the image, no need to build it manually.

Install Docker and docker-compose, clone the repository and change into the directory *./docker*. Then run:

```shell
docker-compose up -d
```

This docker-compose file will create a volume called *movienight-config* and automatically add the standard *settings.json* file to it. It also maps port 8089 and 1935 to the same ports of the host.

#### Notes for Running Using docker-compose
The container needs to be restarted to apply any changes you make to *settings.json*.
