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

	"github.com/gorilla/websocket"
	"github.com/zorchenhimer/MovieNight/common"
)

const (
	UsernameMaxLength int = 36
	UsernameMinLength int = 3
)

var re_username *regexp.Regexp = regexp.MustCompile(`^[0-9a-zA-Z_-]+$`)

type ChatRoom struct {
	clients     map[string]*Client // this needs to be a pointer.
	clientsMtx  sync.Mutex
	tempConn    map[string]*websocket.Conn
	queue       chan string
	playing     string
	playingLink string

	modPasswords    []string // single-use mod passwords
	modPasswordsMtx sync.Mutex
}

//initializing the chatroom
func newChatRoom() (*ChatRoom, error) {
	cr := &ChatRoom{
		queue:    make(chan string, 100),
		clients:  make(map[string]*Client),
		tempConn: make(map[string]*websocket.Conn),
	}

	num, err := LoadEmotes()
	if err != nil {
		return nil, fmt.Errorf("error loading emotes: %s", err)
	}
	fmt.Printf("Loaded %d emotes\n", num)

	//the "heartbeat" for broadcasting messages
	go func() {
		for {
			cr.BroadCast()
			time.Sleep(50 * time.Millisecond)
		}
	}()
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
	emotes = newEmotes
	fmt.Println("")
	return len(emotes), nil
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

func (cr *ChatRoom) JoinTemp(conn *websocket.Conn) (string, error) {
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
	conn, hasConn := cr.tempConn[uid]
	if !hasConn {
		return nil, errors.New("connection is missing from temp connections")
	}

	if len(name) < UsernameMinLength || len(name) > UsernameMaxLength || !re_username.MatchString(name) {
		return nil, UserFormatError{Name: name}
	}

	defer cr.clientsMtx.Unlock()
	cr.clientsMtx.Lock() //preventing simultaneous access to the `clients` map
	if _, exists := cr.clients[strings.ToLower(name)]; exists {
		return nil, UserTakenError{Name: name}
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

	cr.clients[strings.ToLower(name)] = client
	delete(cr.tempConn, uid)

	fmt.Printf("[join] %s %s\n", host, name)
	//client.Send(cr.GetPlayingString())
	playingCommand, err := common.EncodeCommand(common.CmdPlaying, []string{cr.playing, cr.playingLink})
	if err != nil {
		fmt.Printf("Unable to encode playing command on join: %s\n", err)
	} else {
		client.Send(playingCommand)
	}
	//cr.AddMsg(fmt.Sprintf("<i><b style=\"color:%s\">%s</b> has joined the chat.</i><br />", client.color, name))
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

	//cr.AddMsg(fmt.Sprintf("<i><b style=\"color:%s\">%s</b> has left the chat.</i><br />", color, name))
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

	//cr.AddMsg(fmt.Sprintf("<i><b>%s</b> has been kicked.</i><br />", name))
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
		//cr.AddMsg(fmt.Sprintf("<i><b>%s</b> has been kicked.</i><br />", name))
		cr.AddEventMsg(common.EvKick, name, color)
	} else {
		//cr.AddMsg(fmt.Sprintf("<i><b>%s</b> has been banned.</i><br />", name))
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

	data, err := common.EncodeMessage(
		from.name,
		from.color,
		msg,
		t)

	if err != nil {
		fmt.Printf("Error encoding chat message: %s", err)
		cr.queue <- msg
		return
	}

	select {
	case cr.queue <- data:
	default:
		fmt.Println("Unable to queue chat message. Channel full.")
	}
}

func (cr *ChatRoom) AddCmdMsg(command common.CommandType, args []string) {
	data, err := common.EncodeCommand(command, args)

	if err != nil {
		fmt.Printf("Error encoding command: %s", err)
		//cr.queue <- msg
		return
	}

	select {
	case cr.queue <- data:
	default:
		fmt.Println("Unable to queue command message.  Channel full.")
	}
}

func (cr *ChatRoom) AddEventMsg(event common.EventType, name, color string) {
	data, err := common.EncodeEvent(event, name, color)

	if err != nil {
		fmt.Printf("Error encoding command: %s", err)
		//cr.queue <- msg
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
	client.ServerMessage(`You have been unmodded.`)
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
	client.ServerMessage(`You have been modded.`)
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
	running := true
	for running {
		select {
		case msg := <-cr.queue:
			if len(msg) > 0 {
				cr.clientsMtx.Lock()
				for _, client := range cr.clients {
					client.Send(msg)
				}
				for _, conn := range cr.tempConn {
					connSend(msg, conn)
				}
				cr.clientsMtx.Unlock()
			}
		default:
			running = false
		}
	}
}

func (cr *ChatRoom) ClearPlaying() {
	cr.playing = ""
	cr.playingLink = ""
	//cr.AddCmdMsg(`<script>setPlaying("","");</script>`)
	cr.AddCmdMsg(common.CmdPlaying, []string{"", ""})
}

func (cr *ChatRoom) SetPlaying(title, link string) {
	cr.playing = title
	cr.playingLink = link
	//cr.AddCmdMsg(cr.GetPlayingString())
	cr.AddCmdMsg(common.CmdPlaying, []string{title, link})
}

//func (cr *ChatRoom) GetPlayingString() string {
//	return fmt.Sprintf(`<script>setPlaying("%s","%s");</script>`, cr.playing, cr.playingLink)
//}

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
	if client, ok := cr.clients[strings.ToLower(name)]; ok {
		return client, nil
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
