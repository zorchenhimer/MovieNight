package main

import (
	"fmt"

	"strings"
	"sync"
	"time"

	"github.com/zorchenhimer/MovieNight/common"
)

const (
	ColorServerMessage string = "#ea6260"
)

type ChatRoom struct {
	clients    []*Client // this needs to be a pointer. key is suid.
	clientsMtx sync.Mutex

	queue    chan common.ChatData
	modqueue chan common.ChatData // mod and admin broadcast messages

	playing     string
	playingLink string

	modPasswords    []string // single-use mod passwords
	modPasswordsMtx sync.Mutex
}

//initializing the chatroom
func newChatRoom() (*ChatRoom, error) {
	cr := &ChatRoom{
		queue:    make(chan common.ChatData, 1000),
		modqueue: make(chan common.ChatData, 1000),
		clients:  []*Client{},
	}

	err := loadEmotes()
	if err != nil {
		return nil, fmt.Errorf("error loading emotes: %s", err)
	}
	common.LogInfof("Loaded %d emotes\n", len(common.Emotes))

	//the "heartbeat" for broadcasting messages
	go cr.Broadcast()
	return cr, nil
}

// A new client joined
func (cr *ChatRoom) Join(conn *chatConnection, data common.JoinData) (*Client, error) {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock()

	sendHiddenMessage := func(cd common.ClientDataType, i interface{}) {
		// If the message cant be converted, then just don't send
		if d, err := common.NewChatHiddenMessage(cd, i).ToJSON(); err == nil {
			conn.WriteJSON(d)
		}
	}

	if settings.RoomAccess == AccessPin && data.Name == settings.RoomAccessPin {
		sendHiddenMessage(common.CdNotify, "That's the access pin! Please enter a name.")
		return nil, UserFormatError{Name: data.Name}
	}

	if !common.IsValidName(data.Name) {
		sendHiddenMessage(common.CdNotify, common.InvalidNameError)
		return nil, UserFormatError{Name: data.Name}
	}

	nameLower := strings.ToLower(data.Name)
	for _, client := range cr.clients {
		if strings.ToLower(client.name) == nameLower {
			sendHiddenMessage(common.CdNotify, "Name already taken")
			return nil, UserTakenError{Name: data.Name}
		}
	}

	// If color is invalid, then set it to a random color
	if !common.IsValidColor(data.Color) {
		data.Color = common.RandomColor()
	}

	client, err := NewClient(conn, cr, data.Name, data.Color)
	if err != nil {
		sendHiddenMessage(common.CdNotify, "Could not join client")
		return nil, fmt.Errorf("unable to join client: %v", err)
	}

	// Overwrite to use client instead
	sendHiddenMessage = func(cd common.ClientDataType, i interface{}) {
		client.SendChatData(common.NewChatHiddenMessage(cd, i))
	}

	host := client.Host()

	if banned, names := settings.IsBanned(host); banned {
		sendHiddenMessage(common.CdNotify, "You are banned")
		return nil, newBannedUserError(host, data.Name, names)
	}

	cr.clients = append(cr.clients, client)

	common.LogChatf("[join] %s %s\n", host, data.Color)
	playingCommand, err := common.NewChatCommand(common.CmdPlaying, []string{cr.playing, cr.playingLink}).ToJSON()
	if err != nil {
		common.LogErrorf("Unable to encode playing command on join: %s\n", err)
	} else {
		client.Send(playingCommand)
	}
	if !settings.LetThemLurk {
		cr.AddEventMsg(common.EvJoin, data.Name, data.Color)
	}
	sendHiddenMessage(common.CdJoin, nil)
	sendHiddenMessage(common.CdEmote, common.Emotes)

	stats.updateMaxUsers(len(cr.clients))

	return client, nil
}

// TODO: fix this up a bit.  kick and leave are the same, incorrect, error: "That
// name was already used!" leaving the chatroom
func (cr *ChatRoom) Leave(name, color string) {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock() //preventing simultaneous access to the `clients` map

	client, id, err := cr.getClient(name)
	if err != nil {
		common.LogErrorf("[leave] Unable to get client suid %v\n", err)
		return
	}
	host := client.Host()
	name = client.name // grab the name from here for proper capitalization
	client.conn.Close()
	cr.delClient(id)

	if !settings.LetThemLurk {
		cr.AddEventMsg(common.EvLeave, name, color)
	}
	common.LogChatf("[leave] %s %s\n", host, name)
}

