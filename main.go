package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/rtmp"
)

var (
	addr  string
	sKey  string
	stats streamStats
)

func init() {
	format.RegisterAll()

	flag.StringVar(&addr, "l", ":8089", "host:port of the MovieNight")
	flag.StringVar(&sKey, "k", "", "Stream key, to protect your stream")

	stats = newStreamStats()
}

func main() {
	flag.Parse()

	exit := make(chan bool)
	go handleInterrupt(exit)

	// Load emotes before starting server.
	var err error
	if chat, err = newChatRoom(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if addr != "" {
		addr = settings.ListenAddress
	}

	// A stream key was passed on the command line.  Use it, but don't save
	// it over the stream key in the settings.json file.
	if sKey != "" {
		settings.SetTempKey(sKey)
	}

	fmt.Println("Stream key: ", settings.GetStreamKey())
	fmt.Println("Admin password: ", settings.AdminPassword)
	fmt.Println("Listen and serve ", addr)

	go startServer()
	go startRmtpServer()

	<-exit
}

func startRmtpServer() {
	server := &rtmp.Server{
		HandlePlay:    handlePlay,
		HandlePublish: handlePublish,
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("Error trying to start server: %v\n", err)
	}
}

func startServer() {
	// Chat websocket
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/static/js/", wsStaticFiles)
	http.HandleFunc("/static/css/", wsStaticFiles)
	http.HandleFunc("/static/img/", wsImages)
	http.HandleFunc("/static/main.wasm", wsWasmFile)
	http.HandleFunc("/emotes/", wsEmotes)
	http.HandleFunc("/favicon.ico", wsStaticFiles)
	http.HandleFunc("/chat", handleIndexTemplate)
	http.HandleFunc("/video", handleIndexTemplate)
	http.HandleFunc("/help", handleHelpTemplate)

	http.HandleFunc("/", handleDefault)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Printf("Error trying to start rmtp server: %v\n", err)
	}
}

func handleInterrupt(exit chan bool) {
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	<-ch
	fmt.Println("Closing server")
	if settings.StreamStats {
		stats.Print()
	}
	exit <- true
}
