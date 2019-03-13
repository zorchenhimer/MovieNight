package main

import (
	"fmt"
	"time"

	"github.com/dennwc/dom/js"
	"github.com/zorchenhimer/MovieNight/common"
)

func log(s string) {
	js.Get("console").Call("log", s)
}

func recieve(v []js.Value) {
	if len(v) == 0 {
		fmt.Printf("No data received")
		return
	}

	fmt.Printf("Received: %s\n", v[0])
	data, err := common.DecodeData(fmt.Sprintf("%s", v[0]))
	if err != nil {
		fmt.Printf("Error decoding data: %s\n", err)
		js.Call("appendMessages", v)
		return
	}

	switch data.GetType() {
	case common.DT_CHAT, common.DT_EVENT, common.DT_ERROR:
		js.Call("appendMessages", data.HTML())
	case common.DT_COMMAND:
		dc := data.(common.DataCommand)

		switch dc.Command {
		case common.CMD_PLAYING:
			if dc.Arguments == nil || len(dc.Arguments) == 0 {
				js.Call("setPlaying", "", "")

			} else if len(dc.Arguments) == 1 {
				js.Call("setPlaying", dc.Arguments[0], "")

			} else if len(dc.Arguments) == 2 {
				js.Call("setPlaying", dc.Arguments[0], dc.Arguments[1])
			}
		case common.CMD_REFRESHPLAYER:
			js.Call("initPlayer", nil)
		case common.CMD_PURGECHAT:
			fmt.Println("//TODO: chat purge command received.")
		case common.CMD_HELP:
			js.Call("appendMesages", data.HTML())
			// TODO: open window
			//js.Call("")
		}
		return
	}
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
