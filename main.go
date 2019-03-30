package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/gorilla/sessions"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/zorchenhimer/MovieNight/common"
)

var (
	addr  string
	sKey  string
	stats = newStreamStats()
)

func setupSettings() error {
	var err error
	settings, err = LoadSettings("settings.json")
	if err != nil {
		return fmt.Errorf("Unable to load settings: %s", err)
	}
	if len(settings.StreamKey) == 0 {
		return fmt.Errorf("Missing stream key is settings.json")
	}

	sstore = sessions.NewCookieStore([]byte(settings.SessionKey))
	sstore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 24, // one day
		SameSite: http.SameSiteStrictMode,
	}

	return nil
}

func main() {
	flag.StringVar(&addr, "l", ":8089", "host:port of the MovieNight")
	flag.StringVar(&sKey, "k", "", "Stream key, to protect your stream")
	flag.Parse()

	format.RegisterAll()

	if err := setupSettings(); err != nil {
		fmt.Printf("Error loading settings: %v\n", err)
		os.Exit(1)
	}

	exit := make(chan bool)
	go handleInterrupt(exit)

	// Load emotes before starting server.
	var err error
	if chat, err = newChatRoom(); err != nil {
		common.LogErrorln(err)
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

	common.LogInfoln("Stream key: ", settings.GetStreamKey())
	common.LogInfoln("Admin password: ", settings.AdminPassword)
	common.LogInfoln("Listen and serve ", addr)
	common.LogInfoln("RoomAccess: ", settings.RoomAccess)
	common.LogInfoln("RoomAccessPin: ", settings.RoomAccessPin)

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
		// If the server cannot start, don't pretend we can continue.
		panic("Error trying to start rtmp server: " + err.Error())
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
	http.HandleFunc("/pin", handlePin)

	http.HandleFunc("/", handleDefault)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		// If the server cannot start, don't pretend we can continue.
		panic("Error trying to start chat/http server: " + err.Error())
	}
}

func handleInterrupt(exit chan bool) {
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	<-ch
	common.LogInfoln("Closing server")
	if settings.StreamStats {
		stats.Print()
	}
	exit <- true
}
