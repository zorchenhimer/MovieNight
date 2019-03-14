package common

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const CommandNameSeparator = ","

type ChatCommandNames []string

func (c ChatCommandNames) String() string {
	return strings.Join(c, CommandNameSeparator)
}

// Names for commands
var (
	// User Commands
	CNMe     ChatCommandNames = []string{"me"}
	CNHelp   ChatCommandNames = []string{"help"}
	CNCount  ChatCommandNames = []string{"count"}
	CNColor  ChatCommandNames = []string{"color", "colour"}
	CNWhoAmI ChatCommandNames = []string{"w", "whoami"}
	CNAuth   ChatCommandNames = []string{"auth"}
	CNUsers  ChatCommandNames = []string{"users"}
	// Mod Commands
	CNSv      ChatCommandNames = []string{"sv"}
	CNPlaying ChatCommandNames = []string{"playing"}
	CNUnmod   ChatCommandNames = []string{"unmod"}
	CNKick    ChatCommandNames = []string{"kick"}
	CNBan     ChatCommandNames = []string{"ban"}
	CNUnban   ChatCommandNames = []string{"unban"}
	// Admin Commands
	CNMod          ChatCommandNames = []string{"mod"}
	CNReloadPlayer ChatCommandNames = []string{"reloadplayer"}
	CNReloadEmotes ChatCommandNames = []string{"reloademotes"}
	CNModpass      ChatCommandNames = []string{"modpass"}
)

var ChatCommands = []ChatCommandNames{
	CNMe, CNHelp, CNCount, CNColor, CNWhoAmI, CNAuth, CNUsers,
	CNSv, CNPlaying, CNUnmod, CNKick, CNBan, CNUnban,
	CNMod, CNReloadPlayer, CNReloadEmotes, CNModpass,
}

func GetFullChatCommand(c string) string {
	for _, names := range ChatCommands {
		for _, n := range names {
			if c == n {
				return names.String()
			}
		}
	}
	return ""
}

type ClientData struct {
	Type    ClientDataType
	Message string
}

type ChatData struct {
	Type DataType
	Data DataInterface
}

type DataError struct {
	Message string
}

type DataMessage struct {
	From    string
	Color   string
	Message string
	Type    MessageType
}

type DataCommand struct {
	Command   CommandType
	Arguments []string
}

type DataEvent struct {
	Event EventType
	User  string
	Color string
}

type DataInterface interface {
	GetType() DataType
	HTML() string
}

func (c ClientData) GetType() DataType {
	return DTClient
}

func (d DataMessage) GetType() DataType {
	return DTChat
}

func (d DataError) GetType() DataType {
	return DTError
}

func (d DataCommand) GetType() DataType {
	return DTCommand
}

func (d DataEvent) GetType() DataType {
	return DTEvent
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

func ParseDataType(token json.Token) (DataType, error) {
	d := fmt.Sprintf("%.0f", token)
	val, err := strconv.ParseInt(d, 10, 32)
	if err != nil {
		fmt.Printf("Invalid data type value: %q\n", d)
		return DTInvalid, err
	}
	return DataType(val), nil
}

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

func (c ClientData) HTML() string {
	// Client data is for client to server communication only, so clients should not see this
	return `<div style="color: red;"><span>The developer messed up. You should not be seeing this.</span></div>`
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

func (de DataError) HTML() string {
	return `<div class="svmsg"><b>Error</b>: ` + de.Message + `</div>`
}

func (de DataCommand) HTML() string {
	return ""
}

func EncodeMessage(name, color, msg string, msgtype MessageType) (string, error) {
	d := ChatData{
		Type: DTChat,
		Data: DataMessage{
			From:    name,
			Color:   color,
			Message: msg,
			Type:    msgtype,
		},
	}
	j, err := jsonifyChatData(d)
	return j, err
}

func EncodeError(message string) (string, error) {
	d := ChatData{
		Type: DTError,
		Data: DataError{Message: message},
	}
	return jsonifyChatData(d)
}

func EncodeCommand(command CommandType, args []string) (string, error) {
	d := ChatData{
		Type: DTCommand,
		Data: DataCommand{
			Command:   command,
			Arguments: args,
		},
	}
	return jsonifyChatData(d)
}

func EncodeEvent(event EventType, name, color string) (string, error) {
	d := ChatData{
		Type: DTEvent,
		Data: DataEvent{
			Event: event,
			User:  name,
			Color: color,
		},
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

func DecodeData(rawjson string) (DataInterface, error) {
	data := ChatData{}
	decoder := json.NewDecoder(strings.NewReader(rawjson))

	// Open bracket
	_, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("Open bracket token error: %s", err)
	}

	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("Error decoding token: %s", err)
		}

		if key.(string) == "Type" {
			value, err := decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("Error decoding data value: %q", err)
			}

			data.Type, err = ParseDataType(value)
			if err != nil {
				return nil, fmt.Errorf("Error parsing data type: %d", data.Type)
			}
		} else {

			switch DataType(data.Type) {
			case DTChat:
				d := DataMessage{}
				if err := decoder.Decode(&d); err != nil {
					return nil, fmt.Errorf("Unable to decode DataMessage: %s", err)
				}
				return d, nil
			case DTError:
				d := DataError{}
				if err := decoder.Decode(&d); err != nil {
					return nil, fmt.Errorf("Unable to decode DataError: %s", err)
				}
				return d, nil
			case DTCommand:
				d := DataCommand{}
				if err := decoder.Decode(&d); err != nil {
					return nil, fmt.Errorf("Unable to decode DataCommand: %s", err)
				}
				return d, nil
			case DTEvent:
				d := DataEvent{}
				if err := decoder.Decode(&d); err != nil {
					return nil, fmt.Errorf("Unable to decode DataEvent: %s", err)
				}
				return d, nil
			default:
				return nil, fmt.Errorf("Invalid data type: %d", data.Type)
			}

		}
	}
	return nil, fmt.Errorf("Incomplete data")
}
