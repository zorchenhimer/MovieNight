package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
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
		fmt.Println("request key->", conn.URL.Query().Get("key"))
		streamKey := conn.URL.Query().Get("key")
		if streamKey != *sKey {
			fmt.Println("Due to key not match, denied stream")
			return //If key not match, deny stream
		}
		ch := channels[conn.URL.Path]
		if ch == nil {
			ch = &Channel{}
			ch.que = pubsub.NewQueue()
			ch.que.WriteHeader(streams)
			channels[conn.URL.Path] = ch
		} else {
			ch = nil
		}
		l.Unlock()
		if ch == nil {
			return
		}

		avutil.CopyPackets(ch.que, conn)

		l.Lock()
		delete(channels, conn.URL.Path)
		l.Unlock()
		ch.que.Close()
	}

	// Chat websocket
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/static/js/", wsStaticFiles)
	http.HandleFunc("/static/site.css", wsStaticFiles)
	http.HandleFunc("/emotes/", wsEmotes)
	http.HandleFunc("/favicon.ico", wsStaticFiles)
	http.HandleFunc("/justchat", wsStaticFiles)
	http.HandleFunc("/justvideo", wsStaticFiles)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		l.RLock()
		ch := channels[r.URL.Path]
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
				http.ServeFile(w, r, "./static/index.html")
			}
		}
	})

	go http.ListenAndServe(*addr, nil)
	fmt.Println("Listen and serve ", *addr)
	if err := chat.Init(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	server.ListenAndServe()

	// ffmpeg -re -i movie.flv -c copy -f flv rtmp://localhost/movie
	// ffmpeg -f avfoundation -i "0:0" .... -f flv rtmp://localhost/screen
	// ffplay http://localhost:8089/movie
	// ffplay http://localhost:8089/screen
}
