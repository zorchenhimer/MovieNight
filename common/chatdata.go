package common

import (
	"encoding/json"
	"errors"
	"fmt"
)

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

type DataError struct {
	Message string
}

func (de DataError) HTML() string {
	return `<div class="svmsg"><b>Error</b>: ` + de.Message + `</div>`
}

func EncodeError(message string) (string, error) {
	d, err := newChatData(DTError, DataError{Message: message})
	if err != nil {
		return "", err
	}
	return jsonifyChatData(d)
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

	default:
		return `<div><span class="name" style="color:` + dc.Color + `">` + dc.From +
			`</span><b>:</b> <span class="msg">` + dc.Message + `</span></div>`
	}
}

func EncodeMessage(name, color, msg string, msgtype MessageType) (string, error) {
	d, err := newChatData(DTChat, DataMessage{
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

type DataCommand struct {
	Command   CommandType
	Arguments []string
}

func (de DataCommand) HTML() string {
	return ""
}

func EncodeCommand(command CommandType, args []string) (string, error) {
	d, err := newChatData(DTCommand, DataCommand{
		Command:   command,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}
	return jsonifyChatData(d)
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

func EncodeEvent(event EventType, name, color string) (string, error) {
	d, err := newChatData(DTEvent, DataEvent{
		Event: event,
		User:  name,
		Color: color,
	})
	if err != nil {
		return "", err
	}
	return jsonifyChatData(d)
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

func EncodeHiddenMessage(clientType ClientDataType, data interface{}) (string, error) {
	d, err := newChatData(DTHidden, HiddenMessage{
		Type: clientType,
		Data: data,
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
