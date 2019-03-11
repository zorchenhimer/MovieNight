package main

import (
	"fmt"
	//"net"
	"net/http"
	"path/filepath"

	"github.com/gorilla/websocket"
)

//global variable for handling all chat traffic
var chat ChatRoom

// Serving static files
func wsStaticFiles(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/favicon.ico":
		http.ServeFile(w, r, "./static/favicon.png")
		return
	case "/justchat":
		http.ServeFile(w, r, "./static/justchat.html")
		return
	case "/justvideo":
		http.ServeFile(w, r, "./static/justvideo.html")
		return

	// TODO: use a template for this, lol.
	case "/help":
		w.Write([]byte(helpPage(false, false)))
		return
	case "/modhelp":
		w.Write([]byte(helpPage(true, false)))
		return
	case "/adminhelp":
		w.Write([]byte(helpPage(true, true)))
		return
	}

	goodPath := r.URL.Path[8:len(r.URL.Path)]
	fmt.Printf("[static] serving %q from folder ./static/\n", goodPath)

	http.ServeFile(w, r, "./static/"+goodPath)
}

func wsEmotes(w http.ResponseWriter, r *http.Request) {
	emotefile := filepath.Base(r.URL.Path)
	//fmt.Printf("serving emote: %s\n", emotefile)
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
		fmt.Println("Error upgrading to websocket:", err)
		return
	}
	go func() {
		//first message has to be the name
		_, msg, err := conn.ReadMessage()
		client := chat.Join(string(msg), conn)
		if client == nil || err != nil {
			fmt.Printf("[handler] Client closed connection: %s\n", conn.RemoteAddr().String())
			conn.Close() //closing connection to indicate failed Join
			return
		}

		//then watch for incoming messages
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil { //if error then assuming that the connection is closed
				client.Exit()
				return
			}
			client.NewMsg(string(msg))
		}

	}()
}
