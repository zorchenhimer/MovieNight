package main

import (
	"fmt"
	"html"
	"net"
	"strings"
	"unicode"

	"github.com/gorilla/websocket"
	"github.com/zorchenhimer/MovieNight/common"
)

func connSend(s string, c *websocket.Conn) {
	c.WriteMessage(websocket.TextMessage, []byte(s))
}

type Client struct {
	name          string // Display name
	conn          *websocket.Conn
	belongsTo     *ChatRoom
	color         string
	IsMod         bool
	IsAdmin       bool
	IsColorForced bool
}

var emotes map[string]string

func ParseEmotesArray(words []string) []string {
	newWords := []string{}
	for _, word := range words {
		word = strings.Trim(word, "[]")

		found := false
		for key, val := range emotes {
			if key == word {
				newWords = append(newWords, fmt.Sprintf("<img src=\"/emotes/%s\" title=\"%s\" />", val, key))
				//fmt.Printf("[emote] %s\n", val)
				found = true
			}
		}
		if !found {
			newWords = append(newWords, word)
		}
	}
	return newWords
}

func ParseEmotes(msg string) string {
	words := ParseEmotesArray(strings.Split(msg, " "))
	return strings.Join(words, " ")
}

//Client has a new message to broadcast
func (cl *Client) NewMsg(msg string) {
	msg = html.EscapeString(msg)
	msg = removeDumbSpaces(msg)
	msg = strings.Trim(msg, " ")

	// Don't send zero-length messages
	if len(msg) == 0 {
		return
	}

	if strings.HasPrefix(msg, "/") {
		// is a command
		msg = msg[1:len(msg)]
		fullcmd := strings.Split(msg, " ")
		cmd := strings.ToLower(fullcmd[0])
		args := fullcmd[1:len(fullcmd)]

		response := commands.RunCommand(cmd, args, cl)
		if response != "" {
			cl.ServerMessage(response)
			return
		}

	} else {
		// Trim long messages
		if len(msg) > 400 {
			msg = msg[0:400]
		}

		fmt.Printf("[chat] <%s> %q\n", cl.name, msg)

		// Enable links for mods and admins
		if cl.IsMod || cl.IsAdmin {
			msg = formatLinks(msg)
		}

		cl.Message(msg)
	}
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

//Exiting out
func (cl *Client) Exit() {
	cl.belongsTo.Leave(cl.name, cl.color)
}

//Sending message block to the client
func (cl *Client) Send(s string) {
	connSend(s, cl.conn)
}

// Send server message to this client
func (cl *Client) ServerMessage(msg string) {
	msg = ParseEmotes(msg)
	encoded, err := common.EncodeMessage("", "#ea6260", msg, common.MsgError)
	if err != nil {
		fmt.Printf("Error encoding server message to %s: %s; Message: %s\n", cl.name, err, msg)
		return
	}
	cl.Send(encoded)
	//cl.Send(`<span class="svmsg">` + msg + `</span><br />`)
}

// Outgoing messages
func (cl *Client) Message(msg string) {
	msg = ParseEmotes(msg)
	cl.belongsTo.AddMsg(cl, false, false, msg)
	//`<span class="name" style="color:` + cl.color + `">` + cl.name +
	//	`</span><b>:</b> <span class="msg">` + msg + `</span><br />`)
}

// Outgoing /me command
func (cl *Client) Me(msg string) {
	msg = ParseEmotes(msg)
	cl.belongsTo.AddMsg(cl, true, false, msg)
	//cl.belongsTo.AddMsg(fmt.Sprintf(`<span style="color:%s"><span class="name">%s</span> <span class="cmdme">%s</span><br />`, cl.color, cl.name, msg))
}

func (cl *Client) Mod() {
	cl.IsMod = true
}

func (cl *Client) Unmod() {
	cl.IsMod = false
}

func (cl *Client) Host() string {
	host, _, err := net.SplitHostPort(cl.conn.RemoteAddr().String())
	if err != nil {
		host = "err"
	}
	return host
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
