package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"github.com/zorchenhimer/MovieNight/common"
)

var settings *Settings
var settingsMtx sync.Mutex
var sstore *sessions.CookieStore

type Settings struct {
	// Non-Saved settings
	filename   string
	cmdLineKey string // stream key from the command line

	// Saved settings
	StreamStats     bool
	MaxMessageCount int
	TitleLength     int // maximum length of the title that can be set with the /playing
	AdminPassword   string
	StreamKey       string
	ListenAddress   string
	ApprovedEmotes  []EmoteSet // list of channels that have been approved for emote use.  Global emotes are always "approved".
	SessionKey      string     // key for session data
	Bans            []BanInfo
	LogLevel        common.LogLevel
	LogFile         string
	RoomAccess      AccessMode
	RoomAccessPin   string // auto generate this,

	// Rate limiting stuff, in seconds
	RateLimitChat      time.Duration
	RateLimitNick      time.Duration
	RateLimitColor     time.Duration
	RateLimitAuth      time.Duration
	RateLimitDuplicate time.Duration // Amount of seconds between allowed duplicate messages

	// Send the NoCache header?
	NoCache bool
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

func LoadSettings(filename string) (*Settings, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %s", err)
	}

	var s *Settings
	err = json.Unmarshal(raw, &s)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling: %s", err)
	}
	s.filename = filename

	if err = common.SetupLogging(s.LogLevel, s.LogFile); err != nil {
		return nil, fmt.Errorf("Unable to setup logger: %s", err)
	}

	// have a default of 200
	if s.MaxMessageCount == 0 {
		s.MaxMessageCount = 300
	} else if s.MaxMessageCount < 0 {
		return s, fmt.Errorf("value for MaxMessageCount must be greater than 0, given %d", s.MaxMessageCount)
	}

	s.AdminPassword, err = generatePass(time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("unable to generate admin password: %s", err)
	}

	if s.RateLimitChat == -1 {
		s.RateLimitChat = 0
	} else if s.RateLimitChat <= 0 {
		s.RateLimitChat = 1
	}

	if s.RateLimitNick == -1 {
		s.RateLimitNick = 0
	} else if s.RateLimitNick <= 0 {
		s.RateLimitNick = 300
	}

	if s.RateLimitColor == -1 {
		s.RateLimitColor = 0
	} else if s.RateLimitColor <= 0 {
		s.RateLimitColor = 60
	}

	if s.RateLimitAuth == -1 {
		s.RateLimitAuth = 0
	} else if s.RateLimitAuth <= 0 {
		s.RateLimitAuth = 5
	}

	if s.RateLimitDuplicate == -1 {
		s.RateLimitDuplicate = 0
	} else if s.RateLimitDuplicate <= 0 {
		s.RateLimitDuplicate = 30
	}

	// Print this stuff before we multiply it by time.Second
	common.LogInfof("RateLimitChat: %v", s.RateLimitChat)
	common.LogInfof("RateLimitNick: %v", s.RateLimitNick)
	common.LogInfof("RateLimitColor: %v", s.RateLimitColor)
	common.LogInfof("RateLimitAuth: %v", s.RateLimitAuth)

	if len(settings.RoomAccess) == 0 {
		settings.RoomAccess = AccessOpen
	}

	if settings.RoomAccess != AccessOpen && len(settings.RoomAccessPin) == 0 {
		settings.RoomAccessPin = "1234"
	}

	// Don't use LogInfof() here.  Log isn't setup yet when LoadSettings() is called from init().
	fmt.Printf("Settings reloaded.  New admin password: %s\n", s.AdminPassword)

	if s.TitleLength <= 0 {
		s.TitleLength = 50
	}

	// Is this a good way to do this? Probably not...
	if len(settings.SessionKey) == 0 {
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
		settings.SessionKey = out
	}

	// Save admin password to file
	if err = settings.Save(); err != nil {
		return nil, fmt.Errorf("Unable to save settings: %s", err)
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
	marshaled, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return fmt.Errorf("error marshaling: %s", err)
	}

	err = ioutil.WriteFile(s.filename, marshaled, 0777)
	if err != nil {
		return fmt.Errorf("error saving: %s", err)
	}
	return nil
}

func (s *Settings) AddBan(host string, names []string) error {
	if host == "127.0.0.1" {
		return fmt.Errorf("Cannot add a ban for localhost.")
	}

	b := BanInfo{
		Names: names,
		IP:    host,
		When:  time.Now(),
	}
	settings.Bans = append(settings.Bans, b)

	common.LogInfof("[BAN] %q (%s) has been banned.\n", strings.Join(names, ", "), host)

	return settings.Save()
}

func (s *Settings) RemoveBan(name string) error {
	defer settingsMtx.Unlock()
	settingsMtx.Lock()

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
	return settings.Save()
}

func (s *Settings) IsBanned(host string) (bool, []string) {
	defer settingsMtx.Unlock()
	settingsMtx.Lock()

	for _, b := range s.Bans {
		if b.IP == host {
			return true, b.Names
		}
	}
	return false, nil
}

func (s *Settings) SetTempKey(key string) {
	defer settingsMtx.Unlock()
	settingsMtx.Lock()

	s.cmdLineKey = key
}

func (s *Settings) GetStreamKey() string {
	defer settingsMtx.Unlock()
	settingsMtx.Lock()

	if len(s.cmdLineKey) > 0 {
		return s.cmdLineKey
	}
	return s.StreamKey
}

func (s *Settings) generateNewPin() (string, error) {
	num, err := rand.Int(rand.Reader, big.NewInt(int64(9999)))
	if err != nil {
		return "", err
	}
	settings.RoomAccessPin = fmt.Sprintf("%04d", num)
	if err = s.Save(); err != nil {
		return "", err
	}
	return settings.RoomAccessPin, nil
}
