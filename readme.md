<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Table of Contents**

- [MovieNight stream server](#movienight-stream-server)
    - [Build requirements](#build-requirements)
        - [Older Go Versions](#older-go-versions)
        - [Compile and install](#compile-and-install)
        - [Docker build](#docker-build)
            - [Building the Container](#building-the-container)
            - [Running the Container](#running-the-container)
            - [docker-compose](#docker-compose)
            - [Notes for Running Using docker-compose](#notes-for-running-using-docker-compose)
    - [Usage](#usage)
    - [Configuration](#configuration)

<!-- markdown-toc end -->
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

### Compile and install

To just download and run:

```bash
$ git clone https://github.com/zorchenhimer/MovieNight
$ cd MovieNight
$ make
$ ./MovieNight
```

### Docker build
MovieNight provides a Dockerfile and a docker-compose file to run MovieNight using Docker.

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

#### docker-compose
docker-compose will automatically build the image, no need to build it manually.

Install Docker and docker-compose, clone the repository and change into the directory *./docker*. Then run:

```shell
docker-compose up -d
```

This docker-compose file will create a volume called *movienight-config* and automatically add the standard *settings.json* file to it. It also maps port 8089 and 1935 to the same ports of the host.

#### Notes for Running Using docker-compose
The container needs to be restarted to apply any changes you make to *settings.json*.

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
  -e bool
        Whether or not to download approved emotes on startup (default "false")
  -k string
        Stream key, to protect your stream (default: "")
  -l string
        host:port of the MovieNight (default ":8089")
  -r string
        host:port of the RTMP server (default ":1935")
```

## Configuration

MovieNight’s configuration is controlled by `settings.json`:

- `AdminPassword`: users can enter `/auth <value>` into chat to grant themselves
  admin privileges.  This value is automatically regenerated unless
  `RegenAdminPass` is false.
- `ApprovedEmotes`: list of Twitch users whose emotes can be imported into
  MovieNight.  Using `/addemotes <username>` in chat will add to this list.
- `Bans`: list of banned users.
- `LetThemLurk`: if false, announces when a user enters and leaves chat.
- `ListenAddress`: the port that MovieNight listens on, formatted as `:8089`.
- `LogFile`: the path of the MovieNight logfile, relative to the executable.
- `LogLevel`: the log level, defaults to `debug`.
- `MaxMessageCount`: the number of messages displayed in the chat window.
- `NewPin`: if true, regenerates `RoomAccessPin` when the server starts.
- `PageTitle`: The base string used in the `<title>` element of the page.  When
  the stream title is set with `/playing`, it is appended; e.g., `Movie Night | The Man Who Killed Hitler and Then the Bigfoot`
- `RegenAdminPass`: if true, regenerates `AdminPassword` when the server starts.
- `RoomAccess`: the access policy of the chat room; this is managed by the
  application and should not be edited manually.
- `RoomAccessPin`: if set, serves as the password required to enter the chatroom.
- `SessionKey`: key used for storing session data (cookies etc.)
- `StreamKey`: the key that OBS will use to connect to MovieNight.
- `StreamStats`: if true, prints statistics for the stream on server shutdown.
- `TitleLength`: the maximum allowed length for the stream title (set with `/playing`).
- `TwitchClientID`: OAuth client ID for the Twitch API, used for fetching emotes
- `TwitchClientSecret`: OAuth client secret for the Twitch API; [can be generated locally with curl](https://dev.twitch.tv/docs/authentication/getting-tokens-oauth#oauth-client-credentials-flow).
- `WrappedEmotesOnly`: if true, requires that emote codes be wrapped in colons
  or brackets; e.g., `:PogChamp:`
- `RateLimitChat`: the number of seconds between each message a non-privileged
  user can post in chat.
- `RateLimitNick`: the number of seconds before a user can change their nick again.
- `RakeLimitColor`: the number of seconds before a user can change their color again.
- `RateLimitAuth`: the number of seconds between each allowed auth attempt
- `RateLimitDuplicate`: the numeber of seconds before a user can post a
  duplicate message.
- `NoCache`: if true, set `Cache-Control: no-cache, must-revalidate` in the HTTP
  header, to prevent caching responses.
