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

type DataChat struct {
	From     string
	Color    string
	Message  string
	IsAction bool
	IsServer bool // server message?
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

type VisibleData interface {
	HTML() string
}

func (dc DataChat) GetType() DataType {
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
)

type EventType int

const (
	EV_JOIN EventType = iota
	EV_LEAVE
	EV_KICK
	EV_BAN
	EV_SERVERMESSAGE
)

// TODO: Read this HTML from a template somewhere
func (dc DataChat) HTML() string {
	if dc.IsAction {
		return `<span style="color:` + dc.Color + `"><span class="name">` + dc.From +
			`</span> <span class="cmdme">` + dc.Message + `</span><br />`
	}

	if dc.IsServer {
		return `<div class="announcement">` + dc.Message + `</div>`
	}

	return `<span class="name" style="color:` + dc.Color + `">` + dc.From +
		`</span><b>:</b> <span class="msg">` + dc.Message + `</span><br />`
}

func (de DataEvent) HTML() string {
	switch de.Event {
	case EV_KICK:
		return `<span class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</b> has been kicked.</span><br />`
	case EV_LEAVE:
		return `<span class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</b> has left the chat.</span><br />`
	case EV_BAN:
		return `<span class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</b> has been banned.</span><br />`
	case EV_JOIN:
		return `<span class="event"><span class="name" style="color:` + de.Color + `">` +
			de.User + `</b> has joined the chat.</span><br />`
	}
	return ""
}

func (de DataError) HTML() string {
	return `<span class="svmsg"><b>Error</b>: ` + de.Message + `</span><br />`
}

func (de DataCommand) HTML() string {
	return ""
}

func EncodeChat(name, color, msg string, isAction, isServer bool) (string, error) {
	d := ChatData{
		Type: DT_CHAT,
		Data: DataChat{
			From:     name,
			Color:    color,
			IsAction: isAction,
			IsServer: isServer,
			Message:  msg,
		},
	}
	j, err := jsonifyChatData(d)
	fmt.Printf("Err: %s; data: %s\n", err, j)
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
	t, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("Open bracket token error: %s", err)
	}
	fmt.Printf("Token: %q\n", t)

	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("Error decoding token: %s", err)
		}
		fmt.Printf("Key: %q\n", key)

		if fmt.Sprintf("%s", key) == "Type" {
			value, err := decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("Error decoding data value: %q", err)
			}

			data.Type, err = ParseDataType(value)
			if err != nil {
				return nil, fmt.Errorf("Error parsing data type: %d", data.Type)
			}
			fmt.Printf("data.Type: %d\n", data.Type)
		} else {
			fmt.Printf("Key: %q\n", key)
			//value, err := decoder.Token()
			//if err != nil {
			//	return nil, fmt.Errorf("Error decoding value token: %s", err)
			//}
			//fmt.Printf("Value: %q", value)

			switch DataType(data.Type) {
			case DT_CHAT:
				d := DataChat{}
				if err := decoder.Decode(&d); err != nil {
					return nil, fmt.Errorf("Unable to decode DataChat: %s", err)
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

			//fmt.Printf("di.GetType: %q\n", di.GetType())
			//err = decoder.Decode(&di)
			//if err != nil {
			//	return nil, fmt.Errorf("Unable to decode data into interface: %s", err)
			//}
			//return di, nil
		}
	}
	return nil, fmt.Errorf("Incomplete data")
}
