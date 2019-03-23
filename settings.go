package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/sessions"
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
	Bans            []BanInfo
	StreamKey       string
	ListenAddress   string
	SessionKey      string // key for session data
	RoomAccess      AccessMode
	RoomAccessPin   string // auto generate this,
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

func init() {
	var err error
	settings, err = LoadSettings("settings.json")
	if err != nil {
		panic("Unable to load settings: " + err.Error())
	}
	if len(settings.StreamKey) == 0 {
		panic("Missing stream key is settings.json")
	}

	if settings.TitleLength <= 0 {
		settings.TitleLength = 50
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

	if len(settings.RoomAccess) == 0 {
		settings.RoomAccess = AccessOpen
	}

	if settings.RoomAccess != AccessOpen && len(settings.RoomAccessPin) == 0 {
		settings.RoomAccessPin = "1234"
	}

	sstore = sessions.NewCookieStore([]byte(settings.SessionKey))
	sstore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 24, // one day
		SameSite: http.SameSiteStrictMode,
	}

	// Save admin password to file
	if err = settings.Save(); err != nil {
		panic("Unable to save settings: " + err.Error())
	}
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
	fmt.Printf("Settings reloaded.  New admin password: %s\n", s.AdminPassword)

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

	fmt.Printf("[BAN] %q (%s) has been banned.\n", strings.Join(names, ", "), host)

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
				fmt.Printf("[ban] Removed ban for %s [%s]\n", b.IP, n)
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
