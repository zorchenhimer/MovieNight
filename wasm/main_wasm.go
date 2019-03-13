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

	//dt, err := common.ParseDataType(*data.Type)
	//if err != nil {
	//	fmt.Printf("Error decoding type: %s\n", err)
	//	js.Call("appendMessages", v)
	//	return
	//}

	switch data.GetType() {
	case common.DT_CHAT, common.DT_EVENT, common.DT_ERROR:
		fmt.Printf("data raw: %q\n", data)
		dc := common.VisibleData(data)
		js.Call("appendMessages", dc.HTML())
	case common.DT_COMMAND:
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
