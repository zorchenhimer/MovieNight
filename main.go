package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/sessions"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/zorchenhimer/MovieNight/common"
)

var (
	addr       string
	sKey       string
	stats      = newStreamStats()
	chatServer *http.Server
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

	server := &rtmp.Server{
		HandlePlay:    handlePlay,
		HandlePublish: handlePublish,
	}

	// Define this here so we can set some timeouts and things.
	chatServer := &http.Server{
		Addr:           addr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	chatServer.RegisterOnShutdown(func() { chat.Shutdown() })
	// rtmp.Server does not implement .RegisterOnShutdown()
	//server.RegisterOnShutdown(func() { common.LogDebugln("server shutdown callback called.") })

	// These have been moved back to annon functitons so I could use
	// `server`, `chatServer`, and `exit` in them without needing to
	// pass them as parameters.

	// Signal handler
	exit := make(chan bool)
	go func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt)
		<-ch
		common.LogInfoln("Closing server")
		if settings.StreamStats {
			stats.Print()
		}

		if err := chatServer.Shutdown(context.Background()); err != nil {
			common.LogErrorf("Error shutting down chat server: %v", err)
		}

		common.LogInfoln("Shutdown() sent.  Sending exit.")
		exit <- true
	}()

	// Chat and HTTP server
	go func() {
		// Use a ServeMux here instead of the default, global,
		// http handler.  It's a good idea when we're starting more
		// than one server.
		mux := http.NewServeMux()
		mux.HandleFunc("/ws", wsHandler)
		mux.HandleFunc("/static/js/", wsStaticFiles)
		mux.HandleFunc("/static/css/", wsStaticFiles)
		mux.HandleFunc("/static/img/", wsImages)
		mux.HandleFunc("/static/main.wasm", wsWasmFile)
		mux.HandleFunc("/emotes/", wsEmotes)
		mux.HandleFunc("/favicon.ico", wsStaticFiles)
		mux.HandleFunc("/chat", handleIndexTemplate)
		mux.HandleFunc("/video", handleIndexTemplate)
		mux.HandleFunc("/help", handleHelpTemplate)
		mux.HandleFunc("/pin", handlePin)

		mux.HandleFunc("/", handleDefault)

		chatServer.Handler = mux
		err := chatServer.ListenAndServe()
		// http.ErrServerClosed is returned when server.Shuddown()
		// is called.
		if err != http.ErrServerClosed {
			// If the server cannot start, don't pretend we can continue.
			panic("Error trying to start chat/http server: " + err.Error())
		}
		common.LogDebugln("ChatServer closed.")
	}()

	// RTMP server
	go func() {
		err := server.ListenAndServe()
		// http.ErrServerClosed is returned when server.Shuddown()
		// is called.
		if err != http.ErrServerClosed {
			// If the server cannot start, don't pretend we can continue.
			panic("Error trying to start rtmp server: " + err.Error())
		}
		common.LogDebugln("RTMP server closed.")
	}()

	<-exit
}
