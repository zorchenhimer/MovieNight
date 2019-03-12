package main

import (
	"fmt"
	"time"

	"github.com/dennwc/dom/js"
)

func log(s string) {
	js.Get("console").Call("log", s)
}

func recieve(v []js.Value) {
	js.Call("appendMessages", v)
}

func send(v []js.Value) {
	if len(v) != 1 {
		log(fmt.Sprintf("expected 1 parameter, got %d", len(v)))
		return
	}
	js.Call("websocketSend", v)
}

func main() {
	js.Set("recieveMessage", js.CallbackOf(recieve))
	js.Set("sendMessage", js.CallbackOf(send))

	// This is needed so the goroutine does not end
	for {
		time.Sleep(time.Minute)
	}
}
