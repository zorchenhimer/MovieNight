package common

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

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

func (dc DataMessage) GetType() DataType {
	return DT_CHAT
}

func (de DataError) GetType() DataType {
	return DT_ERROR
}

func (dc DataCommand) GetType() DataType {
	return DT_COMMAND
}

func (de DataEvent) GetType() DataType {
	return DT_EVENT
}

type DataType int

const (
	DT_INVALID DataType = iota
	DT_CHAT             // chat message
	DT_ERROR            // something went wrong with the previous request
	DT_COMMAND          // non-chat function
	DT_EVENT            // join/leave/kick/ban events
)

func ParseDataType(token json.Token) (DataType, error) {
	d := fmt.Sprintf("%.0f", token)
	val, err := strconv.ParseInt(d, 10, 32)
	if err != nil {
		fmt.Printf("Invalid data type value: %q\n", d)
		return DT_INVALID, err
	}
	return DataType(val), nil
}

type CommandType int

const (
	CMD_PLAYING CommandType = iota
	CMD_REFRESHPLAYER
	CMD_PURGECHAT
	CMD_HELP
)

type EventType int

const (
	EV_JOIN EventType = iota
	EV_LEAVE
	EV_KICK
	EV_BAN
	EV_SERVERMESSAGE
)

type MessageType int

const (
	MSG_CHAT   MessageType = iota // standard chat
	MSG_ACTION                    // /me command
	MSG_SERVER                    // server message
	MSG_ERROR
)

// TODO: Read this HTML from a template somewhere
func (dc DataMessage) HTML() string {
	fmt.Printf("message type: %d\n", dc.Type)
	switch dc.Type {
	case MSG_ACTION:
		return `<div style="color:` + dc.Color + `"><span class="name">` + dc.From +
			`</span> <span class="cmdme">` + dc.Message + `</span></div>`

	case MSG_SERVER:
		return `<div class="announcement">` + dc.Message + `</div>`

	case MSG_ERROR:
		return `<div class="error">` + dc.Message + `</div>`

	default:
		return `<div><span class="name" style="color:` + dc.Color + `">` + dc.From +
			`</span><b>:</b> <span class="msg">` + dc.Message + `</span></div>`
	}
}

func (de DataEvent) HTML() string {
	switch de.Event {
	case EV_KICK:
		return `<div class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</span> has been kicked.</div>`
	case EV_LEAVE:
		return `<div class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</span> has left the chat.</div>`
	case EV_BAN:
		return `<div class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</span> has been banned.</div>`
	case EV_JOIN:
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
		Type: DT_CHAT,
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
		Type: DT_ERROR,
		Data: DataError{Message: message},
	}
	return jsonifyChatData(d)
}

func EncodeCommand(command CommandType, args []string) (string, error) {
	d := ChatData{
		Type: DT_COMMAND,
		Data: DataCommand{
			Command:   command,
			Arguments: args,
		},
	}
	return jsonifyChatData(d)
}

func EncodeEvent(event EventType, name, color string) (string, error) {
	d := ChatData{
		Type: DT_EVENT,
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

		if fmt.Sprintf("%s", key) == "Type" {
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
			case DT_CHAT:
				d := DataMessage{}
				if err := decoder.Decode(&d); err != nil {
					return nil, fmt.Errorf("Unable to decode DataMessage: %s", err)
				}
				return d, nil
			case DT_ERROR:
				d := DataError{}
				if err := decoder.Decode(&d); err != nil {
					return nil, fmt.Errorf("Unable to decode DataError: %s", err)
				}
				return d, nil
			case DT_COMMAND:
				d := DataCommand{}
				if err := decoder.Decode(&d); err != nil {
					return nil, fmt.Errorf("Unable to decode DataCommand: %s", err)
				}
				return d, nil
			case DT_EVENT:
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
