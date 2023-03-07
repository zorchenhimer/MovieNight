package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"github.com/zorchenhimer/MovieNight/common"
	"github.com/zorchenhimer/MovieNight/files"
)

var settings *Settings
var sstore *sessions.CookieStore

type Settings struct {
	// Non-Saved settings
	filename string
	// the argument from the command line
	args args

	// Saved settings
	AdminPassword     string
	Bans              []BanInfo
	LetThemLurk       bool // whether or not to announce users joining/leaving chat
	ListenAddress     string
	LogFile           string
	LogLevel          common.LogLevel
	MaxMessageCount   int
	NewPin            bool   // Auto generate a new pin on start.  Overwrites RoomAccessPin if set.
	PageTitle         string // primary value for the page <title> element
	RegenAdminPass    bool   // regenerate admin password on start?
	RoomAccess        AccessMode
	RoomAccessPin     string // The current pin
	RtmpListenAddress string // host:port that the RTMP server listens on
	SessionKey        string // key for session data
	StreamKey         string
	StreamStats       bool
	TitleLength       int  // maximum length of the title that can be set with the /playing
	WrappedEmotesOnly bool // only allow "wrapped" emotes.  eg :Kappa: and [Kappa] but not Kappa

	// Rate limiting stuff, in seconds
	RateLimitChat      time.Duration
	RateLimitNick      time.Duration
	RateLimitColor     time.Duration
	RateLimitAuth      time.Duration
	RateLimitDuplicate time.Duration // Amount of seconds between allowed duplicate messages

	// Send the NoCache header?
	NoCache bool

	lock sync.RWMutex
}

type AccessMode string

const (
	AccessOpen    AccessMode = "open"
	AccessPin     AccessMode = "pin"
	AccessRequest AccessMode = "request"
)

type BanInfo struct {
	IP    string
	Names []string
	When  time.Time
}

//go:embed settings_example.json
var settingsExampleFS []byte

func LoadSettings(args args) (*Settings, error) {
	var filename = files.JoinRunPath(args.ConfigFile)
	raw, err := os.ReadFile(filename)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("could not read settings file: %w", err)
		}
		raw = settingsExampleFS
	}

	var s *Settings
	err = json.Unmarshal(raw, &s)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling: %w", err)
	}
	s.filename = filename
	s.args = args // Keep an eye on the args from the command line

	var logFileDir string = s.LogFile
	fmt.Printf("Log file: %s\n", logFileDir)
	if err = common.SetupLogging(s.LogLevel, logFileDir); err != nil {
		return nil, fmt.Errorf("unable to setup logger: %w", err)
	}

	// have a default of 200
	if s.MaxMessageCount == 0 {
		s.MaxMessageCount = 300
	} else if s.MaxMessageCount < 0 {
		return s, fmt.Errorf("value for MaxMessageCount must be greater than 0, given %d", s.MaxMessageCount)
	}

	if s.RegenAdminPass || s.AdminPassword == "" {
		s.AdminPassword, err = generatePass(time.Now().Unix())
		if err != nil {
			return nil, fmt.Errorf("unable to generate admin password: %w", err)
		}
	}

	// Set to -1 to reset
	if s.RateLimitChat == -1 {
		s.RateLimitChat = 1
	} else if s.RateLimitChat < 0 {
		s.RateLimitChat = 0
	}

	if s.RateLimitNick == -1 {
		s.RateLimitNick = 300
	} else if s.RateLimitNick < 0 {
		s.RateLimitNick = 0
	}

	if s.RateLimitColor == -1 {
		s.RateLimitColor = 60
	} else if s.RateLimitColor < 0 {
		s.RateLimitColor = 0
	}

	if s.RateLimitAuth == -1 {
		s.RateLimitAuth = 5
	} else if s.RateLimitAuth < 0 {
		common.LogInfoln("It's not recommended to disable the authentication rate limit.")
		s.RateLimitAuth = 0
	}

	if s.RateLimitDuplicate == -1 {
		s.RateLimitDuplicate = 30
	} else if s.RateLimitDuplicate < 0 {
		s.RateLimitDuplicate = 0
	}

	if s.WrappedEmotesOnly {
		common.LogInfoln("Only allowing wrapped emotes")
		common.WrappedEmotesOnly = true
	}

	// Print this stuff before we multiply it by time.Second
	common.LogInfof("RateLimitChat: %v", s.RateLimitChat)
	common.LogInfof("RateLimitNick: %v", s.RateLimitNick)
	common.LogInfof("RateLimitColor: %v", s.RateLimitColor)
	common.LogInfof("RateLimitAuth: %v", s.RateLimitAuth)

	if len(s.RoomAccess) == 0 {
		s.RoomAccess = AccessOpen
	}

	if (s.RoomAccess != AccessOpen && len(s.RoomAccessPin) == 0) || s.NewPin {
		pin, err := s.generateNewPin()
		if err != nil {
			common.LogErrorf("Unable to generate new pin: %v", err)
		}
		common.LogInfof("New pin generated: %s", pin)
	}

	// Don't use LogInfof() here.  Log isn't setup yet when LoadSettings() is called from init().
	fmt.Printf("Settings reloaded.  New admin password: %s\n", s.AdminPassword)

	if s.TitleLength <= 0 {
		s.TitleLength = 50
	}

	// Is this a good way to do this? Probably not...
	if len(s.SessionKey) == 0 {
		out := ""
		large := big.NewInt(int64(1 << 60))
		large = large.Add(large, large)
		for len(out) < 50 {
			num, err := rand.Int(rand.Reader, large)
			if err != nil {
				panic("Error generating session key: " + err.Error())
			}
			out = fmt.Sprintf("%s%X", out, num)
		}
		s.SessionKey = out
	}

	// Save admin password to file
	if err = s.Save(); err != nil {
		return nil, fmt.Errorf("unable to save settings: %w", err)
	}

	return s, nil
}

