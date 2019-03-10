# Golang rtmp server demo

This is a very tiny demo with rtmp protocol server/client side implement.

## Requirement

You need golang to build all tools.

## Install

```bash
go get -u -v github.com/zorchenhimer/MovieNight

~/go/bin/MovieNight  -l :8089 -k  longSecurityKey
```

## Usage

now you can using obs to push stream to rtmp server

the stream url maybe ```rtmp://your.domain.host/live?key=longSecurityKey```

You can using obs to stream

Now you may visit the demo at

```text
http://your.domain.host:8089/
```

the :8089 is the default listen port of the http server. and you can change it as you want

```text
Usage of .\MovieNight.exe:
  -k string
        Stream key, to protect your stream
  -l string
        host:port of the MovieNight (default ":8089")
```