// kicked from the chatroom
func (cr *ChatRoom) Kick(name string) error {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock() //preventing simultaneous access to the `clients` map

	client, id, err := cr.getClient(name)
	if err != nil {
		return newChatError("Unable to get client for name %s", name)
	}

	if client.CmdLevel == common.CmdlMod {
		return newChatError("You cannot kick another mod.")
	}

	if client.CmdLevel == common.CmdlAdmin {
		return newChatError("Jebaited No.")
	}

	color := client.color
	host := client.Host()
	client.conn.Close()
	cr.delClient(id)

	cr.AddEventMsg(common.EvKick, name, color)
	common.LogInfof("[kick] %s %s has been kicked\n", host, name)
	return nil
}

func (cr *ChatRoom) Ban(name string) error {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock()

	client, id, err := cr.getClient(name)
	if err != nil {
		common.LogErrorf("[ban] Unable to get client for name %q\n", name)
		return newChatError("Cannot find that name")
	}

	if client.CmdLevel == common.CmdlAdmin {
		return newChatError("You cannot ban an admin Jebaited")
	}

	names := []string{}
	host := client.Host()
	color := client.color

	// Remove the named client
	client.conn.Close()
	cr.delClient(id)

	// Remove additional clients on that IP address
	for id, c := range cr.clients {
		if c.Host() == host {
			names = append(names, client.name)
			client.conn.Close()
			cr.delClient(id)
		}
	}

	err = settings.AddBan(host, names)
	if err != nil {
		common.LogErrorf("[BAN] Error banning %q: %s\n", name, err)
		cr.AddEventMsg(common.EvKick, name, color)
	} else {
		cr.AddEventMsg(common.EvBan, name, color)
	}
	return nil
}

// Add a chat message from a viewer
func (cr *ChatRoom) AddMsg(from *Client, isAction, isServer bool, msg string) {
	t := common.MsgChat

	if isAction {
		t = common.MsgAction
	}

	if isServer {
		t = common.MsgServer
	}

	cr.AddChatMsg(common.NewChatMessage(from.name, from.color, msg, from.CmdLevel, t))
}

// Add a chat message object to the queue
func (cr *ChatRoom) AddChatMsg(data common.ChatData) {
	select {
	case cr.queue <- data:
	default:
		common.LogErrorln("Unable to queue chat message. Channel full.")
	}
}

func (cr *ChatRoom) AddCmdMsg(command common.CommandType, args []string) {
	select {
	case cr.queue <- common.NewChatCommand(command, args):
	default:
		common.LogErrorln("Unable to queue command message.  Channel full.")
	}
}

func (cr *ChatRoom) AddModNotice(message string) {
	select {
	case cr.modqueue <- common.NewChatMessage("", "", message, common.CmdlUser, common.MsgNotice):
	default:
		common.LogErrorln("Unable to queue notice.  Channel full.")
	}
}

func (cr *ChatRoom) AddEventMsg(event common.EventType, name, color string) {
	select {
	case cr.queue <- common.NewChatEvent(event, name, color):
	default:
		common.LogErrorln("Unable to queue event message.  Channel full.")
	}
}

func (cr *ChatRoom) Unmod(name string) error {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock()

	client, _, err := cr.getClient(name)
	if err != nil {
		return err
	}

	client.Unmod()
	client.SendServerMessage(`You have been unmodded.`)
	return nil
}

func (cr *ChatRoom) Mod(name string) error {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock()

	client, _, err := cr.getClient(name)
	if err != nil {
		return err
	}

	if client.CmdLevel < common.CmdlMod {
		client.CmdLevel = common.CmdlMod
		client.SendServerMessage(`You have been modded.`)
	}
	return nil
}

func (cr *ChatRoom) ForceColorChange(name, color string) error {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock()

	client, _, err := cr.getClient(name)
	if err != nil {
		return err
	}

	client.IsColorForced = true
	client.color = color
	return nil
}

func (cr *ChatRoom) UserCount() int {
	return len(cr.clients)
}