func generatePass(seed int64) (string, error) {
	out := ""
	for len(out) < 20 {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(15)))
		if err != nil {
			return "", err
		}

		out = fmt.Sprintf("%s%X", out, num)
	}
	return out, nil
}

func (s *Settings) Save() error {
	defer s.lock.Unlock()
	s.lock.Lock()

	return s.unlockedSave()
}

// unlockedSave expects the calling function to lock the RWMutex
func (s *Settings) unlockedSave() error {
	marshaled, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return fmt.Errorf("error marshaling: %w", err)
	}

	err = os.WriteFile(s.filename, marshaled, 0777)
	if err != nil {
		return fmt.Errorf("error saving: %w", err)
	}
	return nil
}

func (s *Settings) AddBan(host string, names []string) error {
	defer s.lock.Unlock()
	s.lock.Lock()

	if host == "127.0.0.1" {
		return fmt.Errorf("cannot add a ban for localhost")
	}

	b := BanInfo{
		Names: names,
		IP:    host,
		When:  time.Now(),
	}
	s.Bans = append(s.Bans, b)

	common.LogInfof("[BAN] %q (%s) has been banned.\n", strings.Join(names, ", "), host)

	return s.unlockedSave()
}

func (s *Settings) RemoveBan(name string) error {
	defer s.lock.Unlock()
	s.lock.Lock()

	name = strings.ToLower(name)
	newBans := []BanInfo{}
	for _, b := range s.Bans {
		for _, n := range b.Names {
			if n == name {
				common.LogInfof("[ban] Removed ban for %s [%s]\n", b.IP, n)
			} else {
				newBans = append(newBans, b)
			}
		}
	}
	s.Bans = newBans
	return s.unlockedSave()
}

func (s *Settings) IsBanned(host string) (bool, []string) {
	defer s.lock.RUnlock()
	s.lock.RLock()

	for _, b := range s.Bans {
		if b.IP == host {
			return true, b.Names
		}
	}
	return false, nil
}

func (s *Settings) GetStreamKey() string {
	defer s.lock.RUnlock()
	s.lock.RLock()

	if s.args.StreamKey != "" {
		return s.args.StreamKey
	}

	if s.StreamKey == "" {
		panic("Missing stream key, add it in settings.json, or use --streamKey at runtime")
	}

	return s.StreamKey
}

func (s *Settings) generateNewPin() (string, error) {
	defer s.lock.Unlock()
	s.lock.Lock()

	num, err := rand.Int(rand.Reader, big.NewInt(int64(9999)))
	if err != nil {
		return "", err
	}
	s.RoomAccessPin = fmt.Sprintf("%04d", num)
	if err = s.unlockedSave(); err != nil {
		return "", err
	}
	return s.RoomAccessPin, nil
}

func (s *Settings) GetAdminPassword() string {
	if s.args.AdminPass != "" {
		common.LogInfoln("Password provided at runtime; ignoring password in set in settings.")
		return s.args.AdminPass
	}
	return s.AdminPassword
}

func (s *Settings) GetAddr() string {
	if s.args.Addr != "" {
		return s.args.Addr
	}
	return s.ListenAddress
}

func (s *Settings) GetRtmpAddr() string {
	if s.args.RtmpAddr != "" {
		return s.args.RtmpAddr
	}
	return s.RtmpListenAddress
}

func (s *Settings) GetRoomAccess() AccessMode {
	return s.RoomAccess
}

func (s *Settings) GetRoomAccessPin() string {
	return s.RoomAccessPin
}

func (s *Settings) GetEmotesDir() string {
	return files.JoinRunPath(s.args.EmotesDir)
}

func (s *Settings) GetStaticDir() string {
	return s.args.StaticDir
}

func (s *Settings) GetStaticFsys() files.FileSystem {
	fSys, err := files.FS(staticFS, settings.GetStaticDir(), "static")
	if err != nil {
		log.Fatalf("Error creating static FS: %v\n", err)
		return nil
	}

	return fSys
}

func (s *Settings) GetWriteStatic() bool {
	return s.args.WriteStatic
}
