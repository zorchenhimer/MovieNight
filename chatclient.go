package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/zorchenhimer/MovieNight/common"
)

var (
	regexSpoiler = regexp.MustCompile(`\|\|(.*?)\|\|`)
	spoilerStart = `<span class="spoiler" onclick='$(this).removeClass("spoiler").addClass("spoiler-active")'>`
	spoilerEnd   = `</span>`
)

type Client struct {
	name          string // Display name
	conn          *chatConnection
	belongsTo     *ChatRoom
	color         string
	CmdLevel      common.CommandLevel
	IsColorForced bool
	IsNameForced  bool

	// Times since last event.  use time.Duration.Since()
	nextChat  time.Time // rate limit chat messages
	nextNick  time.Time // rate limit nickname changes
	nextColor time.Time // rate limit color changes
	nextAuth  time.Time // rate limit failed auth attempts.  Sould prolly have a backoff policy.
	authTries int       // number of failed auth attempts

	nextDuplicate time.Time
	lastMsg       string
}

func NewClient(connection *chatConnection, room *ChatRoom, name, color string) (*Client, error) {
	c := &Client{
		conn:      connection,
		belongsTo: room,
		color:     color,
	}

	if err := c.setName(name); err != nil {
		return nil, fmt.Errorf("could not set client name to %#v: %w", name, err)
	}

	// Set initial vaules to their rate limit duration in the past.
	c.nextChat = time.Now()
	c.nextNick = time.Now()
	c.nextColor = time.Now()
	c.nextAuth = time.Now()

	return c, nil
}

// Client has a new message to broadcast
func (cl *Client) NewMsg(data common.ClientData) {
	switch data.Type {
	case common.CdAuth:
		common.LogChatf("[chat|hidden] <%s> get auth level\n", cl.name)
		err := cl.SendChatData(common.NewChatHiddenMessage(data.Type, cl.CmdLevel))
		if err != nil {
			common.LogErrorf("Error sending auth level to client: %v\n", err)
		}
	case common.CdUsers:
		common.LogChatf("[chat|hidden] <%s> get list of users\n", cl.name)

		names := chat.GetNames()
		idx := -1
		for i := range names {
			if names[i] == cl.name {
				idx = i
			}
		}

		err := cl.SendChatData(common.NewChatHiddenMessage(data.Type, append(names[:idx], names[idx+1:]...)))
		if err != nil {
			common.LogErrorf("Error sending chat data: %v\n", err)
		}
	case common.CdMessage:
		msg := html.EscapeString(data.Message)
		msg = removeDumbSpaces(msg)
		msg = strings.Trim(msg, " ")

		// Add the spoiler tag outside of the command vs message statement
		// because the /me command outputs to the messages
		msg = addSpoilerTags(msg)

		msgLen := len(msg)

		// Don't send zero-length messages
		if msgLen == 0 {
			return
		}

		if strings.HasPrefix(msg, "/") {
			// is a command
			msg = msg[1:msgLen]
			fullcmd := strings.Split(msg, " ")
			cmd := strings.ToLower(fullcmd[0])
			fullcmdLen := len(fullcmd)
			args := fullcmd[1:fullcmdLen]

			response, err := commands.RunCommand(cmd, args, cl)
			if response != "" || err != nil {
				msgType := common.MsgCommandResponse
				respText := response
				if err != nil {
					respText = err.Error()
					msgType = common.MsgCommandError
				}

				err := cl.SendChatData(common.NewChatMessage("", "",
					common.ParseEmotes(respText),
					common.CmdlUser,
					msgType))
				if err != nil {
					common.LogErrorf("Error command results %v\n", err)
				}
				return
			}

		} else {
			// Limit the rate of sent chat messages.  Ignore mods and admins
			if time.Now().Before(cl.nextChat) && cl.CmdLevel == common.CmdlUser {
				err := cl.SendChatData(common.NewChatMessage("", "",
					"Slow down.",
					common.CmdlUser,
					common.MsgCommandResponse))
				if err != nil {
					common.LogErrorf("Unable to send slowdown for chat: %v", err)
				}
				return
			}

			// Trim long messages
			if msgLen > 400 {
				msg = msg[0:400]
			}

			// Limit the rate of duplicate messages.  Ignore mods and admins.
			// Only checks the last message.
			if strings.TrimSpace(strings.ToLower(msg)) == cl.lastMsg &&
				time.Now().Before(cl.nextDuplicate) &&
				cl.CmdLevel == common.CmdlUser {
				err := cl.SendChatData(common.NewChatMessage("", "",
					common.ParseEmotes("You already sent that PeepoSus"),
					common.CmdlUser,
					common.MsgCommandResponse))
				if err != nil {
					common.LogErrorf("Unable to send slowdown for chat: %v", err)
				}
				return
			}

			cl.nextChat = time.Now().Add(time.Second * settings.RateLimitChat)
			cl.nextDuplicate = time.Now().Add(time.Second * settings.RateLimitDuplicate)
			cl.lastMsg = strings.TrimSpace(strings.ToLower(msg))

			common.LogChatf("[chat] <%s> %q\n", cl.name, msg)

			// Enable links for mods and admins
			if cl.CmdLevel >= common.CmdlMod {
				msg = formatLinks(msg)
			}

			cl.Message(msg)
		}
	}
}

