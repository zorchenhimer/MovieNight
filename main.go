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
	pullEmotes bool
	addr       string
	rtmpAddr   string
	sKey       string
	stats      = newStreamStats()
	sAdminPass string
	confFile   string
)

func setupSettings() error {
	var err error
	settings, err = LoadSettings(confFile)
	if err != nil {
		return fmt.Errorf("unable to load settings: %s", err)
	}
	if len(settings.StreamKey) == 0 {
		return fmt.Errorf("missing stream key is settings.json")
	}

	if sAdminPass != "" {
		fmt.Println("Password provided at runtime; ignoring password in set in settings.")
		settings.AdminPassword = sAdminPass
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
	flag.StringVar(&addr, "l", "", "host:port of the HTTP server")
	flag.StringVar(&rtmpAddr, "r", "", "host:port of the RTMP server")
	flag.StringVar(&sKey, "k", "", "Stream key, to protect your stream")
	flag.StringVar(&sAdminPass, "a", "", "Set admin password.  Overrides configuration in settings.json.  This will not write the password to settings.json.")
	flag.BoolVar(&pullEmotes, "e", false, "Pull emotes")
	flag.StringVar(&confFile, "f", "./settings.json", "URI of the conf file")
	flag.Parse()

	format.RegisterAll()

	if err := setupSettings(); err != nil {
		fmt.Printf("Error loading settings: %v\n", err)
		os.Exit(1)
	}

	if pullEmotes {
		common.LogInfoln("Pulling emotes")
		err := getEmotes(settings.ApprovedEmotes)
		if err != nil {
			common.LogErrorf("Error downloading emotes: %+v\n", err)
			common.LogErrorf("Error downloading emotes: %v\n", err)
			os.Exit(1)
		}
	}

	if err := common.InitTemplates(); err != nil {
		common.LogErrorln(err)
		os.Exit(1)
	}

	exit := make(chan bool)
	go handleInterrupt(exit)

	// Load emotes before starting server.
	var err error
	chat, err = newChatRoom()
	if err != nil {
		common.LogErrorln(err)
		os.Exit(1)
	}

	if addr == "" {
		addr = settings.ListenAddress
	}

	if rtmpAddr == "" {
		rtmpAddr = settings.RtmpListenAddress
	}

	// A stream key was passed on the command line.  Use it, but don't save
	// it over the stream key in the settings.json file.
	if sKey != "" {
		settings.SetTempKey(sKey)
	}

	common.LogInfoln("Stream key: ", settings.GetStreamKey())
	common.LogInfoln("Admin password: ", settings.AdminPassword)
	common.LogInfoln("HTTP server listening on: ", addr)
	common.LogInfoln("RTMP server listening on: ", rtmpAddr)
	common.LogInfoln("RoomAccess: ", settings.RoomAccess)
	common.LogInfoln("RoomAccessPin: ", settings.RoomAccessPin)

	rtmpServer := &rtmp.Server{
		HandlePlay:    handlePlay,
		HandlePublish: handlePublish,
		Addr:          rtmpAddr,
	}

	router := http.NewServeMux()

	router.HandleFunc("/ws", wsHandler) // Chat websocket
	router.HandleFunc("/static/js/", wsStaticFiles)
	router.HandleFunc("/static/css/", wsStaticFiles)
	router.HandleFunc("/static/img/", wsImages)
	router.HandleFunc("/static/main.wasm", wsWasmFile)
	router.HandleFunc("/emotes/", wsEmotes)
	router.HandleFunc("/favicon.ico", wsStaticFiles)
	router.HandleFunc("/chat", handleIndexTemplate)
	router.HandleFunc("/video", handleIndexTemplate)
	router.HandleFunc("/help", handleHelpTemplate)
	router.HandleFunc("/emotes", handleEmoteTemplate)

	router.HandleFunc("/live", handleLive)
	router.HandleFunc("/", handleDefault)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// RTMP Server
	go func() {
		err := rtmpServer.ListenAndServe()
		if err != nil {
			// If the server cannot start, don't pretend we can continue.
			panic("Error trying to start rtmp server: " + err.Error())
		}
	}()

	// HTTP Server
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			// If the server cannot start, don't pretend we can continue.
			panic("Error trying to start chat/http server: " + err.Error())
		}
	}()

	<-exit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		panic("Gracefull HTTP server shutdown failed: " + err.Error())
	}

	// I don't think the RTMP server can be shutdown cleanly.  Apparently the author
	// of joy4 want's everyone to use joy5, but that one doesn't seem to allow clean
	// shutdowns either? Idk, the documentation on joy4 and joy5 are non-existent.
}

func handleInterrupt(exit chan bool) {
	ch := make(chan os.Signal, 5)
	signal.Notify(ch, os.Interrupt)
	<-ch
	common.LogInfoln("Closing server")
	if settings.StreamStats {
		stats.Print()
	}
	exit <- true
}
