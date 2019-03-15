package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dennwc/dom/js"
	"github.com/zorchenhimer/MovieNight/common"
)

var names []string

func recieve(v []js.Value) {
	if len(v) == 0 {
		fmt.Println("No data received")
		return
	}

	chat, err := common.DecodeData(v[0].String())
	if err != nil {
		fmt.Printf("Error decoding data: %s\n", err)
		js.Call("appendMessages", v)
		return
	}

	data, err := chat.GetData()
	if err != nil {
		fmt.Printf("Error parsing DataInterface: %v\n", err)
		js.Call("appendMessages", v)
		return
	}

	switch chat.Type {
	case common.DTHidden:
		h := data.(common.HiddenMessage)
		switch h.Type {
		case common.CdUsers:
			names = nil
			for _, i := range h.Data.([]interface{}) {
				names = append(names, i.(string))
			}
		}
	case common.DTEvent:
		d := data.(common.DataEvent)
		if d.Event == common.EvJoin {
			websocketSend("", common.CdUsers)
		}
		// on join or leave, update list of possible user names
		fallthrough
	case common.DTChat, common.DTError:
		js.Call("appendMessages", data.HTML())
	case common.DTCommand:
		d := data.(common.DataCommand)

		switch d.Command {
		case common.CmdPlaying:
			if d.Arguments == nil || len(d.Arguments) == 0 {
				js.Call("setPlaying", "", "")

			} else if len(d.Arguments) == 1 {
				js.Call("setPlaying", d.Arguments[0], "")

			} else if len(d.Arguments) == 2 {
				js.Call("setPlaying", d.Arguments[0], d.Arguments[1])
			}
		case common.CmdRefreshPlayer:
			js.Call("initPlayer", nil)
		case common.CmdPurgeChat:
			fmt.Println("//TODO: chat purge command received.")
		case common.CmdHelp:
			js.Call("appendMessages", data.HTML())
			// TODO: open window
		}
	}
}

func websocketSend(msg string, dataType common.ClientDataType) error {
	data, err := json.Marshal(common.ClientData{
		Type:    dataType,
		Message: msg,
	})
	if err != nil {
		return fmt.Errorf("could not marshal data: %v", err)
	}

	js.Call("websocketSend", string(data))
	return nil
}

func send(this js.Value, v []js.Value) interface{} {
	if len(v) != 1 {
		showSendError(fmt.Errorf("expected 1 parameter, got %d", len(v)))
		return false
	}

	err := websocketSend(v[0].String(), common.CdMessage)
	if err != nil {
		showSendError(err)
		return false
	}
	return true
}

func showSendError(err error) {
	if err != nil {
		fmt.Printf("Could not send: %v\n", err)
		js.Call("appendMessages", `<div><span style="color: red;">Could not send message</span></div>`)
	}
}

func main() {
	js.Set("recieveMessage", js.CallbackOf(recieve))
	js.Set("sendMessage", js.FuncOf(send))
	js.Set("getNames", js.FuncOf(func(_ js.Value, v []js.Value) interface{} {
		return strings.Join(names, ",")
	}))

	// This is needed so the goroutine does not end
	for {
		time.Sleep(time.Minute)
	}
}