func (cl *Client) SendChatData(data common.ChatData) error {
	// Don't send chat or event data to clients that have not fully joined the
	// chatroom (ie, they have not set a name).
	if cl.name == "" && (data.Type == common.DTChat || data.Type == common.DTEvent) {
		return nil
	}

	// Colorize name on chat messages
	if data.Type == common.DTChat {
		data = cl.replaceColorizedName(data)
	}

	cd, err := data.ToJSON()
	if err != nil {
		return fmt.Errorf("could not create ChatDataJSON of type %d: %w", data.Type, err)
	}
	return cl.Send(cd)
}

func (cl *Client) Send(data common.ChatDataJSON) error {
	err := cl.conn.WriteData(data)
	if err != nil {
		return fmt.Errorf("could not send message: %w", err)
	}
	return nil
}

func (cl *Client) SendServerMessage(s string) error {
	err := cl.SendChatData(common.NewChatMessage("", ColorServerMessage, s, common.CmdlUser, common.MsgServer))
	if err != nil {
		return fmt.Errorf("could send server message to %s: message - %#v: %w", cl.name, s, err)
	}
	return nil
}

// Make links clickable
func formatLinks(input string) string {
	newMsg := []string{}
	for _, word := range strings.Split(input, " ") {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			word = html.UnescapeString(word)
			word = fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, word, word)
		}
		newMsg = append(newMsg, word)
	}
	return strings.Join(newMsg, " ")
}

// Exiting out
func (cl *Client) Exit() {
	cl.belongsTo.Leave(cl.name, cl.color)
}

// Outgoing messages
func (cl *Client) Message(msg string) {
	msg = common.ParseEmotes(msg)
	cl.belongsTo.AddMsg(cl, false, false, msg)
}

// Outgoing /me command
func (cl *Client) Me(msg string) {
	msg = common.ParseEmotes(msg)
	cl.belongsTo.AddMsg(cl, true, false, msg)
}

func (cl *Client) Mod() {
	if cl.CmdLevel < common.CmdlMod {
		cl.CmdLevel = common.CmdlMod
	}
}

func (cl *Client) Unmod() {
	cl.CmdLevel = common.CmdlUser
}

func (cl *Client) Host() string {
	return cl.conn.Host()
}

func (cl *Client) setName(s string) error {
	cl.name = s
	if cl.conn != nil {
		cl.conn.clientName = s
	}
	return nil
}

func (cl *Client) setColor(s string) error {
	cl.color = s
	return cl.SendChatData(common.NewChatHiddenMessage(common.CdColor, cl.color))
}

func (cl *Client) replaceColorizedName(chatData common.ChatData) common.ChatData {
	data := chatData.Data.(common.DataMessage)
	words := strings.Split(data.Message, " ")
	newWords := []string{}

	for _, word := range words {
		if strings.EqualFold(cl.name, strings.TrimPrefix(word, "@")) {
			newWords = append(newWords, `<span class="mention">`+word+`</span>`)
		} else {
			newWords = append(newWords, word)
		}
	}

	data.Message = strings.Join(newWords, " ")
	chatData.Data = data
	return chatData
}

var dumbSpaces = []string{
	"\n",
	"\t",
	"\r",
	"\u200b",
}

func removeDumbSpaces(msg string) string {
	for _, ds := range dumbSpaces {
		msg = strings.ReplaceAll(msg, ds, " ")
	}

	newMsg := ""
	for _, r := range msg {
		if unicode.IsSpace(r) {
			newMsg += " "
		} else {
			newMsg += string(r)
		}
	}
	return newMsg
}

func addSpoilerTags(msg string) string {
	return regexSpoiler.ReplaceAllString(msg, fmt.Sprintf(`%s$1%s`, spoilerStart, spoilerEnd))
}
