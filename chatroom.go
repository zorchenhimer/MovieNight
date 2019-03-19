package main

import (
	"errors"
	"fmt"

	uuid "github.com/satori/go.uuid"

	"math/rand"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/zorchenhimer/MovieNight/common"
)

const (
	UsernameMaxLength  int    = 36
	UsernameMinLength  int    = 3
	ColorServerMessage string = "#ea6260"
)

var re_username *regexp.Regexp = regexp.MustCompile(`^[0-9a-zA-Z_-]+$`)

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
		queue:    make(chan common.ChatData, 100),
		modqueue: make(chan common.ChatData, 100),
		clients:  make(map[string]*Client),
		tempConn: make(map[string]*chatConnection),
	}

	num, err := LoadEmotes()
	if err != nil {
		return nil, fmt.Errorf("error loading emotes: %s", err)
	}
	fmt.Printf("Loaded %d emotes\n", num)

	//the "heartbeat" for broadcasting messages
	go cr.BroadCast()
	return cr, nil
}

func LoadEmotes() (int, error) {
	newEmotes := map[string]string{}

	emotePNGs, err := filepath.Glob("./static/emotes/*.png")
	if err != nil {
		return 0, fmt.Errorf("unable to glob emote directory: %s\n", err)
	}

	emoteGIFs, err := filepath.Glob("./static/emotes/*.gif")
	if err != nil {
		return 0, fmt.Errorf("unable to glob emote directory: %s\n", err)
	}
	globbed_files := []string(emotePNGs)
	globbed_files = append(globbed_files, emoteGIFs...)

	fmt.Println("Loading emotes...")
	for _, file := range globbed_files {
		file = filepath.Base(file)
		key := file[0 : len(file)-4]
		newEmotes[key] = file
		fmt.Printf("%s ", key)
	}
	common.Emotes = newEmotes
	fmt.Println("")
	return len(common.Emotes), nil
}

func randomColor() string {
	nums := []int32{}
	for i := 0; i < 6; i++ {
		nums = append(nums, rand.Int31n(15))
	}
	return fmt.Sprintf("#%X%X%X%X%X%X",
		nums[0], nums[1], nums[2],
		nums[3], nums[4], nums[5])
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

	if len(name) < UsernameMinLength || len(name) > UsernameMaxLength || !re_username.MatchString(name) {
		return nil, UserFormatError{Name: name}
	}

	nameLower := strings.ToLower(name)
	for _, client := range cr.clients {
		if strings.ToLower(client.name) == nameLower {
			return nil, UserTakenError{Name: name}
		}
	}

	client := &Client{
		name:      name,
		conn:      conn,
		belongsTo: cr,
		color:     randomColor(),
	}

	host := client.Host()

	if banned, names := settings.IsBanned(host); banned {
		return nil, newBannedUserError(host, name, names)
	}

	cr.clients[uid] = client
	delete(cr.tempConn, uid)

	fmt.Printf("[join] %s %s\n", host, name)
	playingCommand, err := common.NewChatCommand(common.CmdPlaying, []string{cr.playing, cr.playingLink})()
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

	client, err := cr.getClient(name)
	if err != nil {
		fmt.Printf("[leave] Unable to get client for name %q\n", name)
		return
	}
	client.conn.Close()
	cr.delClient(name)

	cr.AddEventMsg(common.EvLeave, name, color)
	fmt.Printf("[leave] %s %s\n", client.Host(), client.name)
}

// kicked from the chatroom
func (cr *ChatRoom) Kick(name string) string {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock() //preventing simultaneous access to the `clients` map

	client, err := cr.getClient(name)
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
	cr.delClient(name)

	cr.AddEventMsg(common.EvKick, name, color)
	fmt.Printf("[kick] %s %s has been kicked\n", host, name)
	return ""
}

func (cr *ChatRoom) Ban(name string) string {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock()

	client, err := cr.getClient(name)
	if err != nil {
		fmt.Printf("[ban] Unable to get client for name %q\n", name)
		return "Cannot find that name"
	}

	names := []string{}
	host := client.Host()
	color := client.color

	client.conn.Close()
	cr.delClient(name)

	for name, c := range cr.clients {
		if c.Host() == host {
			names = append(names, name)
			client.conn.Close()
			cr.delClient(name)
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

	data, err := common.NewChatMessage(from.name, from.color, msg, lvl, t)()
	if err != nil {
		fmt.Printf("Error encoding chat message: %s", err)
		return
	}

	select {
	case cr.queue <- data:
	default:
		fmt.Println("Unable to queue chat message. Channel full.")
	}
}

func (cr *ChatRoom) AddCmdMsg(command common.CommandType, args []string) {
	data, err := common.NewChatCommand(command, args)()
	if err != nil {
		fmt.Printf("Error encoding command: %s", err)
		return
	}

	select {
	case cr.queue <- data:
	default:
		fmt.Println("Unable to queue command message.  Channel full.")
	}
}

func (cr *ChatRoom) AddModNotice(message string) {
	data, err := common.NewChatMessage("", "", message, common.CmdUser, common.MsgNotice)()
	if err != nil {
		fmt.Printf("Error encoding notice: %v", err)
		return
	}

	select {
	case cr.modqueue <- data:
	default:
		fmt.Println("Unable to queue notice.  Channel full.")
	}
}

func (cr *ChatRoom) AddEventMsg(event common.EventType, name, color string) {
	data, err := common.NewChatEvent(event, name, color)()

	if err != nil {
		fmt.Printf("Error encoding command: %s", err)
		return
	}

	select {
	case cr.queue <- data:
	default:
		fmt.Println("Unable to queue event message.  Channel full.")
	}
}

func (cr *ChatRoom) Unmod(name string) error {
	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock()

	client, err := cr.getClient(name)
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

	client, err := cr.getClient(name)
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

	client, err := cr.getClient(name)
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
func (cr *ChatRoom) BroadCast() {
	for {
		select {
		case msg := <-cr.queue:
			cr.clientsMtx.Lock()
			for _, client := range cr.clients {
				client.Send(msg)
			}
			for _, conn := range cr.tempConn {
				conn.WriteJSON(msg)
			}
			cr.clientsMtx.Unlock()
		case msg := <-cr.modqueue:
			cr.clientsMtx.Lock()
			for _, client := range cr.clients {
				if client.IsMod || client.IsAdmin {
					client.Send(msg)
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

func (cr *ChatRoom) delClient(name string) {
	delete(cr.clients, strings.ToLower(name))
}

func (cr *ChatRoom) getClient(name string) (*Client, error) {
	for _, client := range cr.clients {
		if client.name == name {
			return client, nil
		}
	}

	return nil, fmt.Errorf("client with that name not found")
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

	if !re_username.MatchString(newName) {
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
