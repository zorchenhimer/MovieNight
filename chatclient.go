package main

import (
	"fmt"
	"html"
	"net"
	"strings"
	"unicode"

	"github.com/zorchenhimer/MovieNight/common"
)

type Client struct {
	name          string // Display name
	conn          *chatConnection
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
func (cl *Client) NewMsg(data common.ClientData) {
	switch data.Type {
	case common.CdUsers:
		fmt.Printf("[chat|hidden] <%s> get list of users\n", cl.name)

		names := chat.GetNames()
		idx := -1
		for i := range names {
			if names[i] == cl.name {
				idx = i
			}
		}

		err := cl.SendChatData(common.NewChatHiddenMessage(data.Type, append(names[:idx], names[idx+1:]...)))
		if err != nil {
			fmt.Printf("Error sending chat data: %v\n", err)
		}
	case common.CdMessage:
		msg := html.EscapeString(data.Message)
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
				err := cl.SendChatData(common.NewChatMessage("", "", ParseEmotes(response), common.MsgCommandResponse))
				if err != nil {
					fmt.Printf("Error command results %v\n", err)
				}
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
}

func (cl *Client) SendChatData(newData common.NewChatDataFunc) error {
	cd, err := newData()
	if err != nil {
		return fmt.Errorf("could not create chatdata of type %d: %v", cd.Type, err)
	}
	return cl.Send(cd)
}

func (cl *Client) Send(data common.ChatData) error {
	err := cl.conn.WriteData(data)
	if err != nil {
		return fmt.Errorf("could not send message: %v", err)
	}
	return nil
}

func (cl *Client) SendServerMessage(s string) error {
	err := cl.SendChatData(common.NewChatMessage("", ColorServerMessage, s, common.MsgServer))
	if err != nil {
		return fmt.Errorf("could send server message to %s: %s; Message: %s\n", cl.name, err, s)
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

//Exiting out
func (cl *Client) Exit() {
	cl.belongsTo.Leave(cl.name, cl.color)
}

// Outgoing messages
func (cl *Client) Message(msg string) {
	msg = ParseEmotes(msg)
	cl.belongsTo.AddMsg(cl, false, false, msg)
}

// Outgoing /me command
func (cl *Client) Me(msg string) {
	msg = ParseEmotes(msg)
	cl.belongsTo.AddMsg(cl, true, false, msg)
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
