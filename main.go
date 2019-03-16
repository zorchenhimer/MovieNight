package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/rtmp"
)

var (
	addr = flag.String("l", ":8089", "host:port of the MovieNight")
	sKey = flag.String("k", "", "Stream key, to protect your stream")
)

func init() {
	format.RegisterAll()
}

func main() {
	flag.Parse()
	server := &rtmp.Server{}

	server.HandlePlay = handlePlay
	server.HandlePublish = handlePublish

	// Chat websocket
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/static/js/", wsStaticFiles)
	http.HandleFunc("/static/css/", wsStaticFiles)
	http.HandleFunc("/static/main.wasm", wsStaticFiles)
	http.HandleFunc("/emotes/", wsEmotes)
	http.HandleFunc("/favicon.ico", wsStaticFiles)
	http.HandleFunc("/chat", handleIndexTemplate)
	http.HandleFunc("/video", handleIndexTemplate)
	http.HandleFunc("/help", wsStaticFiles)
	http.HandleFunc("/modhelp", wsStaticFiles)
	http.HandleFunc("/adminhelp", wsStaticFiles)

	http.HandleFunc("/", handleDefault)

	address := settings.ListenAddress
	if addr != nil && len(*addr) != 0 {
		address = *addr
	}

	// Load emotes before starting server.
	var err error
	if chat, err = newChatRoom(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// A stream key was passed on the command line.  Use it, but don't save
	// it over the stream key in the settings.json file.
	if sKey != nil && len(*sKey) != 0 {
		settings.SetTempKey(*sKey)
	}

	fmt.Println("Stream key: ", settings.GetStreamKey())
	fmt.Println("Admin password: ", settings.AdminPassword)

	go http.ListenAndServe(address, nil)
	fmt.Println("Listen and serve ", *addr)

	server.ListenAndServe()
}
