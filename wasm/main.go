package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dennwc/dom/js"
	"github.com/zorchenhimer/MovieNight/common"
)

const (
	keyTab   = 9
	keyEnter = 13
	keyUp    = 38
	keyDown  = 40
)

var (
	currentName   string
	auth          common.CommandLevel
	names         []string
	filteredNames []string
)

// The returned value is a bool deciding to prevent the event from propagating
func processMessageKey(this js.Value, v []js.Value) interface{} {
	if len(filteredNames) == 0 || currentName == "" {
		return false
	}

	startIdx := v[0].Get("target").Get("selectionStart").Int()
	keyCode := v[0].Get("keyCode").Int()
	switch keyCode {
	case keyUp, keyDown:
		newidx := 0
		for i, n := range filteredNames {
			if n == currentName {
				newidx = i
				if keyCode == keyDown {
					newidx = i + 1
					if newidx == len(filteredNames) {
						newidx--
					}
				} else if keyCode == keyUp {
					newidx = i - 1
					if newidx < 0 {
						newidx = 0
					}
				}
				break
			}
		}
		currentName = filteredNames[newidx]
	case keyTab, keyEnter:
		msg := js.Get("msg")
		val := msg.Get("value").String()
		newval := val[:startIdx]

		if i := strings.LastIndex(newval, "@"); i != -1 {
			newval = newval[:i+1] + currentName
		}

		endVal := val[startIdx:]
		if len(val) == startIdx || val[startIdx:][0] != ' ' {
			// insert a space into val so selection indexing can be one line
			endVal = " " + endVal
		}
		msg.Set("value", newval+endVal)
		msg.Set("selectionStart", len(newval)+1)
		msg.Set("selectionEnd", len(newval)+1)

		// Clear out filtered names since it is no longer needed
		filteredNames = nil
	default:
		// We only want to handle the caught keys, so return early
		return false
	}

	updateSuggestionDiv()
	return true
}

func processMessage(v []js.Value) {
	msg := js.Get("msg")
	text := strings.ToLower(msg.Get("value").String())
	startIdx := msg.Get("selectionStart").Int()

	filteredNames = nil
	if len(text) != 0 {
		if len(names) > 0 {
			var caretIdx int
			textParts := strings.Split(text, " ")

			for i, word := range textParts {
				// Increase caret index at beginning if not first word to account for spaces
				if i != 0 {
					caretIdx++
				}

				// It is possible to have a double space "  ", which will lead to an
				// empty string element in the slice. Also check that the index of the
				// cursor is between the start of the word and the end
				if len(word) > 0 && word[0] == '@' &&
					caretIdx <= startIdx && startIdx <= caretIdx+len(word) {
					// fill filtered first so the "modifier" keys can modify it
					for _, n := range names {
						if len(word) == 1 || strings.HasPrefix(strings.ToLower(n), word[1:]) {
							filteredNames = append(filteredNames, n)
						}
					}
				}

				if len(filteredNames) > 0 {
					currentName = ""
					break
				}

				caretIdx += len(word)
			}
		}
	}

	updateSuggestionDiv()
}

func updateSuggestionDiv() {
	const selectedClass = ` class="selectedName"`

	var divs []string
	if len(filteredNames) > 0 {
		// set current name to first if not set already
		if currentName == "" {
			currentName = filteredNames[0]
		}

		var hasCurrentName bool
		divs = make([]string, len(filteredNames))

		// Create inner body of html
		for i := range filteredNames {
			divs[i] = "<div"
			if filteredNames[i] == currentName {
				hasCurrentName = true
				divs[i] += selectedClass
			}
			divs[i] += ">" + filteredNames[i] + "</div>"
		}

		if !hasCurrentName {
			divs[0] = divs[0][:4] + selectedClass + divs[0][4:]
		}
	}
	// The \n is so it's easier to read th source in web browsers for the dev
	js.Get("suggestions").Set("innerHTML", strings.Join(divs, "\n"))
}

func recieve(v []js.Value) {
	if len(v) == 0 {
		fmt.Println("No data received")
		return
	}

	chatJSON, err := common.DecodeData(v[0].String())
	if err != nil {
		fmt.Printf("Error decoding data: %s\n", err)
		js.Call("appendMessages", v)
		return
	}

	chat, err := chatJSON.ToData()
	if err != nil {
		fmt.Printf("Error converting ChatDataJSON to ChatData of type %d: %v", chatJSON.Type, err)
	}

	switch chat.Type {
	case common.DTHidden:
		h := chat.Data.(common.HiddenMessage)
		switch h.Type {
		case common.CdUsers:
			names = nil
			for _, i := range h.Data.([]interface{}) {
				names = append(names, i.(string))
			}
		case common.CdAuth:
			auth = h.Data.(common.CommandLevel)
		}
	case common.DTEvent:
		d := chat.Data.(common.DataEvent)
		// A server message is the only event that doesn't deal with names.
		if d.Event != common.EvServerMessage {
			websocketSend("", common.CdUsers)
		}
		// on join or leave, update list of possible user names
		fallthrough
	case common.DTChat:
		js.Call("appendMessages", chat.Data.HTML())
	case common.DTCommand:
		d := chat.Data.(common.DataCommand)

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
			js.Call("purgeChat", nil)
			js.Call("appendMessages", d.HTML())
		case common.CmdHelp:
			url := "/help"
			if d.Arguments != nil && len(d.Arguments) > 0 {
				url = d.Arguments[0]
			}
			js.Call("appendMessages", d.HTML())
			js.Get("window").Call("open", url, "_blank", "menubar=0,status=0,toolbar=0,width=300,height=600")
		}
	}
}

func websocketSend(msg string, dataType common.ClientDataType) error {
	if strings.TrimSpace(msg) == "" && dataType == common.CdMessage {
		return nil
	}

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

func isValidColor(this js.Value, v []js.Value) interface{} {
	if len(v) != 1 {
		return false
	}
	return common.IsValidColor(v[0].String())
}

func isValidName(this js.Value, v []js.Value) interface{} {
	if len(v) != 1 {
		return false
	}
	return common.IsValidName(v[0].String())
}

func debugValues(v []js.Value) {
	fmt.Printf("currentName %#v\n", currentName)
	fmt.Printf("auth %#v\n", auth)
	fmt.Printf("names %#v\n", names)
	fmt.Printf("filteredNames %#v\n", filteredNames)
}

func main() {
	js.Set("processMessageKey", js.FuncOf(processMessageKey))
	js.Set("sendMessage", js.FuncOf(send))
	js.Set("isValidColor", js.FuncOf(isValidColor))
	js.Set("isValidName", js.FuncOf(isValidName))

	js.Set("recieveMessage", js.CallbackOf(recieve))
	js.Set("processMessage", js.CallbackOf(processMessage))
	js.Set("debugValues", js.CallbackOf(debugValues))

	// This is needed so the goroutine does not end
	for {
		// heatbeat to keep connection alive to deal with nginx
		if js.Get("inChat").Bool() {
			websocketSend("", common.CdPing)
		}
		time.Sleep(time.Second * 10)
	}
}
