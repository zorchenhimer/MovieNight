package main

import (
	"fmt"

	//"html"
	"math/rand"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	UsernameMaxLength int = 36
	UsernameMinLength int = 3
)

var re_username *regexp.Regexp = regexp.MustCompile(`^[0-9a-zA-Z_-]+$`)

type ChatRoom struct {
	clients     map[string]*Client // this needs to be a pointer.
	clientsMtx  sync.Mutex
	queue       chan string
	playing     string
	playingLink string
}

//initializing the chatroom
func (cr *ChatRoom) Init() error {
	cr.queue = make(chan string, 5)
	cr.clients = make(map[string]*Client)

	num, err := LoadEmotes()
	if err != nil {
		return fmt.Errorf("Error loading emotes: %s", err)
	}
	fmt.Printf("Loaded %d emotes\n", num)

	//the "heartbeat" for broadcasting messages
	go func() {
		for {
			cr.BroadCast()
			time.Sleep(100 * time.Millisecond)
		}
	}()
	return nil
}

func LoadEmotes() (int, error) {
	newEmotes := map[string]string{}

	emotePNGs, err := filepath.Glob("./static/emotes/*.png")
	if err != nil {
		return 0, fmt.Errorf("Unable to glob emote directory: %s\n", err)
	}

	emoteGIFs, err := filepath.Glob("./static/emotes/*.gif")
	if err != nil {
		return 0, fmt.Errorf("Unable to glob emote directory: %s\n", err)
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

//registering a new client
//returns pointer to a Client, or Nil, if the name is already taken
func (cr *ChatRoom) Join(name string, conn *websocket.Conn) (*Client, error) {

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

	fmt.Printf("[join] %s %s\n", host, name)
	client.Send(cr.GetPlayingString())
	cr.AddMsg(fmt.Sprintf("<i><b style=\"color:%s\">%s</b> has joined the chat.</i><br />", client.color, name))
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

	cr.AddMsg(fmt.Sprintf("<i><b style=\"color:%s\">%s</b> has left the chat.</i><br />", color, name))
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

	host := client.Host()
	client.conn.Close()
	cr.delClient(name)

	cr.AddMsg(fmt.Sprintf("<i><b>%s</b> has been kicked.</i><br />", name))
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
		cr.AddMsg(fmt.Sprintf("<i><b>%s</b> has been kicked.</i><br />", name))
	} else {
		cr.AddMsg(fmt.Sprintf("<i><b>%s</b> has been banned.</i><br />", name))
	}
	return ""
}

//adding message to queue
func (cr *ChatRoom) AddMsg(msg string) {
	cr.queue <- msg
}

func (cr *ChatRoom) AddCmdMsg(msg string) {
	cr.queue <- msg
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
	msgBlock := ""
infLoop:
	for {
		select {
		case m := <-cr.queue:
			msgBlock += m // + "<br />"
		default:
			break infLoop
		}
	}
	if len(msgBlock) > 0 {
		for _, client := range cr.clients {
			client.Send(msgBlock)
		}
	}
}

func (cr *ChatRoom) ClearPlaying() {
	cr.playing = ""
	cr.playingLink = ""
	cr.AddCmdMsg(`<script>setPlaying("","");</script>`)
}

func (cr *ChatRoom) SetPlaying(title, link string) {
	cr.playing = title
	cr.playingLink = link
	cr.AddCmdMsg(cr.GetPlayingString())
}

func (cr *ChatRoom) GetPlayingString() string {
	return fmt.Sprintf(`<script>setPlaying("%s","%s");</script>`, cr.playing, cr.playingLink)
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
	if client, ok := cr.clients[strings.ToLower(name)]; ok {
		return client, nil
	}
	return nil, fmt.Errorf("Client with that name not found.")
}
