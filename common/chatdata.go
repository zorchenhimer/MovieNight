package common

import (
	"encoding/json"
	"errors"
	"fmt"
)

type NewChatDataFunc func() (ChatData, error)

type DataInterface interface {
	HTML() string
}

type ChatData struct {
	Type DataType
	Data json.RawMessage
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
	case DTHidden:
		d := HiddenMessage{}
		err = json.Unmarshal(c.Data, &d)
		data = d
	default:
		err = fmt.Errorf("unhandled data type: %d", c.Type)
	}

	return data, err
}

func newChatData(dtype DataType, d DataInterface) (ChatData, error) {
	rawData, err := json.Marshal(d)
	return ChatData{
		Type: dtype,
		Data: rawData,
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

	case MsgNotice:
		return `<div class="notice">` + dc.Message + `</div>`

	case MsgCommandResponse:
		return `<div class="command">` + dc.Message + `</div>`
	default:
		return `<div><span class="name" style="color:` + dc.Color + `">` + dc.From +
			`</span><b>:</b> <span class="msg">` + dc.Message + `</span></div>`
	}
}

func NewChatMessage(name, color, msg string, msgtype MessageType) NewChatDataFunc {
	return func() (ChatData, error) {
		return newChatData(DTChat, DataMessage{
			From:    name,
			Color:   color,
			Message: msg,
			Type:    msgtype,
		})
	}
}

type DataCommand struct {
	Command   CommandType
	Arguments []string
}

func (de DataCommand) HTML() string {
	return ""
}

func NewChatCommand(command CommandType, args []string) NewChatDataFunc {
	return func() (ChatData, error) {
		return newChatData(DTCommand, DataCommand{
			Command:   command,
			Arguments: args,
		})
	}
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

func NewChatEvent(event EventType, name, color string) NewChatDataFunc {
	return func() (ChatData, error) {
		return newChatData(DTEvent, DataEvent{
			Event: event,
			User:  name,
			Color: color,
		})
	}
}

// DataHidden is for the server to send instructions and data
// to the client without the purpose of outputting it on the chat
type HiddenMessage struct {
	Type ClientDataType
	Data interface{}
}

func (h HiddenMessage) HTML() string {
	return ""
}

func NewChatHiddenMessage(clientType ClientDataType, data interface{}) NewChatDataFunc {
	return func() (ChatData, error) {
		return newChatData(DTHidden, HiddenMessage{
			Type: clientType,
			Data: data,
		})
	}
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
