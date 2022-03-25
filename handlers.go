package main

import (
	"encoding/json"
	"io"
	"net/http"
	"path"
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

func (w writeFlusher) Flush() error {
	w.httpflusher.Flush()
	return nil
}

func wsEmotes(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, path.Join("static/", r.URL.Path))
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

	common.LogDebugln("Connection has been upgraded to websocket")

	chatConn := &chatConnection{
		Conn: conn,
		// If the server is behind a reverse proxy (eg, Nginx), look
		// for this header to get the real IP address of the client.
		forwardedFor: common.ExtractForwarded(r),
	}

	go func() {
		var client *Client

		// Get the client object
		for client == nil {
			var data common.ClientData
			err := chatConn.ReadData(&data)
			if err != nil {
				common.LogInfof("[handler] Client closed connection: %s: %v\n",
					conn.RemoteAddr().String(), err)
				conn.Close()
				return
			}

			var joinData common.JoinData
			err = json.Unmarshal([]byte(data.Message), &joinData)
			if err != nil {
				common.LogInfof("[handler] Could not unmarshal join data %#v: %v\n", data.Message, err)
				continue
			}

			client, err = chat.Join(chatConn, joinData)
			if err != nil {
				switch err.(type) { //nolint:errorlint
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

		// Handle incomming messages
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

// returns if it's OK to proceed
func checkRoomAccess(w http.ResponseWriter, r *http.Request) bool {
	session, err := sstore.Get(r, "moviesession")
	if err != nil {
		// Don't return as server error here, just make a new session.
		common.LogErrorf("Unable to get session for client %s: %v\n", r.RemoteAddr, err)
	}

	if settings.RoomAccess == AccessPin {
		pin := session.Values["pin"]
		// No pin found in session
		if pin == nil || len(pin.(string)) == 0 {
			if r.Method == "POST" {
				// Check for correct pin
				err = r.ParseForm()
				if err != nil {
					common.LogErrorf("Error parsing form")
					http.Error(w, "Unable to get session data", http.StatusInternalServerError)
				}

				postPin := strings.TrimSpace(r.Form.Get("txtInput"))
				common.LogDebugf("Received pin: %s\n", postPin)
				if postPin == settings.RoomAccessPin {
					// Pin is correct.  Save it to session and return true.
					session.Values["pin"] = settings.RoomAccessPin
					err = session.Save(r, w)
					if err != nil {
						common.LogErrorf("Could not save pin cookie: %v\n", err)
						return false
					}
					return true
				}
				// Pin is incorrect.
				handlePinTemplate(w, r, "Incorrect PIN")
				return false
			} else {
				qpin := r.URL.Query().Get("pin")
				if qpin != "" && qpin == settings.RoomAccessPin {
					// Pin is correct.  Save it to session and return true.
					session.Values["pin"] = settings.RoomAccessPin
					err = session.Save(r, w)
					if err != nil {
						common.LogErrorf("Could not save pin cookie: %v\n", err)
						return false
					}
					return true
				}
			}
			// nope.  display pin entry and return
			handlePinTemplate(w, r, "")
			return false
		}

		// Pin found in session, but it has changed since last time.
		if pin.(string) != settings.RoomAccessPin {
			// Clear out the old pin.
			session.Values["pin"] = nil
			err = session.Save(r, w)
			if err != nil {
				common.LogErrorf("Could not clear pin cookie: %v\n", err)
			}

			// Prompt for new one.
			handlePinTemplate(w, r, "Pin has changed.  Enter new PIN.")
			return false
		}

		// Correct pin found in session
		return true
	}

	// TODO: this.
	if settings.RoomAccess == AccessRequest {
		http.Error(w, "Requesting access not implemented yet", http.StatusNotImplemented)
		return false
	}

	// Room is open.
	return true
}

func handlePinTemplate(w http.ResponseWriter, r *http.Request, errorMessage string) {
	type Data struct {
		Title      string
		SubmitText string
		Notice     string
	}

	if errorMessage == "" {
		errorMessage = "Please enter the PIN"
	}

	data := Data{
		Title:      "Enter Pin",
		SubmitText: "Submit Pin",
		Notice:     errorMessage,
	}

	err := common.ExecuteServerTemplate(w, "pin", data)
	if err != nil {
		common.LogErrorf("Error executing file, %v", err)
	}
}

func handleHelpTemplate(w http.ResponseWriter, r *http.Request) {
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

	err := common.ExecuteServerTemplate(w, "help", data)
	if err != nil {
		common.LogErrorf("Error executing file, %v", err)
	}
}

func handleEmoteTemplate(w http.ResponseWriter, r *http.Request) {
	type Data struct {
		Title  string
		Emotes map[string]string
	}

	data := Data{
		Title:  "Available Emotes",
		Emotes: common.Emotes,
	}

	common.LogDebugf("Emotes Data: %s", data)
	err := common.ExecuteServerTemplate(w, "emotes", data)
	if err != nil {
		common.LogErrorf("Error executing file, %v", err)
	}
}

func handleIndexTemplate(w http.ResponseWriter, r *http.Request) {
	if settings.RoomAccess != AccessOpen {
		if !checkRoomAccess(w, r) {
			common.LogDebugln("Denied access")
			return
		}
		common.LogDebugln("Granted access")
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
		Title:               settings.PageTitle,
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
	if settings.NoCache {
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	}

	err := common.ExecuteServerTemplate(w, "main", data)
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
		l.Unlock()
		conn.Close()
		return
	}

	if len(urlParts) != 2 {
		common.LogErrorln("Missing stream key")
		l.Unlock()
		conn.Close()
		return
	}

	if urlParts[1] != settings.GetStreamKey() {
		common.LogErrorln("Stream key is incorrect.  Denying stream.")
		l.Unlock()
		conn.Close()
		return //If key not match, deny stream
	}

	streamPath := urlParts[0]
	_, exists := channels[streamPath]
	if exists {
		common.LogErrorln("Stream already running.  Denying publish.")
		conn.Close()
		l.Unlock()
		return
	}

	ch := &Channel{}
	ch.que = pubsub.NewQueue()
	err := ch.que.WriteHeader(streams)
	if err != nil {
		common.LogErrorf("Could not write header to streams: %v\n", err)
	}
	channels[streamPath] = ch
	l.Unlock()

	stats.startStream()

	common.LogInfoln("Stream started")
	err = avutil.CopyPackets(ch.que, conn)
	if err != nil {
		common.LogErrorf("Could not copy packets to connections: %v\n", err)
	}
	common.LogInfoln("Stream finished")

	stats.endStream()

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
		err := avutil.CopyFile(conn, cursor)
		if err != nil {
			common.LogErrorf("Could not copy video to connection: %v\n", err)
		}
	}
}

func handleLive(w http.ResponseWriter, r *http.Request) {
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

		session, _ := sstore.Get(r, "moviesession")
		stats.addViewer(session.ID)
		err := avutil.CopyFile(muxer, cursor)
		if err != nil {
			common.LogErrorf("Could not copy video to connection: %v\n", err)
		}
		stats.removeViewer(session.ID)
	} else {
		// Maybe HTTP_204 is better than HTTP_404
		w.WriteHeader(http.StatusNoContent)
		stats.resetViewers()
	}
}

func handleDefault(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		// not really an error for the server, but for the client.
		common.LogInfoln("[http 404] ", r.URL.Path)
		http.NotFound(w, r)
	} else {
		handleIndexTemplate(w, r)
	}
}
