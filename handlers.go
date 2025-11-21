package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	que     *pubsub.Queue
	hlsChan *HLSChannel
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
	file := strings.TrimPrefix(r.URL.Path, "/")

	emoteDirSuffix := filepath.Base(emotesDir)
	if emoteDirSuffix == filepath.SplitList(file)[0] {
		file = strings.TrimPrefix(file, emoteDirSuffix+"/")
	}

	var body []byte
	err := filepath.WalkDir(emotesDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || err != nil || len(body) > 0 {
			return nil
		}

		if filepath.Base(path) != filepath.Base(file) {
			return nil
		}

		body, err = os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	})
	if err != nil {
		common.LogErrorf("Emote could not be read %s: %v\n", file, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if len(body) == 0 {
		common.LogErrorf("Found emote file but pulled no data: %v\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	_, err = w.Write(body)
	if err != nil {
		common.LogErrorf("Could not write emote %s to response: %v\n", file, err)
		w.WriteHeader(http.StatusNotFound)
	}
}

// Handling the websocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, //not checking origin
}

// this is also the handler for joining to the chat
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

			if data.Type == common.CdPing {
				continue
			}

			var joinData common.JoinData
			err = json.Unmarshal([]byte(data.Message), &joinData)
			if err != nil {
				common.LogInfof("[handler] Could not unmarshal websocket %d data %#v: %v\n", data.Type, data.Message, err)
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

	// Initialize HLS channel for this stream immediately
	common.LogInfof("Creating HLS channel for stream: %s\n", streamPath)
	hlsChan, err := NewHLSChannel(ch.que)
	if err != nil {
		common.LogErrorf("Failed to create HLS channel: %v\n", err)
	} else {
		ch.hlsChan = hlsChan
		err = ch.hlsChan.Start()
		if err != nil {
			common.LogErrorf("Failed to start HLS channel: %v\n", err)
			ch.hlsChan = nil
		} else {
			common.LogDebugf("HLS channel started for stream: %s\n", streamPath)
		}
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
	// Clean up HLS channel if it exists
	if ch.hlsChan != nil {
		ch.hlsChan.Stop()
	}
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

	// Debug logging for HLS troubleshooting
	userAgent := r.Header.Get("User-Agent")
	format := r.URL.Query().Get("format")
	common.LogDebugf("handleLive: path=%s, format=%s, userAgent=%s\n", r.URL.Path, format, userAgent)

	// If the user-agent is missing or invalid, reject the request
	if !ValidateUserAgent(userAgent) {
		common.LogInfof("Rejected live request with invalid User-Agent: %s\n", userAgent)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if ch != nil {
		// Detect streaming format based on device capabilities or explicit request
		streamingFormat := GetStreamingFormat(r)
		common.LogDebugf("Detected streaming format: %s\n", streamingFormat)

		// Also check if this is an HLS playlist request (for native iOS)
		if streamingFormat == "hls" || strings.HasSuffix(r.URL.Path, ".m3u8") || r.URL.Query().Get("format") == "hls" {
			common.LogDebugf("Routing to HLS handler\n")
			handleHLSStream(w, r, ch)
		} else {
			common.LogDebugf("Routing to FLV handler\n")
			handleFLVStream(w, r, ch)
		}
	} else {
		// When no stream is active, return appropriate response based on request type
		if strings.HasSuffix(r.URL.Path, ".m3u8") || r.URL.Query().Get("format") == "hls" {
			// For HLS requests, return a proper HTTP status
			common.LogInfof("HLS request for inactive stream: %s\n", r.URL.Path)
			w.Header().Set("Content-Type", GetContentTypeForFormat("hls"))
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusServiceUnavailable) // 503 - Service Unavailable is more appropriate than 204
		} else {
			// For FLV requests, use the original behavior
			common.LogInfof("FLV request for inactive stream: %s\n", r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		}
		stats.resetViewers()
	}
}

func handleFLVStream(w http.ResponseWriter, r *http.Request, ch *Channel) {
	if ch == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "video/x-flv")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
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
}

func handleHLSStream(w http.ResponseWriter, r *http.Request, ch *Channel) {
	common.LogDebugf("handleHLSStream called for path: %s\n", r.URL.Path)

	if ch == nil {
		common.LogDebugf("handleHLSStream: channel is nil\n")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Initialize HLS channel if not already done
	if ch.hlsChan == nil {
		// Check if the queue has any data before creating HLS channel
		if ch.que == nil {
			common.LogDebugf("handleHLSStream: no queue available\n")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		common.LogDebugf("handleHLSStream: initializing HLS channel\n")
		hlsChan, err := NewHLSChannelWithDeviceOptimization(ch.que, r)
		if err != nil {
			common.LogErrorf("Failed to create HLS channel: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ch.hlsChan = hlsChan
		err = ch.hlsChan.Start()
		if err != nil {
			common.LogErrorf("Failed to start HLS channel: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		common.LogDebugf("handleHLSStream: HLS channel initialized and started\n")
	}

	// Handle different HLS requests
	if IsHLSPlaylistRequest(r) {
		common.LogDebugf("handleHLSStream: routing to playlist handler\n")
		handleHLSPlaylist(w, r, ch.hlsChan)
	} else if IsHLSSegmentRequest(r) {
		common.LogDebugf("handleHLSStream: routing to segment handler\n")
		handleHLSSegment(w, r, ch.hlsChan)
	} else {
		// It is neither a playlist nor a segment request. Return 404.
		common.LogDebugf("handleHLSStream: invalid HLS request\n")
		w.WriteHeader(http.StatusNotFound)
	}
}

func handleHLSPlaylist(w http.ResponseWriter, r *http.Request, hlsChan *HLSChannel) {
	common.LogDebugf("handleHLSPlaylist called\n")

	if hlsChan == nil {
		common.LogDebugf("handleHLSPlaylist: hlsChan is nil\n")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", GetContentTypeForFormat("hls"))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	playlist := hlsChan.GetPlaylist()
	common.LogDebugf("handleHLSPlaylist: playlist length = %d\n", len(playlist))

	// Check if playlist has segments rather than just being empty string
	hasSegments := hlsChan.HasSegments()
	common.LogDebugf("handleHLSPlaylist: hasSegments = %v\n", hasSegments)

	// Track viewer for HLS - count viewers even when waiting for segments
	session, _ := sstore.Get(r, "moviesession")
	if session != nil {
		isNewViewer := hlsChan.AddViewer(session.ID)
		if isNewViewer {
			stats.addViewer(session.ID)
			common.LogInfof("[HLS] New viewer added to stats: %s\n", session.ID)
		}
	}

	if playlist == "" || !hasSegments {
		common.LogDebugf("handleHLSPlaylist: playlist is empty or has no segments\n")
		// Return 503 (Service Unavailable) for empty playlists to indicate segments are still being generated
		// This is more appropriate than 404 and allows clients to retry
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("#EXTM3U\n#EXT-X-VERSION:6\n#EXT-X-TARGETDURATION:10\n#EXT-X-MEDIA-SEQUENCE:0\n"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(playlist))
	common.LogDebugf("handleHLSPlaylist: playlist sent successfully\n")
}

func handleHLSSegment(w http.ResponseWriter, r *http.Request, hlsChan *HLSChannel) {
	if hlsChan == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Extract segment filename from path
	pathParts := strings.Split(r.URL.Path, "/")
	segmentFilename := pathParts[len(pathParts)-1]

	if !IsValidSegmentURI(segmentFilename) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Reconstruct the full URI that was used when storing the segment
	// The segment was stored with the full absolute path like "/live/segment_N.ts"
	segmentURI := r.URL.Path

	segmentData, err := hlsChan.GetSegmentByURI(segmentURI)
	if err != nil {
		common.LogErrorf("Failed to get HLS segment %s: %v\n", segmentURI, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", GetContentTypeForFormat("ts"))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// Use shorter cache time for live segments to prevent stale content issues
	// Long cache (1 year) can cause problems when service restarts with different content
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache segments for 1 hour instead of 1 year
	w.Header().Set("Content-Length", strconv.Itoa(len(segmentData)))

	w.WriteHeader(http.StatusOK)
	w.Write(segmentData)
}

func handleHLS(w http.ResponseWriter, r *http.Request) {
	// Extract stream path from URL like /hls/streamname/playlist.m3u8 or /hls/streamname/segment_N.ts
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	streamName := pathParts[1]

	l.RLock()
	ch := channels[streamName]
	l.RUnlock()

	if ch == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Initialize HLS channel if not already done
	if ch.hlsChan == nil {
		hlsChan, err := NewHLSChannelWithDeviceOptimization(ch.que, r)
		if err != nil {
			common.LogErrorf("Failed to create HLS channel: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ch.hlsChan = hlsChan
		err = ch.hlsChan.Start()
		if err != nil {
			common.LogErrorf("Failed to start HLS channel: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Handle different HLS requests
	if len(pathParts) >= 3 {
		fileName := pathParts[2]
		if strings.HasSuffix(fileName, ".m3u8") {
			handleHLSPlaylist(w, r, ch.hlsChan)
		} else if strings.HasSuffix(fileName, ".ts") {
			handleHLSSegment(w, r, ch.hlsChan)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	} else {
		// Default to playlist
		handleHLSPlaylist(w, r, ch.hlsChan)
	}
}

// handleLiveSegments handles HLS segment requests from /live/ path
func handleLiveSegments(w http.ResponseWriter, r *http.Request) {
	// Extract segment name from URL like /live/segment_N.ts
	path := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(path, "/")

	common.LogDebugf("handleLiveSegments: path=%s, pathParts=%v", path, pathParts)

	if len(pathParts) < 2 {
		common.LogDebugf("handleLiveSegments: invalid path, not enough parts")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	segmentName := pathParts[1]
	if !strings.HasSuffix(segmentName, ".ts") {
		common.LogDebugf("handleLiveSegments: not a .ts file: %s", segmentName)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Use "live" as the default stream name for /live/ requests
	streamName := "live"

	l.RLock()
	ch := channels[streamName]
	l.RUnlock()

	if ch == nil {
		common.LogDebugf("handleLiveSegments: no channel found for stream: %s", streamName)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Make sure we have an HLS channel
	if ch.hlsChan == nil {
		common.LogDebugf("handleLiveSegments: no HLS channel found for stream: %s", streamName)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	common.LogDebugf("handleLiveSegments: requesting segment %s from HLS channel", segmentName)

	// Handle the segment request
	handleHLSSegment(w, r, ch.hlsChan)
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

func wrapAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if settings.RoomAccess != AccessOpen {
			if !checkRoomAccess(w, r) {
				common.LogDebugln("Denied access")
				return
			}
			common.LogDebugln("Granted access")
		}
		next.ServeHTTP(w, r)
	})
}
