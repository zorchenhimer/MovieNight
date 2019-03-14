package main

import (
	"fmt"
	"time"

	"github.com/dennwc/dom/js"
	"github.com/zorchenhimer/MovieNight/common"
)

func recieve(v []js.Value) {
	if len(v) == 0 {
		fmt.Printf("No data received")
		return
	}

	fmt.Printf("Received: %s\n", v[0])
	data, err := common.DecodeData(v[0].String())
	if err != nil {
		fmt.Printf("Error decoding data: %s\n", err)
		js.Call("appendMessages", v)
		return
	}

	switch data.GetType() {
	case common.DTChat, common.DTError, common.DTEvent:
		js.Call("appendMessages", data.HTML())
	case common.DTCommand:
		dc := data.(common.DataCommand)

		switch dc.Command {
		case common.CmdPlaying:
			if dc.Arguments == nil || len(dc.Arguments) == 0 {
				js.Call("setPlaying", "", "")

			} else if len(dc.Arguments) == 1 {
				js.Call("setPlaying", dc.Arguments[0], "")

			} else if len(dc.Arguments) == 2 {
				js.Call("setPlaying", dc.Arguments[0], dc.Arguments[1])
			}
		case common.CmdRefreshPlayer:
			js.Call("initPlayer", nil)
		case common.CmdPurgeChat:
			fmt.Println("//TODO: chat purge command received.")
		case common.CmdHelp:
			js.Call("appendMesages", data.HTML())
			// TODO: open window
			//js.Call("")
		}
		return
	}
}

func send(v []js.Value) {
	if len(v) != 1 {
		fmt.Printf("expected 1 parameter, got %d", len(v))
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
