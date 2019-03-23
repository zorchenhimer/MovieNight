package main

import (
	"errors"
	"fmt"

	uuid "github.com/satori/go.uuid"

	"strings"
	"sync"
	"time"

	"github.com/zorchenhimer/MovieNight/common"
)

const (
	ColorServerMessage string = "#ea6260"
)

type ChatRoom struct {
	clients    map[string]*Client // this needs to be a pointer. key is suid.
	clientsMtx sync.Mutex
	tempConn   map[string]*chatConnection

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
		clients:  make(map[string]*Client),
		tempConn: make(map[string]*chatConnection),
	}

	num, err := common.LoadEmotes()
	if err != nil {
		return nil, fmt.Errorf("error loading emotes: %s", err)
	}
	fmt.Printf("Loaded %d emotes\n", num)

	//the "heartbeat" for broadcasting messages
	go cr.Broadcast()
	return cr, nil
}

func (cr *ChatRoom) JoinTemp(conn *chatConnection) (string, error) {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock()

	if conn == nil {
		return "", errors.New("conn should not be nil")
	}

	uid, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("could not create uuid: %v", err)
	}

	suid := uid.String()
	if _, ok := cr.tempConn[suid]; ok {
		return "", fmt.Errorf("%#v is already in the temp connections", suid)
	}

	cr.tempConn[suid] = conn
	return suid, nil
}

//registering a new client
//returns pointer to a Client, or Nil, if the name is already taken
func (cr *ChatRoom) Join(name, uid string) (*Client, error) {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock()

	conn, hasConn := cr.tempConn[uid]
	if !hasConn {
		return nil, errors.New("connection is missing from temp connections")
	}

	if !common.IsValidName(name) || common.IsValidColor(name) {
		return nil, UserFormatError{Name: name}
	}

	nameLower := strings.ToLower(name)
	for _, client := range cr.clients {
		if strings.ToLower(client.name) == nameLower {
			return nil, UserTakenError{Name: name}
		}
	}

	conn.clientName = name
	client := &Client{
		name:      name,
		conn:      conn,
		belongsTo: cr,
		color:     common.RandomColor(),
	}

	host := client.Host()

	if banned, names := settings.IsBanned(host); banned {
		return nil, newBannedUserError(host, name, names)
	}

	cr.clients[uid] = client
	delete(cr.tempConn, uid)

	fmt.Printf("[join] %s %s\n", host, name)
	playingCommand, err := common.NewChatCommand(common.CmdPlaying, []string{cr.playing, cr.playingLink}).ToJSON()
	if err != nil {
		fmt.Printf("Unable to encode playing command on join: %s\n", err)
	} else {
		client.Send(playingCommand)
	}
	cr.AddEventMsg(common.EvJoin, name, client.color)
	return client, nil
}

// TODO: fix this up a bit.  kick and leave are the same, incorrect, error: "That name was already used!"
//leaving the chatroom
func (cr *ChatRoom) Leave(name, color string) {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock() //preventing simultaneous access to the `clients` map

	client, suid, err := cr.getClient(name)
	if err != nil {
		fmt.Printf("[leave] Unable to get client suid %v\n", err)
		return
	}
	host := client.Host()
	name = client.name // grab the name from here for proper capitalization
	client.conn.Close()
	cr.delClient(suid)

	cr.AddEventMsg(common.EvLeave, name, color)
	fmt.Printf("[leave] %s %s\n", host, name)
}

// kicked from the chatroom
func (cr *ChatRoom) Kick(name string) string {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock() //preventing simultaneous access to the `clients` map

	client, suid, err := cr.getClient(name)
	if err != nil {
		return "Unable to get client for name " + name
	}

	if client.IsMod {
		return "You cannot kick another mod."
	}

	if client.IsAdmin {
		return "Jebaited No."
	}

	color := client.color
	host := client.Host()
	client.conn.Close()
	cr.delClient(suid)

	cr.AddEventMsg(common.EvKick, name, color)
	fmt.Printf("[kick] %s %s has been kicked\n", host, name)
	return ""
}

func (cr *ChatRoom) Ban(name string) string {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock()

	client, suid, err := cr.getClient(name)
	if err != nil {
		fmt.Printf("[ban] Unable to get client for name %q\n", name)
		return "Cannot find that name"
	}

	names := []string{}
	host := client.Host()
	color := client.color

	client.conn.Close()
	cr.delClient(suid)

	for suid, c := range cr.clients {
		if c.Host() == host {
			names = append(names, client.name)
			client.conn.Close()
			cr.delClient(suid)
		}
	}

	defer settingsMtx.Unlock()
	settingsMtx.Lock()

	err = settings.AddBan(host, names)
	if err != nil {
		fmt.Printf("[BAN] Error banning %q: %s\n", name, err)
		cr.AddEventMsg(common.EvKick, name, color)
	} else {
		cr.AddEventMsg(common.EvBan, name, color)
	}
	return ""
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

	lvl := common.CmdUser
	if from.IsMod {
		lvl = common.CmdMod
	}
	if from.IsAdmin {
		lvl = common.CmdAdmin
	}

	select {
	case cr.queue <- common.NewChatMessage(from.name, from.color, msg, lvl, t):
	default:
		fmt.Println("Unable to queue chat message. Channel full.")
	}
}

func (cr *ChatRoom) AddCmdMsg(command common.CommandType, args []string) {
	select {
	case cr.queue <- common.NewChatCommand(command, args):
	default:
		fmt.Println("Unable to queue command message.  Channel full.")
	}
}

func (cr *ChatRoom) AddModNotice(message string) {
	select {
	case cr.modqueue <- common.NewChatMessage("", "", message, common.CmdUser, common.MsgNotice):
	default:
		fmt.Println("Unable to queue notice.  Channel full.")
	}
}

func (cr *ChatRoom) AddEventMsg(event common.EventType, name, color string) {
	select {
	case cr.queue <- common.NewChatEvent(event, name, color):
	default:
		fmt.Println("Unable to queue event message.  Channel full.")
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

	client.IsMod = true
	client.SendServerMessage(`You have been modded.`)
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
			fmt.Printf("Error sending data to client: %v\n", err)
		}
	}

	for {
		select {
		case msg := <-cr.queue:
			cr.clientsMtx.Lock()
			for _, client := range cr.clients {
				go send(msg, client)
			}

			data, err := msg.ToJSON()
			if err != nil {
				fmt.Printf("Error converting ChatData to ChatDataJSON: %v\n", err)
			} else {
				for _, conn := range cr.tempConn {
					go func(c *chatConnection) {
						err = c.WriteData(data)
						if err != nil {
							fmt.Printf("Error writing data to connection: %v\n", err)
						}
					}(conn)
				}
			}

			cr.clientsMtx.Unlock()
		case msg := <-cr.modqueue:
			cr.clientsMtx.Lock()
			for _, client := range cr.clients {
				if client.IsMod || client.IsAdmin {
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

func (cr *ChatRoom) delClient(suid string) {
	delete(cr.clients, strings.ToLower(suid))
}

func (cr *ChatRoom) getClient(name string) (*Client, string, error) {
	for suid, client := range cr.clients {
		if client.name == name {
			return client, suid, nil
		}
	}

	return nil, "", fmt.Errorf("client with that name not found")
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
				return fmt.Errorf("%q is already taken.", newName)
			}
		}

		if strings.ToLower(client.name) == oldLower {
			currentClient = client
		}
	}

	if currentClient != nil {
		currentClient.name = newName
		fmt.Printf("%q -> %q\n", oldName, newName)

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
