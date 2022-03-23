package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/zorchenhimer/MovieNight/common"
)

const emoteDir = "/static/emotes/"

type TwitchUser struct {
	ID    string
	Login string
}

type EmoteInfo struct {
	ID   int
	Code string
}

func loadEmotes() error {
	newEmotes, err := processEmoteDir(common.RunPath() + emoteDir)
	if err != nil {
		return err
	}

	common.Emotes = newEmotes

	return nil
}

func processEmoteDir(path string) (common.EmotesMap, error) {
	dirInfo, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("could not open emoteDir: %w", err)
	}

	subDirs := []string{}

	for _, item := range dirInfo {
		// Get first level subdirs (eg, "twitch", "discord", etc)
		if item.IsDir() {
			subDirs = append(subDirs, item.Name())
			continue
		}
	}

	em := common.NewEmotesMap()
	// Find top level emotes
	em, err = findEmotes(path, em)
	if err != nil {
		return nil, fmt.Errorf("could not findEmotes() in top level directory: %w", err)
	}

	// Get second level subdirs (eg, "twitch", "zorchenhimer", etc)
	for _, dir := range subDirs {
		subd, err := os.ReadDir(filepath.Join(path, dir))
		if err != nil {
			fmt.Printf("Error reading dir %q: %v\n", subd, err)
			continue
		}
		for _, d := range subd {
			if d.IsDir() {
				p := filepath.Join(path, dir, d.Name())
				em, err = findEmotes(p, em)
				if err != nil {
					fmt.Printf("Error finding emotes in %q: %v\n", p, err)
				}
			}
		}
	}

	common.LogInfof("processEmoteDir: %d\n", len(em))
	return em, nil
}

func findEmotes(dir string, em common.EmotesMap) (common.EmotesMap, error) {
	var runPathLength = len(common.RunPath() + "/static/")

	common.LogDebugf("finding emotes in %q\n", dir)
	emotePNGs, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil {
		return em, fmt.Errorf("unable to glob emote directory: %w", err)
	}
	common.LogInfof("Found %d emotePNGs\n", len(emotePNGs))

	emoteGIFs, err := filepath.Glob(filepath.Join(dir, "*.gif"))
	if err != nil {
		return em, fmt.Errorf("unable to glob emote directory: %w", err)
	}
	common.LogInfof("Found %d emoteGIFs\n", len(emoteGIFs))

	for _, file := range emotePNGs {
		png := strings.ReplaceAll(common.Substr(file, runPathLength, len(file)), "\\", "/")
		//common.LogDebugf("Emote PNG: %s", png)
		em = em.Add(png)
	}

	for _, file := range emoteGIFs {
		gif := strings.ReplaceAll(common.Substr(file, runPathLength, len(file)), "\\", "/")
		//common.LogDebugf("Emote GIF: %s", gif)
		em = em.Add(gif)
	}

	return em, nil
}

func getEmotes(names []string) error {
	users := getUserIDs(names)
	users = append(users, TwitchUser{ID: "0", Login: "twitch"})

	for _, user := range users {
		emotes, cheers, err := getChannelEmotes(user.ID)
		if err != nil {
			return fmt.Errorf("could not get emote data for %#v: %w", user.ID, err)
		}

		emoteUserDir := filepath.Join(common.RunPath()+emoteDir, "twitch", user.Login)
		if _, err := os.Stat(emoteUserDir); os.IsNotExist(err) {
			err = os.MkdirAll(emoteUserDir, os.ModePerm)
			if err != nil {
				return fmt.Errorf("could not make emote dir %#v: %w", emoteUserDir, err)
			}
		}

		for _, emote := range emotes {
			if !strings.ContainsAny(emote.Code, `:;\[]|?&`) {
				filePath := filepath.Join(emoteUserDir, emote.Code+".png")
				file, err := os.Create(filePath)
				if err != nil {

					return fmt.Errorf("could not create emote file in path %#v: %w", filePath, err)
				}

				err = downloadEmote(emote.ID, file)
				if err != nil {
					return fmt.Errorf("could not download emote %s: %w", emote.Code, err)
				}
			}
		}

		for amount, sizes := range cheers {
			name := fmt.Sprintf("%sCheer%s.gif", user.Login, amount)
			filePath := filepath.Join(emoteUserDir, name)
			file, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("could not create emote file in path %#v: %w", filePath, err)
			}

			err = downloadCheerEmote(sizes["4"], file)
			if err != nil {
				return fmt.Errorf("could not download emote %s: %w", name, err)
			}
		}
	}
	return nil
}

func getUserIDs(names []string) []TwitchUser {
	logins := strings.Join(names, "&login=")
	request, err := http.NewRequest("GET", fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", logins), nil)
	if err != nil {
		log.Fatalln("Error generating new request:", err)
	}
	request.Header.Set("Client-ID", settings.TwitchClientID)
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.TwitchClientSecret))

	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Fatalln("Error sending request:", err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	type userResponse struct {
		Data []TwitchUser
	}
	var data userResponse

	err = decoder.Decode(&data)
	if err != nil {
		log.Fatalln("Error decoding data:", err)
	}

	return data.Data
}

func getChannelEmotes(ID string) ([]EmoteInfo, map[string]map[string]string, error) {
	resp, err := http.Get("https://api.twitchemotes.com/api/v4/channels/" + ID)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get emotes %w", err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	type EmoteResponse struct {
		Emotes     []EmoteInfo
		Cheermotes map[string]map[string]string
	}
	var data EmoteResponse

	err = decoder.Decode(&data)
	if err != nil {
		return nil, nil, fmt.Errorf("could not decode emotes %w", err)
	}

	return data.Emotes, data.Cheermotes, nil
}

func downloadEmote(ID int, file *os.File) error {
	resp, err := http.Get(fmt.Sprintf("https://static-cdn.jtvnw.net/emoticons/v1/%d/3.0", ID))
	if err != nil {
		return fmt.Errorf("could not download emote file %s: %w", file.Name(), err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("could not save emote: %w", err)
	}
	return nil
}

func downloadCheerEmote(url string, file *os.File) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("could not download cheer file %s: %w", file.Name(), err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("could not save cheer: %w", err)
	}
	return nil
}
