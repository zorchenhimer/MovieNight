package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/flv"
	"github.com/nareix/joy4/format/rtmp"
)

var (
	addr = flag.String("l", ":8089", "host:port of the MovieNight")
	sKey = flag.String("k", "", "Stream key, to protect your stream")
)

func init() {
	format.RegisterAll()
}

type writeFlusher struct {
	httpflusher http.Flusher
	io.Writer
}

func (self writeFlusher) Flush() error {
	self.httpflusher.Flush()
	return nil
}

func main() {
	flag.Parse()
	server := &rtmp.Server{}

	l := &sync.RWMutex{}
	type Channel struct {
		que *pubsub.Queue
	}
	channels := map[string]*Channel{}

	server.HandlePlay = func(conn *rtmp.Conn) {
		l.RLock()
		ch := channels[conn.URL.Path]
		l.RUnlock()

		if ch != nil {
			cursor := ch.que.Latest()
			avutil.CopyFile(conn, cursor)
		}
	}

	server.HandlePublish = func(conn *rtmp.Conn) {
		streams, _ := conn.Streams()

		l.Lock()
		fmt.Println("request string->", conn.URL.RequestURI())
		urlParts := strings.Split(strings.Trim(conn.URL.RequestURI(), "/"), "/")
		fmt.Println("urlParts->", urlParts)

		if len(urlParts) > 2 {
			fmt.Println("Extra garbage after stream key")
			return
		}

		if len(urlParts) != 2 {
			fmt.Println("Missing stream key")
			return
		}

		if urlParts[1] != settings.GetStreamKey() {
			fmt.Println("Due to key not match, denied stream")
			return //If key not match, deny stream
		}

		streamPath := urlParts[0]
		ch := channels[streamPath]
		if ch == nil {
			ch = &Channel{}
			ch.que = pubsub.NewQueue()
			ch.que.WriteHeader(streams)
			channels[streamPath] = ch
		} else {
			ch = nil
		}
		l.Unlock()
		if ch == nil {
			fmt.Println("Unable to start stream, channel is nil.")
			return
		}

		fmt.Println("Stream started")
		avutil.CopyPackets(ch.que, conn)
		fmt.Println("Stream finished")

		l.Lock()
		delete(channels, streamPath)
		l.Unlock()
		ch.que.Close()
	}

	// Chat websocket
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/static/js/", wsStaticFiles)
	http.HandleFunc("/static/main.wasm", wsStaticFiles)
	http.HandleFunc("/static/site.css", wsStaticFiles)
	http.HandleFunc("/emotes/", wsEmotes)
	http.HandleFunc("/favicon.ico", wsStaticFiles)
	http.HandleFunc("/chat", handleIndexTemplate)
	http.HandleFunc("/video", handleIndexTemplate)
	http.HandleFunc("/help", wsStaticFiles)
	http.HandleFunc("/modhelp", wsStaticFiles)
	http.HandleFunc("/adminhelp", wsStaticFiles)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		l.RLock()
		ch := channels[strings.Trim(r.URL.Path, "/")]
		l.RUnlock()

		if ch != nil {
			w.Header().Set("Content-Type", "video/x-flv")
			w.Header().Set("Transfer-Encoding", "chunked")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(200)
			flusher := w.(http.Flusher)
			flusher.Flush()

			muxer := flv.NewMuxerWriteFlusher(writeFlusher{httpflusher: flusher, Writer: w})
			cursor := ch.que.Latest()

			avutil.CopyFile(muxer, cursor)
		} else {
			if r.URL.Path != "/" {
				fmt.Println("[http 404] ", r.URL.Path)
				http.NotFound(w, r)
			} else {
				handleIndexTemplate(w, r)
			}
		}
	})

	address := settings.ListenAddress
	if addr != nil && len(*addr) != 0 {
		address = *addr
	}

	// Load emotes before starting server.
	if err := chat.Init(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if sKey != nil && len(*sKey) != 0 {
		settings.SetTempKey(*sKey)
	}

	fmt.Println("Stream key: ", settings.GetStreamKey())
	fmt.Println("Admin password: ", settings.AdminPassword)

	go http.ListenAndServe(address, nil)
	fmt.Println("Listen and serve ", *addr)

	server.ListenAndServe()

	// ffmpeg -re -i movie.flv -c copy -f flv rtmp://localhost/movie
	// ffmpeg -f avfoundation -i "0:0" .... -f flv rtmp://localhost/screen
	// ffplay http://localhost:8089/movie
	// ffplay http://localhost:8089/screen
}
