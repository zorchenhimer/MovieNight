package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"
)

const htmlConversionErrSpan = `<span class="error">Could not process %s.</span>`

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

var (
	cmdme        = template.Must(template.New("cmdme").Parse(`<span style="color:{{.Color}}"><span class="name">{{.From}}</span> <span class="cmdme">{{.Message}}</span></span>`))
	announcement = template.Must(template.New("announcement").Parse(`<span class="announcement">{{.Message}}</span>`))
	errormsg     = template.Must(template.New("error").Parse(`<span class="error">{{.Message}}</span>`))
	notice       = template.Must(template.New("notice").Parse(`<span class="notice">{{.Message}}</span>`))
	command      = template.Must(template.New("command").Parse(`<span class="command">{{.Message}}</span>`))
	commanderror = template.Must(template.New("commanderror").Parse(`<span class="commanderror">{{.Message}}</span>`))
	cmdlMod      = template.Must(template.New("cmdlMod").Parse(`<span><img src="/static/img/mod.png" class="badge" /><span class="name" style="color:{{.Color}}">{{.From}}</span><b>:</b> <span class="msg">{{.Message}}</span></span>`))
	cmdlAdmin    = template.Must(template.New("CmdlAdmin").Parse(`<span><img src="/static/img/admin.png" class="badge" /><span class="name" style="color:{{.Color}}">{{.From}}</span><b>:</b> <span class="msg">{{.Message}}</span></span>`))
	defaultMsg   = template.Must(template.New("defaultMsg").Parse(`<span><span class="name" style="color:{{.Color}}">{{.From}}</span><b>:</b> <span class="msg">{{.Message}}</span></span>`))
)

// TODO: Read this HTML from a template somewhere
func (dc DataMessage) HTML() string {
	var err error
	buf := &bytes.Buffer{}
	switch dc.Type {
	case MsgAction:
		err = cmdme.Execute(buf, dc)
	case MsgServer:
		err = announcement.Execute(buf, dc)
	case MsgError:
		err = errormsg.Execute(buf, dc)
	case MsgNotice:
		err = notice.Execute(buf, dc)
	case MsgCommandResponse:
		err = command.Execute(buf, dc)
	case MsgCommandError:
		err = commanderror.Execute(buf, dc)

	default:
		switch dc.Level {
		case CmdlMod:
			err = cmdlMod.Execute(buf, dc)
		case CmdlAdmin:
			err = cmdlAdmin.Execute(buf, dc)
		default:
			err = defaultMsg.Execute(buf, dc)
		}
	}

	if err != nil {
		LogErrorf("Could not convert data message to html: %v\n", err)
		return fmt.Sprintf(htmlConversionErrSpan, "message")
	}

	return buf.String()
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
	Users []string
}

var (
	evKick               = template.Must(template.New("evKick").Parse(`<span class="event"><span class="name" style="color:{{.Color}}">{{.User}}</span> has been kicked.</span>`))
	evLeave              = template.Must(template.New("evLeave").Parse(`<span class="event"><span class="name" style="color:{{.Color}}">{{.User}}</span> has left the chat.</span>`))
	evBan                = template.Must(template.New("evBan").Parse(`<span class="event"><span class="name" style="color:{{.Color}}">{{.User}}</span> has been banned.</span>`))
	evJoin               = template.Must(template.New("evJoin").Parse(`<span class="event"><span class="name" style="color:{{.Color}}">{{.User}}</span> has joined the chat.</span>`))
	evNameChangeWC       = template.Must(template.New("evNameChangeWC").Parse(`<span class="event">Somebody changed their name, but IDK who {{.}}.</span>`))
	evNameChange         = template.Must(template.New("evNameChange").Parse(`<span class="event"><span class="name" style="color:{{.Color}}">{{index .Users 0}}</span> has changed their name to <span class="name" style="color:{{.Color}}">{{index .Users 1}}</span>.</span>`))
	evNameChangeForced   = template.Must(template.New("evNameChangeForced").Parse(`<span class="event"><span class="name" style="color:{{.Color}}">{{index .Users 0}}</span> has had their name changed to <span class="name" style="color:{{.Color}}">{{index .Users 1}}</span> by an admin.</span>`))
	evNameChangeForcedWC = template.Must(template.New("evNameChangeForcedWC").Parse(`<span class="event">An admin changed somebody's name, but IDK who {{.}}.</span>`))
)

func (de DataEvent) HTML() string {
	var err error
	buf := &bytes.Buffer{}
	switch de.Event {
	case EvKick:
		err = evKick.Execute(buf, de)
	case EvLeave:
		err = evLeave.Execute(buf, de)
	case EvBan:
		err = evBan.Execute(buf, de)
	case EvJoin:
		err = evJoin.Execute(buf, de)
	case EvNameChange:
		de.Users = strings.Split(de.User, ":")
		if len(de.Users) < 2 {
			err = evNameChangeWC.Execute(buf, ParseEmotes("Jebaited"))
		} else {
			err = evNameChange.Execute(buf, de)
		}
	case EvNameChangeForced:
		de.Users = strings.Split(de.User, ":")
		if len(de.Users) < 2 {
			err = evNameChangeForcedWC.Execute(buf, ParseEmotes("Jebaited"))
		} else {
			err = evNameChangeForced.Execute(buf, de)
		}
	}

	if err != nil {
		LogErrorf("Could not convert event message to html: %v\n", err)
		return fmt.Sprintf(htmlConversionErrSpan, "event")
	}

	return buf.String()
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

type JoinData struct {
	Name  string
	Color string
}
