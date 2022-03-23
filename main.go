package main

import (
	"context"
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/gorilla/sessions"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/zorchenhimer/MovieNight/common"
)

var stats = newStreamStats()

func setupSettings(adminPass string, confFile string) error {
	var err error
	settings, err = LoadSettings(confFile)
	if err != nil {
		return fmt.Errorf("unable to load settings: %w", err)
	}
	if len(settings.StreamKey) == 0 {
		return fmt.Errorf("missing stream key is settings.json")
	}

	if adminPass != "" {
		fmt.Println("Password provided at runtime; ignoring password in set in settings.")
		settings.AdminPassword = adminPass
	}

	sstore = sessions.NewCookieStore([]byte(settings.SessionKey))
	sstore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 24, // one day
		SameSite: http.SameSiteStrictMode,
	}

	return nil
}

//go:embed static/*.html static/css static/img static/js
var staticFs embed.FS

func writeStaticFiles(fileDir, name string) error {
	items, err := staticFs.ReadDir(name)
	if err != nil {
		return fmt.Errorf("could not read staticFs directory %#v: %w", name, err)
	}

	for _, item := range items {
		fsPath := path.Join(name, item.Name())
		filePath := strings.Replace(fsPath, "static", fileDir, 1)

		_, err := os.Open(filePath)
		notExist := errors.Is(err, os.ErrNotExist)

		if item.IsDir() {
			if notExist {
				fmt.Printf("creating dir %q\n", filePath)

				err = os.MkdirAll(filePath, os.ModeDir)
				if err != nil {
					return fmt.Errorf("could not make missing directory: %w", err)
				}
			}

			err = writeStaticFiles(fileDir, fsPath)
			if err != nil {
				return err
			}
		} else if notExist {
			fmt.Printf("creating file %q\n", filePath)

			staticFile, err := staticFs.Open(fsPath)
			if err != nil {
				return fmt.Errorf("could not open embeded file %q: %w", fsPath, err)
			}

			var staticData []byte
			_, err = staticFile.Read(staticData)
			if err != nil {
				return fmt.Errorf("could not read embeded file %q: %w", fsPath, err)
			}

			err = os.WriteFile(filePath, staticData, 0644)
			if err != nil {
				return fmt.Errorf("could not write static data to file %q: %w", filePath, err)
			}
		}
	}
	return nil
}

func main() {
	var err error
	var args struct {
		Addr       string `arg:"-l,--addr" help:"host:port of the HTTP server"`
		RtmpAddr   string `arg:"-r,--rtmp" help:"host:port of the RTMP server"`
		StreamKey  string `arg:"-k,--key" help:"Stream key, to protect your stream"`
		AdminPass  string `arg:"-a,--admin" help:"Set admin password.  Overrides configuration in settings.json.  This will not write the password to settings.json."`
		PullEmotes bool   `arg:"-e,--pull-emotes" help:"Pull emotes"`
		ConfigFile string `arg:"-f,--config" default:"./settings.json" help:"URI of the conf file"`
		StaticDir  string `arg:"-s,--static" default:"static" help:"The directory MovieNight looks for the static dir"`
	}
	arg.MustParse(&args)

	format.RegisterAll()

	if err := setupSettings(args.AdminPass, args.ConfigFile); err != nil {
		log.Fatalf("Error loading settings: %v\n", err)
	}

	err = writeStaticFiles(args.StaticDir, ".")
	if err != nil {
		common.LogErrorf("Error writing static files: %v\n", err)
		os.Exit(1)
	}

	if args.PullEmotes {
		common.LogInfoln("Pulling emotes")
		err := getEmotes(settings.ApprovedEmotes)
		if err != nil {
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
	chat, err = newChatRoom()
	if err != nil {
		common.LogErrorln(err)
		os.Exit(1)
	}

	if args.Addr == "" {
		args.Addr = settings.ListenAddress
	}

	if args.RtmpAddr == "" {
		args.RtmpAddr = settings.RtmpListenAddress
	}

	// A stream key was passed on the command line.  Use it, but don't save
	// it over the stream key in the settings.json file.
	if args.StreamKey != "" {
		settings.SetTempKey(args.StreamKey)
	}

	common.LogInfoln("Stream key: ", settings.GetStreamKey())
	common.LogInfoln("Admin password: ", settings.AdminPassword)
	common.LogInfoln("HTTP server listening on: ", args.Addr)
	common.LogInfoln("RTMP server listening on: ", args.RtmpAddr)
	common.LogInfoln("RoomAccess: ", settings.RoomAccess)
	common.LogInfoln("RoomAccessPin: ", settings.RoomAccessPin)

	rtmpServer := &rtmp.Server{
		HandlePlay:    handlePlay,
		HandlePublish: handlePublish,
		Addr:          args.RtmpAddr,
	}

	router := http.NewServeMux()

	router.HandleFunc("/ws", wsHandler) // Chat websocket
	router.HandleFunc("/static/js/", wsStaticFiles)
	router.HandleFunc("/static/css/", wsStaticFiles)
	router.HandleFunc("/static/img/", wsImages)
	router.HandleFunc("/emotes/", wsEmotes)
	router.HandleFunc("/favicon.ico", wsStaticFiles)
	router.HandleFunc("/chat", handleIndexTemplate)
	router.HandleFunc("/video", handleIndexTemplate)
	router.HandleFunc("/help", handleHelpTemplate)
	router.HandleFunc("/emotes", handleEmoteTemplate)

	router.HandleFunc("/live", handleLive)
	router.HandleFunc("/", handleDefault)

	httpServer := &http.Server{
		Addr:    args.Addr,
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
