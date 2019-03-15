package common

import (
	"encoding/json"
	"errors"
)

type DataInterface interface {
	HTML() string
}

type ChatData struct {
	Hidden bool
	Type   DataType
	Data   json.RawMessage
}

func (c ChatData) GetData() (DataInterface, error) {
	var data DataInterface
	var err error

	switch c.Type {
	case DTInvalid:
		return nil, errors.New("data type is invalid")
	case DTChat:
		d := DataMessage{}
		err = json.Unmarshal(c.Data, &d)
		data = d
	case DTError:
		d := DataError{}
		err = json.Unmarshal(c.Data, &d)
		data = d
	case DTCommand:
		d := DataCommand{}
		err = json.Unmarshal(c.Data, &d)
		data = d
	case DTEvent:
		d := DataEvent{}
		err = json.Unmarshal(c.Data, &d)
		data = d
	case DTClient:
		d := ClientData{}
		err = json.Unmarshal(c.Data, &d)
		data = d
	}

	return data, err
}

func newChatData(hidden bool, dtype DataType, d DataInterface) (ChatData, error) {
	rawData, err := json.Marshal(d)
	return ChatData{
		Hidden: hidden,
		Type:   dtype,
		Data:   rawData,
	}, err
}

type ClientData struct {
	Type    ClientDataType
	Message string
}

func (c ClientData) HTML() string {
	// Client data is for client to server communication only, so clients should not see this
	return `<div style="color: red;"><span>The developer messed up. You should not be seeing this.</span></div>`
}

type DataError struct {
	Message string
}

func (de DataError) HTML() string {
	return `<div class="svmsg"><b>Error</b>: ` + de.Message + `</div>`
}

type DataMessage struct {
	From    string
	Color   string
	Message string
	Type    MessageType
}

// TODO: Read this HTML from a template somewhere
func (dc DataMessage) HTML() string {
	switch dc.Type {
	case MsgAction:
		return `<div style="color:` + dc.Color + `"><span class="name">` + dc.From +
			`</span> <span class="cmdme">` + dc.Message + `</span></div>`

	case MsgServer:
		return `<div class="announcement">` + dc.Message + `</div>`

	case MsgError:
		return `<div class="error">` + dc.Message + `</div>`

	default:
		return `<div><span class="name" style="color:` + dc.Color + `">` + dc.From +
			`</span><b>:</b> <span class="msg">` + dc.Message + `</span></div>`
	}
}

type DataCommand struct {
	Command   CommandType
	Arguments []string
}

func (de DataCommand) HTML() string {
	return ""
}

type DataEvent struct {
	Event EventType
	User  string
	Color string
}

func (de DataEvent) HTML() string {
	switch de.Event {
	case EvKick:
		return `<div class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</span> has been kicked.</div>`
	case EvLeave:
		return `<div class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</span> has left the chat.</div>`
	case EvBan:
		return `<div class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</span> has been banned.</div>`
	case EvJoin:
		return `<div class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</span> has joined the chat.</div>`
	}
	return ""
}

type ClientDataType int

const (
	CdMessage ClientDataType = iota // a normal message from the client meant to be broadcast
	CdUsers                         // get a list of users
)

type DataType int

// Data Types
const (
	DTInvalid DataType = iota
	DTChat             // chat message
	DTError            // something went wrong with the previous request
	DTCommand          // non-chat function
	DTEvent            // join/leave/kick/ban events
	DTClient           // a message coming from the client
)

type CommandType int

// Command Types
const (
	CmdPlaying CommandType = iota
	CmdRefreshPlayer
	CmdPurgeChat
	CmdHelp
)

type EventType int

// Event Types
const (
	EvJoin EventType = iota
	EvLeave
	EvKick
	EvBan
	EvServerMessage
)

type MessageType int

// Message Types
const (
	MsgChat   MessageType = iota // standard chat
	MsgAction                    // /me command
	MsgServer                    // server message
	MsgError
)

func EncodeMessage(name, color, msg string, msgtype MessageType) (string, error) {
	d, err := newChatData(false, DTChat, DataMessage{
		From:    name,
		Color:   color,
		Message: msg,
		Type:    msgtype,
	})
	if err != nil {
		return "", err
	}
	return jsonifyChatData(d)
}

func EncodeError(message string) (string, error) {
	d, err := newChatData(false, DTError, DataError{Message: message})
	if err != nil {
		return "", err
	}
	return jsonifyChatData(d)
}

func EncodeCommand(command CommandType, args []string) (string, error) {
	d, err := newChatData(false, DTCommand, DataCommand{
		Command:   command,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}
	return jsonifyChatData(d)
}

func EncodeEvent(event EventType, name, color string) (string, error) {
	d, err := newChatData(false, DTEvent, DataEvent{
		Event: event,
		User:  name,
		Color: color,
	})
	if err != nil {
		return "", err
	}
	return jsonifyChatData(d)
}

func jsonifyChatData(data ChatData) (string, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func DecodeData(rawjson string) (ChatData, error) {
	var data ChatData
	err := json.Unmarshal([]byte(rawjson), &data)
	return data, err
}
