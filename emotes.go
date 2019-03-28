package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type twitchChannel struct {
	ChannelName     string            `json:"channel_name"`
	DisplayName     string            `json:"display_name"`
	ChannelId       string            `json:"channel_id"`
	BroadcasterType string            `json:"broadcaster_type"`
	Plans           map[string]string `json:"plans"`
	Emotes          []struct {
		Code string `json:"code"`
		Set  int    `json:"emoticon_set"`
		Id   int    `json:"id"`
	} `json:"emotes"`
	BaseSetId   string `json:"base_set_id"`
	GeneratedAt string `json:"generated_at"`
}

// Used in settings
type EmoteSet struct {
	Channel string // channel name
	Prefix  string // emote prefix
	Found   bool   `json:"-"`
}

const subscriberJson string = `subscribers.json`

// Download a single channel's emote set
func (tc *twitchChannel) downloadEmotes() (*EmoteSet, error) {
	es := &EmoteSet{Channel: strings.ToLower(tc.ChannelName)}
	for _, emote := range tc.Emotes {
		url := fmt.Sprintf(`https://static-cdn.jtvnw.net/emoticons/v1/%d/1.0`, emote.Id)
		png := `static/emotes/` + emote.Code + `.png`

		if len(es.Prefix) == 0 {
			// For each letter
			for i := 0; i < len(emote.Code); i++ {
				// Find the first capital
				b := emote.Code[i]
				if b >= 'A' && b <= 'Z' {
					es.Prefix = emote.Code[0 : i-1]
					fmt.Printf("Found prefix for channel %q: %q (%q)\n", es.Channel, es.Prefix, emote)
					break
				}
			}
		}

		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		f, err := os.Create(png)
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(f, resp.Body)
		if err != nil {
			return nil, err
		}
	}

	return es, nil
}

func GetEmotes(names []string) ([]*EmoteSet, error) {
	// Do this up-front
	for i := 0; i < len(names); i++ {
		names[i] = strings.ToLower(names[i])
	}

	channels, err := findChannels(names)
	if err != nil {
		return nil, fmt.Errorf("Error reading %q: %v", subscriberJson, err)
	}

	emoteSets := []*EmoteSet{}
	for _, c := range channels {
		es, err := c.downloadEmotes()
		if err != nil {
			return nil, fmt.Errorf("Error downloading emotes: %v", err)
		}
		emoteSets = append(emoteSets, es)
	}

	for _, es := range emoteSets {
		found := false
		for _, name := range names {
			if es.Channel == name {
				found = true
				break
			}
		}
		if !found {
			es.Found = false
		}
	}

	return emoteSets, nil
}

func findChannels(names []string) ([]twitchChannel, error) {
	file, err := os.Open(subscriberJson)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data := []twitchChannel{}
	dec := json.NewDecoder(file)

	// Open bracket
	_, err = dec.Token()
	if err != nil {
		return nil, err
	}

	done := false
	for dec.More() && !done {
		// opening bracket of channel
		_, err = dec.Token()
		if err != nil {
			return nil, err
		}

		// Decode the channel stuff
		var c twitchChannel
		err = dec.Decode(&c)
		if err != nil {
			return nil, err
		}

		// Is this a channel we are looking for?
		found := false
		for _, search := range names {
			if strings.ToLower(c.ChannelName) == search {
				found = true
				break
			}
		}

		// Yes it is.  Add it to the data
		if found {
			data = append(data, c)
		}

		// Check for completion.  Don't bother parsing the rest of
		// the json file if we've already found everything that we're
		// looking for.
		if len(data) == len(names) {
			done = true
		}
	}

	return data, nil
}