//broadcasting all the messages in the queue in one block
func (cr *ChatRoom) Broadcast() {
	send := func(data common.ChatData, client *Client) {
		err := client.SendChatData(data)
		if err != nil {
			common.LogErrorf("Error sending data to client: %v\n", err)
		}
	}

	for {
		select {
		case msg := <-cr.queue:
			cr.clientsMtx.Lock()
			for _, client := range cr.clients {
				go send(msg, client)
			}
			cr.clientsMtx.Unlock()
		case msg := <-cr.modqueue:
			cr.clientsMtx.Lock()
			for _, client := range cr.clients {
				if client.CmdLevel >= common.CmdlMod {
					send(msg, client)
				}
			}
			cr.clientsMtx.Unlock()
		default:
			time.Sleep(50 * time.Millisecond)
			// No messages to send
			// This default block is required so the above case
			// does not block.
		}
	}
}

func (cr *ChatRoom) ClearPlaying() {
	cr.playing = ""
	cr.playingLink = ""
	cr.AddCmdMsg(common.CmdPlaying, []string{"", ""})
}

func (cr *ChatRoom) SetPlaying(title, link string) {
	cr.playing = title
	cr.playingLink = link
	cr.AddCmdMsg(common.CmdPlaying, []string{title, link})
}

func (cr *ChatRoom) GetNames() []string {
	names := []string{}
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock()

	for _, val := range cr.clients {
		names = append(names, val.name)
	}

	return names
}

func (cr *ChatRoom) delClient(sliceId int) {
	cr.clients = append(cr.clients[:sliceId], cr.clients[sliceId+1:]...)
}

func (cr *ChatRoom) getClient(name string) (*Client, int, error) {
	for id, client := range cr.clients {
		if client.name == name {
			return client, id, nil
		}
	}

	return nil, -1, fmt.Errorf("client with that name not found")
}

func (cr *ChatRoom) generateModPass() string {
	defer cr.modPasswordsMtx.Unlock()
	cr.modPasswordsMtx.Lock()

	pass, err := generatePass(time.Now().Unix())
	if err != nil {
		return fmt.Sprintf("Error generating moderator password: %s", err)
	}

	// Make sure the password is unique
	for existsInSlice(cr.modPasswords, pass) {
		pass, err = generatePass(time.Now().Unix())
		if err != nil {
			return fmt.Sprintf("Error generating moderator password: %s", err)
		}
	}

	cr.modPasswords = append(cr.modPasswords, pass)
	return pass
}

func (cr *ChatRoom) redeemModPass(pass string) bool {
	if pass == "" {
		return false
	}

	defer cr.modPasswordsMtx.Unlock()
	cr.modPasswordsMtx.Lock()

	if existsInSlice(cr.modPasswords, pass) {
		cr.modPasswords = removeFromSlice(cr.modPasswords, pass)
		return true
	}
	return false
}

func removeFromSlice(slice []string, needle string) []string {
	slc := []string{}
	for _, item := range slice {
		if item != needle {
			slc = append(slc, item)
		}
	}
	return slc
}

func existsInSlice(slice []string, needle string) bool {
	for _, item := range slice {
		if item == needle {
			return true
		}
	}
	return false
}

func (cr *ChatRoom) changeName(oldName, newName string, forced bool) error {
	cr.clientsMtx.Lock()
	defer cr.clientsMtx.Unlock()

	if !common.IsValidName(newName) {
		return fmt.Errorf("%q nick is not a valid name", newName)
	}

	newLower := strings.ToLower(newName)
	oldLower := strings.ToLower(oldName)

	var currentClient *Client
	for _, client := range cr.clients {
		if strings.ToLower(client.name) == newLower {
			if strings.ToLower(client.name) != oldLower {
				return fmt.Errorf("%q is already taken", newName)
			}
		}

		if strings.ToLower(client.name) == oldLower {
			currentClient = client
		}
	}

	if currentClient != nil {
		err := currentClient.setName(newName)
		if err != nil {
			return fmt.Errorf("could not set client name to %#v: %v", newName, err)
		}
		common.LogDebugf("%q -> %q\n", oldName, newName)

		if forced {
			cr.AddEventMsg(common.EvNameChangeForced, oldName+":"+newName, currentClient.color)
			currentClient.IsNameForced = true
		} else {
			cr.AddEventMsg(common.EvNameChange, oldName+":"+newName, currentClient.color)
		}
		return nil
	}

	return fmt.Errorf("Client not found with name %q", oldName)
}
