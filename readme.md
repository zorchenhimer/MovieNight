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
        - [FreeNAS / TrueNAS / FreeBSD build and run](#freenas-freebsd-build-and-run)
        - [Clever-Cloud deployment and run](#clever-cloud-deployment-and-run)
    - [Usage](#usage)
    - [Configuration](#configuration)

<!-- markdown-toc end -->
# MovieNight stream server
[![Build status](https://api.travis-ci.org/zorchenhimer/MovieNight.svg?branch=master)](https://travis-ci.org/zorchenhimer/MovieNight)

This is a single-instance streaming server with chat. Originally written to replace Rabbit as the platform for watching movies with a group of people online.

## Build requirements
- Go 1.16 or newer
- GNU Make

### Older Go Versions
You can install a newer version of Go alongside your OS's distribution by following the guide here: [https://golang.org/doc/manage-install](https://golang.org/doc/manage-install)

Once you have that setup add an enviromnent variable named `GO_VERSION` and set it to the version you installed (eg, `1.16.1`). The Makefile will now use the newer version.

### Compile and install
You have to: 
- download `git clone https://github.com/zorchenhimer/MovieNight`, go into the source directory `cd MovieNight`;
- run `go build`

If you want to cross compile instead of running `go build`:
- choose your `TARGET` oneof "android darwin dragonfly freebsd linux nacl netbsd openbsd plan9 solaris windows";
- choose your `ARCH` oneof "386 amd64 amd64p32 arm arm64 ppc64 ppc64le mips mipsle mips64 mips64le mips64p32 mips64p32leppc s390 s390x sparc sparc64";
- build `make TARGET=windows ARCH=386` (On BSD systems use `gmake`);
- and run `./MovieNight`;

Example:
```shell
$ git clone https://github.com/zorchenhimer/MovieNight
$ cd MovieNight
$ (make|gmake) TARGET=windows ARCH=386
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
# with default settings file (this uses the settings_example.json config file)
docker run -d -p 8089:8089 -p 1935:1935 movienight

# using a custom settings file
docker run -d -p 8089:8089 -p 1935:1935 -v ./settings.json:/data/config/settings.json movienight
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

### FreeNAS-FreeBSD build and run
A [FreeNAS & TrueNAS plugin](https://github.com/zorglube/iocage-plugin-movienight) had been released. You should find MovieNight into the plugin section of you management GUI. However you still can make an manual plugin deployment, documentation [here](https://github.com/freenas/iocage-ix-plugins)
If you prefer to make an Jail without using the plugin management, a script wich setup an Jail and build and run MovieNight into that Jail as been written, you'll find it here [freenas-iocage-movienight](https://github.com/zorglube/freenas-iocage-movienight)  

### Clever-Cloud deployment and run
If you don't like to handle the build and run of your MovieNight instance, here is an samll manual of "how to make it run on [Clever-Cloud](https://www.clever-cloud.com)".
Into your Clever-Cloud dashboard:
 - Create a "Brand New App" and choose `Go` runtime instance
 - Name it, choose your datacenter
 - You don't neet any "Add-On", unless you want to provide some Emotes to you MovieNight instance
 - Add thoses environement-variables (expert mode allow you copy past frome this page):
    `CC_GO_BUILD_TOOL="gobuild"` Set The build method
    `CC_GO_PKG="github.com/zorchenhimer/MovieNight"` Set the `Go` dependencies origin
    `CC_PRE_RUN_HOOK="echo \"{\\\"ApprovedEmotes\\\": true, \\\"Bans\\\": [], \\\"LetThemLurk\\\": false, \\\"ListenAddress\\\": \\\":8080\\\", \\\"LogFile\\\": \\\"thelog.log\\\", \\\"LogLevel\\\": \\\"debug\\\", \\\"MaxMessageCount\\\": 300, \\\"NoCache\\\": false, \\\"NewPin\\\": true, \\\"PageTitle\\\": \\\"Movie Night\\\", \\\"RateLimitAuth\\\": 5, \\\"RateLimitChat\\\": 1, \\\"RateLimitColor\\\": 60, \\\"RateLimitDuplicate\\\": 30, \\\"RateLimitNick\\\": 300, \\\"RegenAdminPass\\\": true, \\\"RtmpListenAddress\\\": \\\":4040\\\", \\\"RoomAccess\\\": \\\"pin\\\", \\\"RoomAccessPin\\\": \\\"9999\\\", \\\"StreamKey\\\": \\\"ALongStreamKey\\\", \\\"StreamStats\\\": true, \\\"TitleLength\\\": 50, \\\"WrappedEmotesOnly\\\": true}\" >> /home/bas/go_home/bin/settings.json"` The command that will build a default configuration file `settings.json`
    `CC_RUN_COMMAND="${MN_BIN_DIR}/${MN_BIN_NAME} -l ${MN_PORT} -r ${MN_RTMP} -s ${MN_STATIC} -k ${MN_STREAM_KEY}"` The run command
    `MN_BIN_DIR="/home/bas/go_home/bin"` The built binary directory
    `MN_BIN_NAME="MovieNight"` The name of the runnable bin
    `MN_PORT=":8080"` The http port
    `MN_RTMP=":4040"` The rtmp port
    `MN_STATIC="${APP_HOME}/static"` The static contents dir
    `MN_STREAM_KEY="YourStreamKey"` Your secret stream key, you have to change it
 - Turn on the TCP redirection
 - Try run your MovieNight instance

```text
CC_GO_BUILD_TOOL="gobuild"
CC_GO_PKG="github.com/zorchenhimer/MovieNight"
CC_PRE_RUN_HOOK="echo \"{\\\"ApprovedEmotes\\\": true, \\\"Bans\\\": [], \\\"LetThemLurk\\\": false, \\\"ListenAddress\\\": \\\":8080\\\", \\\"LogFile\\\": \\\"thelog.log\\\", \\\"LogLevel\\\": \\\"debug\\\", \\\"MaxMessageCount\\\": 300, \\\"NoCache\\\": false, \\\"NewPin\\\": true, \\\"PageTitle\\\": \\\"Movie Night\\\", \\\"RateLimitAuth\\\": 5, \\\"RateLimitChat\\\": 1, \\\"RateLimitColor\\\": 60, \\\"RateLimitDuplicate\\\": 30, \\\"RateLimitNick\\\": 300, \\\"RegenAdminPass\\\": true, \\\"RtmpListenAddress\\\": \\\":4040\\\", \\\"RoomAccess\\\": \\\"pin\\\", \\\"RoomAccessPin\\\": \\\"9999\\\", \\\"StreamKey\\\": \\\"ALongStreamKey\\\", \\\"StreamStats\\\": true, \\\"TitleLength\\\": 50, \\\"WrappedEmotesOnly\\\": true}\" >> /home/bas/go_home/bin/settings.json"
CC_RUN_COMMAND="${MN_BIN_DIR}/${MN_BIN_NAME} -l ${MN_PORT} -r ${MN_RTMP} -s ${MN_STATIC} -k ${MN_STREAM_KEY}"
MN_BIN_DIR="/home/bas/go_home/bin"
MN_BIN_NAME="MovieNight"
MN_PORT=":8080"
MN_RTMP=":4040"
MN_STATIC="${APP_HOME}/static"
MN_STREAM_KEY="YourStreamKey"
```

## Usage
Now you can use OBS to push a stream to the server. Set the stream URL to

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

The default listen port is `:8089`. It can be changed by providing a new port at startup:

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
  -f string
        the settings file you want to use (default "./settings.json")
```

## Configuration
MovieNight’s configuration is controlled by `settings.json`:

    - `AdminPassword`: users can enter `/auth <value>` into chat to grant themselves admin privileges.  This value is automatically regenerated unless `RegenAdminPass` is false.
    - `Bans`: list of banned users.
    - `LetThemLurk`: if false, announces when a user enters and leaves chat.
    - `ListenAddress`: the port that MovieNight listens on, formatted as `:8089`.
    - `LogFile`: the path of the MovieNight logfile, relative to the executable.
    - `LogLevel`: the log level, defaults to `debug`.
    - `MaxMessageCount`: the number of messages displayed in the chat window.
    - `NewPin`: if true, regenerates `RoomAccessPin` when the server starts.
    - `PageTitle`: The base string used in the `<title>` element of the page.  When the stream title is set with `/playing`, it is appended; e.g., `Movie Night | The Man Who Killed Hitler and Then the Bigfoot`
    - `RegenAdminPass`: if true, regenerates `AdminPassword` when the server starts.
    - `RoomAccess`: the access policy of the chat room; this is managed by the application and should not be edited manually.
    - `RoomAccessPin`: if set, serves as the password required to enter the chatroom.
    - `SessionKey`: key used for storing session data (cookies etc.)
    - `StreamKey`: the key that OBS will use to connect to MovieNight.
    - `StreamStats`: if true, prints statistics for the stream on server shutdown.
    - `TitleLength`: the maximum allowed length for the stream title (set with `/playing`).
    - `WrappedEmotesOnly`: if true, requires that emote codes be wrapped in colons or brackets; e.g., `:PogChamp:`
    - `RateLimitChat`: the number of seconds between each message a non-privileged user can post in chat.
    - `RateLimitNick`: the number of seconds before a user can change their nick again.
    - `RakeLimitColor`: the number of seconds before a user can change their color again.
    - `RateLimitAuth`: the number of seconds between each allowed auth attempt.
    - `RateLimitDuplicate`: the numeber of seconds before a user can post a duplicate message.
    - `NoCache`: if true, set `Cache-Control: no-cache, must-revalidate` in the HTTP header, to prevent caching responses.

## License
`flv.js` is Licensed under the Apache 2.0 license. This project is licened under the MIT license.
