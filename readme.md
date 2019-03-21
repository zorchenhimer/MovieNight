# MovieNight stream server

[![Build status](https://api.travis-ci.org/zorchenhimer/MovieNight.svg?branch=master)](https://travis-ci.org/zorchenhimer/MovieNight)

This is a single-instance streaming server with chat.  Originally written to
replace Rabbit as the platform for watching movies with a group of people
online.

## Build requirements

- Go 1.12 or newer
- GNU Make

## Install

To just download and run:

```bash
go get -u -v github.com/zorchenhimer/MovieNight
MovieNight  -l :8089 -k longSecurityKey
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

## VsCode Setup

For development of this project in VsCode, the wasm has to be handled specially. Since the architecture is `WASM` and the OS is `js` for go wasm files, the tools must have those environment args. This can be solved by adding a `.vscode` folder at `MovieNight\wasm\.vscode` and having the settings below. When doing development in the `MovieNight\wasm`, open the folder in a new vscode instance. This will allow autocomplete to work in the `*_wasm.go` files.

```json
{
    "go.toolsEnvVars": {
        "GOOS": "js",
        "GOARCH": "wasm"
    }
}
```
