package main

import (
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zorchenhimer/MovieNight/common"

	"github.com/gorilla/websocket"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format/flv"
	"github.com/nareix/joy4/format/rtmp"
)

var (
	//global variable for handling all chat traffic
	chat *ChatRoom

	// Read/Write mutex for rtmp stream
	l = &sync.RWMutex{}

	// Map of active streams
	channels = map[string]*Channel{}
)

type Channel struct {
	que *pubsub.Queue
}

type writeFlusher struct {
	httpflusher http.Flusher
	io.Writer
}

func (self writeFlusher) Flush() error {
	self.httpflusher.Flush()
	return nil
}

// Serving static files
func wsStaticFiles(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/favicon.ico":
		http.ServeFile(w, r, "./favicon.png")
		return
	case "/justchat":
		http.ServeFile(w, r, "./static/justchat.html")
		return
	case "/justvideo":
		http.ServeFile(w, r, "./static/justvideo.html")
		return
	}

	goodPath := r.URL.Path[8:len(r.URL.Path)]
	common.LogDebugf("[static] serving %q from folder ./static/\n", goodPath)

	http.ServeFile(w, r, "./static/"+goodPath)
}

func wsWasmFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	http.ServeFile(w, r, "./static/main.wasm")
}

func wsImages(w http.ResponseWriter, r *http.Request) {
	base := filepath.Base(r.URL.Path)
	common.LogDebugln("[img] ", base)
	http.ServeFile(w, r, "./static/img/"+base)
}

func wsEmotes(w http.ResponseWriter, r *http.Request) {
	emotefile := filepath.Base(r.URL.Path)
	http.ServeFile(w, r, "./static/emotes/"+emotefile)
}

// Handling the websocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, //not checking origin
}

//this is also the handler for joining to the chat
func wsHandler(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		common.LogErrorln("Error upgrading to websocket:", err)
		return
	}

	chatConn := &chatConnection{
		Conn: conn,
		// If the server is behind a reverse proxy (eg, Nginx), look
		// for this header to get the real IP address of the client.
		forwardedFor: r.Header.Get("X-Forwarded-For"),
	}

	go func() {
		var client *Client

		uid, err := chat.JoinTemp(chatConn)
		if err != nil {
			common.LogErrorf("[handler] could not do a temp join, %v\n", err)
			conn.Close()
		}

		//first message has to be the name
		// loop through name since websocket is opened once
		for client == nil {
			var data common.ClientData
			err := chatConn.ReadData(&data)
			if err != nil {
				common.LogInfof("[handler] Client closed connection: %s\n", conn.RemoteAddr().String())
				conn.Close()
				return
			}

			client, err = chat.Join(data.Message, uid)
			if err != nil {
				switch err.(type) {
				case UserFormatError, UserTakenError:
					common.LogInfof("[handler|%s] %v\n", errorName(err), err)
				case BannedUserError:
					common.LogInfof("[handler|%s] %v\n", errorName(err), err)
					// close connection since banned users shouldn't be connecting
					conn.Close()
				default:
					// for now all errors not caught need to be warned
					common.LogErrorf("[handler|uncaught] %v\n", err)
					conn.Close()
				}
			}
		}

		//then watch for incoming messages
		for {
			var data common.ClientData
			err := conn.ReadJSON(&data)
			if err != nil { //if error then assuming that the connection is closed
				client.Exit()
				return
			}
			client.NewMsg(data)
		}

	}()
}

func handleHelpTemplate(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./static/base.html", "./static/help.html")
	if err != nil {
		common.LogErrorf("Error parsing template file, %v\n", err)
		return
	}

	type Data struct {
		Title         string
		Commands      map[string]string
		ModCommands   map[string]string
		AdminCommands map[string]string
	}

	data := Data{
		Title:    "Help",
		Commands: getHelp(common.CmdlUser),
	}

	if len(r.URL.Query().Get("mod")) > 0 {
		data.ModCommands = getHelp(common.CmdlMod)
	}

	if len(r.URL.Query().Get("admin")) > 0 {
		data.AdminCommands = getHelp(common.CmdlAdmin)
	}

	err = t.Execute(w, data)
	if err != nil {
		common.LogErrorf("Error executing file, %v", err)
	}
}

func handleIndexTemplate(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./static/base.html", "./static/main.html")
	if err != nil {
		common.LogErrorf("Error parsing template file, %v\n", err)
		return
	}

	type Data struct {
		Video, Chat         bool
		MessageHistoryCount int
		Title               string
	}

	data := Data{
		Video:               true,
		Chat:                true,
		MessageHistoryCount: settings.MaxMessageCount,
		Title:               "Movie Night!",
	}

	path := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
	if path[0] == "chat" {
		data.Video = false
		data.Title += " - chat"
	} else if path[0] == "video" {
		data.Chat = false
		data.Title += " - video"
	}

	// Force browser to replace cache since file was not changed
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")

	err = t.Execute(w, data)
	if err != nil {
		common.LogErrorf("Error executing file, %v", err)
	}
}

func handlePublish(conn *rtmp.Conn) {
	streams, _ := conn.Streams()

	l.Lock()
	common.LogDebugln("request string->", conn.URL.RequestURI())
	urlParts := strings.Split(strings.Trim(conn.URL.RequestURI(), "/"), "/")
	common.LogDebugln("urlParts->", urlParts)

	if len(urlParts) > 2 {
		common.LogErrorln("Extra garbage after stream key")
		return
	}

	if len(urlParts) != 2 {
		common.LogErrorln("Missing stream key")
		return
	}

	if urlParts[1] != settings.GetStreamKey() {
		common.LogErrorln("Stream key is incorrect.  Denying stream.")
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
		common.LogErrorln("Unable to start stream, channel is nil.")
		return
	}

	common.LogInfoln("Stream started")
	avutil.CopyPackets(ch.que, conn)
	common.LogInfoln("Stream finished")

	l.Lock()
	delete(channels, streamPath)
	l.Unlock()
	ch.que.Close()
}

func handlePlay(conn *rtmp.Conn) {
	l.RLock()
	ch := channels[conn.URL.Path]
	l.RUnlock()

	if ch != nil {
		cursor := ch.que.Latest()
		avutil.CopyFile(conn, cursor)
	}
}

func handleDefault(w http.ResponseWriter, r *http.Request) {
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
			// not really an error for the server, but for the client.
			common.LogInfoln("[http 404] ", r.URL.Path)
			http.NotFound(w, r)
		} else {
			handleIndexTemplate(w, r)
		}
	}
}
