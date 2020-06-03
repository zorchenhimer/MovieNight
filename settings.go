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
var sstore *sessions.CookieStore

type Settings struct {
	// Non-Saved settings
	filename   string
	cmdLineKey string // stream key from the command line

	// Saved settings
	StreamStats     bool
	MaxMessageCount int
	TitleLength     int // maximum length of the title that can be set with the /playing
	PageTitle       string // primary value for the page <title> element
	AdminPassword   string
	RegenAdminPass  bool // regenerate admin password on start?
	StreamKey       string
	ListenAddress   string
	ApprovedEmotes  []string // list of channels that have been approved for emote use.  Global emotes are always "approved".
	TwitchClientID  string   // client id from twitch developers portal
	SessionKey      string   // key for session data
	Bans            []BanInfo
	LogLevel        common.LogLevel
	LogFile         string
	RoomAccess      AccessMode
	RoomAccessPin   string // The current pin
	NewPin          bool   // Auto generate a new pin on start.  Overwrites RoomAccessPin if set.

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

	if s.RegenAdminPass == true || s.AdminPassword == "" {
		s.AdminPassword, err = generatePass(time.Now().Unix())
		if err != nil {
			return nil, fmt.Errorf("unable to generate admin password: %s", err)
		}
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
	defer s.lock.Unlock()
	s.lock.Lock()

	return s.unlockedSave()
}

// unlockedSave expects the calling function to lock the RWMutex
func (s *Settings) unlockedSave() error {
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
	defer s.lock.Unlock()
	s.lock.Lock()

	if host == "127.0.0.1" {
		return fmt.Errorf("Cannot add a ban for localhost.")
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

func (s *Settings) SetTempKey(key string) {
	defer s.lock.Unlock()
	s.lock.Lock()

	s.cmdLineKey = key
}

func (s *Settings) GetStreamKey() string {
	defer s.lock.RUnlock()
	s.lock.RLock()

	if len(s.cmdLineKey) > 0 {
		return s.cmdLineKey
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

func (s *Settings) AddApprovedEmotes(channels []string) error {
	defer s.lock.Unlock()
	s.lock.Lock()

	approved := map[string]int{}
	for _, e := range s.ApprovedEmotes {
		approved[e] = 1
	}

	for _, name := range channels {
		approved[name] = 1
	}

	filtered := []string{}
	for key, _ := range approved {
		filtered = append(filtered, key)
	}

	s.ApprovedEmotes = filtered
	return s.unlockedSave()
}
