package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/gorilla/sessions"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/zorchenhimer/MovieNight/common"
)

//go:embed static/*.html static/css static/img static/js
var staticFS embed.FS

var stats = newStreamStats()

func setupSettings(args args) error {
	var err error
	settings, err = LoadSettings(args)
	if err != nil {
		return fmt.Errorf("unable to load settings: %w", err)
	}

	return nil
}

func setupCookieStore() {
	sstore = sessions.NewCookieStore([]byte(settings.SessionKey))
	sstore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 24, // one day
		SameSite: http.SameSiteStrictMode,
	}
}

type args struct {
	Addr        string `arg:"-l,--addr,env:MN_ADDR" help:"host:port of the HTTP server"`
	RtmpAddr    string `arg:"-r,--rtmp,env:MN_RTMP" help:"host:port of the RTMP server"`
	StreamKey   string `arg:"-k,--key,env:MN_STREAM_KEY" default:"" help:"Stream key, to protect your stream"`
	AdminPass   string `arg:"-a,--admin,env:MN_ADMIN_PASS" default:"" help:"Set admin password. Overrides configuration in settings.json. This will not write the password to settings.json."`
	ConfigFile  string `arg:"-f,--config,env:MN_CONFIG" default:"settings.json" help:"URI of the conf file"`
	StaticDir   string `arg:"-s,--static,env:MN_STATIC" default:"" help:"Directory to read static files from by default"` // default static dir should be `static` I guess. Zorglube
	EmotesDir   string `arg:"-e,--emotes,env:MN_EMOTES" default:"emotes" help:"Directory to read emotes. By default it uses the executable directory"`
	WriteStatic bool   `arg:"--write-static,env:MN_WRITE_STATIC" default:"false" help:"write static files to the static dir"`
}

func main() {
	var args args
	arg.MustParse(&args)
	run(args)
}

func run(args args) {
	var err error
	start := time.Now()

	if err := setupSettings(args); err != nil {
		log.Fatalf("Error loading settings: %v\n", err)
	}

	setupCookieStore()
	staticFsys := settings.GetStaticFsys()

	if settings.GetWriteStatic() {
		count, err := staticFsys.WriteFiles(".")
		fmt.Printf("%d files were writen to disk\n", count)
		if err != nil {
			log.Fatalf("Error writing files to static dir %q: %v\n", settings.GetStaticDir(), err)
		}
	}

	format.RegisterAll()

	if err := common.InitTemplates(staticFsys); err != nil {
		common.LogErrorln(err)
		os.Exit(1)
	}

	exit := make(chan bool)
	go handleInterrupt(exit)

	// Load emotes before starting server.
	chat, err = newChatRoom()
	if err != nil {
		common.LogErrorln(err)
		os.Exit(1)
	}

	common.LogInfoln("Stream key: ", settings.GetStreamKey())
	common.LogInfoln("Admin password: ", settings.GetAdminPassword())
	common.LogInfoln("HTTP server listening on: ", settings.GetAddr())
	common.LogInfoln("RTMP server listening on: ", settings.GetRtmpAddr())
	common.LogInfoln("RoomAccess: ", settings.GetRoomAccess())
	common.LogInfoln("RoomAccessPin: ", settings.GetRoomAccessPin())

	rtmpServer := &rtmp.Server{
		HandlePlay:    handlePlay,
		HandlePublish: handlePublish,
		Addr:          settings.GetRtmpAddr(),
	}

	router := http.NewServeMux()

	router.Handle("/static/", http.FileServer(http.FS(staticFsys)))
	router.HandleFunc("/emotes/", wsEmotes)

	router.HandleFunc("/ws", wrapAuth(wsHandler)) // Chat websocket
	router.HandleFunc("/chat", wrapAuth(handleIndexTemplate))
	router.HandleFunc("/video", wrapAuth(handleIndexTemplate))
	router.HandleFunc("/help", wrapAuth(handleHelpTemplate))
	router.HandleFunc("/emotes", wrapAuth(handleEmoteTemplate))

	router.HandleFunc("/live", wrapAuth(handleLive))
	router.HandleFunc("/", wrapAuth(handleDefault))

	httpServer := &http.Server{
		Addr:    settings.GetAddr(),
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

	common.LogInfof("Startup took %v\n", time.Since(start))

	<-exit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
