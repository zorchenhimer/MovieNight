package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type DataInterface interface {
	HTML() string
}

type ChatData struct {
	Type DataType
	Data DataInterface
}

func (c ChatData) ToJSON() (ChatDataJSON, error) {
	rawData, err := json.Marshal(c.Data)
	return ChatDataJSON{
		Type: c.Type,
		Data: rawData,
	}, err
}

type ChatDataJSON struct {
	Type DataType
	Data json.RawMessage
}

func (c ChatDataJSON) ToData() (ChatData, error) {
	data, err := c.GetData()
	return ChatData{
		Type: c.Type,
		Data: data,
	}, err
}

func (c ChatDataJSON) GetData() (DataInterface, error) {
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

type ClientData struct {
	Type    ClientDataType
	Message string
}

func (c ClientData) HTML() string {
	// Client data is for client to server communication only, so clients should not see this
	return `<span style="color: red;">The developer messed up. You should not be seeing this.</span>`
}

type DataMessage struct {
	From    string
	Color   string
	Message string
	Level   CommandLevel
	Type    MessageType
}

// TODO: Read this HTML from a template somewhere
func (dc DataMessage) HTML() string {
	switch dc.Type {
	case MsgAction:
		return `<span style="color:` + dc.Color + `"><span class="name">` + dc.From +
			`</span> <span class="cmdme">` + dc.Message + `</span></span>`

	case MsgServer:
		return `<span class="announcement">` + dc.Message + `</span>`

	case MsgError:
		return `<span class="error">` + dc.Message + `</span>`

	case MsgNotice:
		return `<span class="notice">` + dc.Message + `</span>`

	case MsgCommandResponse:
		return `<span class="command">` + dc.Message + `</span>`

	default:
		badge := ""
		switch dc.Level {
		case CmdlMod:
			badge = `<img src="/static/img/mod.png" class="badge" />`
		case CmdlAdmin:
			badge = `<img src="/static/img/admin.png" class="badge" />`
		}
		return `<span>` + badge + `<span class="name" style="color:` + dc.Color + `">` + dc.From +
			`</span><b>:</b> <span class="msg">` + dc.Message + `</span></span>`
	}
}

func NewChatMessage(name, color, msg string, lvl CommandLevel, msgtype MessageType) ChatData {
	return ChatData{
		Type: DTChat,
		Data: DataMessage{
			From:    name,
			Color:   color,
			Message: msg,
			Type:    msgtype,
			Level:   lvl,
		},
	}
}

type DataCommand struct {
	Command   CommandType
	Arguments []string
}

func (de DataCommand) HTML() string {
	switch de.Command {
	case CmdPurgeChat:
		return `<span class="notice">Chat has been purged by a moderator.</span>`
	default:
		return ""
	}
}

func NewChatCommand(command CommandType, args []string) ChatData {
	return ChatData{
		Type: DTCommand,
		Data: DataCommand{
			Command:   command,
			Arguments: args,
		},
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
		return `<span class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</span> has been kicked.</span>`
	case EvLeave:
		return `<span class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</span> has left the chat.</span>`
	case EvBan:
		return `<span class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</span> has been banned.</span>`
	case EvJoin:
		return `<span class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</span> has joined the chat.</span>`
	case EvNameChange:
		names := strings.Split(de.User, ":")
		if len(names) != 2 {
			return `<span class="event">Somebody changed their name, but IDK who ` +
				ParseEmotes("Jebaited") + `.</span>`
		}

		return `<span class="event"><span class="name" style="color:` + de.Color + `">` +
			names[0] + `</span> has changed their name to <span class="name" style="color:` +
			de.Color + `">` + names[1] + `</span>.</span>`
	case EvNameChangeForced:
		names := strings.Split(de.User, ":")
		if len(names) != 2 {
			return `<span class="event">An admin changed somebody's name, but IDK who ` +
				ParseEmotes("Jebaited") + `.</span>`
		}

		return `<span class="event"><span class="name" style="color:` + de.Color + `">` +
			names[0] + `</span> has had their name changed to <span class="name" style="color:` +
			de.Color + `">` + names[1] + `</span> by an admin.</span>`
	}
	return ""
}

func NewChatEvent(event EventType, name, color string) ChatData {
	return ChatData{
		Type: DTEvent,
		Data: DataEvent{
			Event: event,
			User:  name,
			Color: color,
		},
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

func NewChatHiddenMessage(clientType ClientDataType, data interface{}) ChatData {
	return ChatData{
		Type: DTHidden,
		Data: HiddenMessage{
			Type: clientType,
			Data: data,
		},
	}
}

func DecodeData(rawjson string) (ChatDataJSON, error) {
	var data ChatDataJSON
	err := json.Unmarshal([]byte(rawjson), &data)
	return data, err
}
