package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
)

const emoteDir = "./static/emotes/"

var Emotes map[string]Emote

type Emote struct {
	Dir  string
	File string
}

func (e Emote) path() string {
	return path.Join(e.Dir, e.File)
}

type TwitchUser struct {
	ID    string
	Login string
}

type EmoteInfo struct {
	ID   int
	Code string
}

// func loadEmotes() error {
// 	newEmotes := map[string]string{}

// 	emotePNGs, err := filepath.Glob("./static/emotes/*.png")
// 	if err != nil {
// 		return 0, fmt.Errorf("unable to glob emote directory: %s\n", err)
// 	}

// 	emoteGIFs, err := filepath.Glob("./static/emotes/*.gif")
// 	if err != nil {
// 		return 0, fmt.Errorf("unable to glob emote directory: %s\n", err)
// 	}
// 	globbed_files := []string(emotePNGs)
// 	globbed_files = append(globbed_files, emoteGIFs...)

// 	LogInfoln("Loading emotes...")
// 	emInfo := []string{}
// 	for _, file := range globbed_files {
// 		file = filepath.Base(file)
// 		key := file[0 : len(file)-4]
// 		newEmotes[key] = file
// 		emInfo = append(emInfo, key)
// 	}
// 	Emotes = newEmotes
// 	LogInfoln(strings.Join(emInfo, " "))
// 	return len(Emotes), nil
// }

func loadEmotes() error {
	fmt.Println(processEmoteDir(emoteDir))
	return nil
}

func processEmoteDir(path string) ([]Emote, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not open emoteDir:")
	}

	files, err := dir.Readdir(0)
	if err != nil {
		return nil, errors.Wrap(err, "could not get files:")
	}

	var emotes []Emote
	for _, file := range files {
		emotes = append(emotes, Emote{Dir: path, File: file.Name()})
	}

	subdir, err := dir.Readdirnames(0)
	if err != nil {
		return nil, errors.Wrap(err, "could not get sub directories:")
	}

	for _, d := range subdir {
		subEmotes, err := processEmoteDir(d)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process sub directory \"%s\":", d)
		}
		emotes = append(emotes, subEmotes...)
	}

	return emotes, nil
}

func getEmotes(names []string) error {
	users := getUserIDs(names)
	users = append(users, TwitchUser{ID: "0", Login: "global"})

	for _, user := range users {
		emotes, cheers, err := getChannelEmotes(user.ID)
		if err != nil {
			return errors.Wrapf(err, "could not get emote data for \"%s\"", user.ID)
		}

		emoteUserDir := path.Join(emoteDir, "twitch", user.Login)
		if _, err := os.Stat(emoteUserDir); os.IsNotExist(err) {
			os.MkdirAll(emoteUserDir, os.ModePerm)
		}

		for _, emote := range emotes {
			if !strings.ContainsAny(emote.Code, `:;\[]|?&`) {
				filePath := path.Join(emoteUserDir, emote.Code+".png")
				file, err := os.Create(filePath)
				if err != nil {

					return errors.Wrapf(err, "could not create emote file in path \"%s\":", filePath)
				}

				err = downloadEmote(emote.ID, file)
				if err != nil {
					return errors.Wrapf(err, "could not download emote %s:", emote.Code)
				}
			}
		}

		for amount, sizes := range cheers {
			name := fmt.Sprintf("%sCheer%s.gif", user.Login, amount)
			filePath := path.Join(emoteUserDir, name)
			file, err := os.Create(filePath)
			if err != nil {
				return errors.Wrapf(err, "could not create emote file in path \"%s\":", filePath)
			}

			err = downloadCheerEmote(sizes["4"], file)
			if err != nil {
				return errors.Wrapf(err, "could not download emote %s:", name)
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

	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Fatalln("Error sending request:", err)
	}

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
		return nil, nil, errors.Wrap(err, "could not get emotes")
	}
	decoder := json.NewDecoder(resp.Body)

	type EmoteResponse struct {
		Emotes     []EmoteInfo
		Cheermotes map[string]map[string]string
	}
	var data EmoteResponse

	err = decoder.Decode(&data)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not decode emotes")
	}

	return data.Emotes, data.Cheermotes, nil
}

func downloadEmote(ID int, file *os.File) error {
	resp, err := http.Get(fmt.Sprintf("https://static-cdn.jtvnw.net/emoticons/v1/%d/3.0", ID))
	if err != nil {
		return errors.Errorf("could not download emote file %s: %v", file.Name(), err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return errors.Errorf("could not save emote: %v", err)
	}
	return nil
}

func downloadCheerEmote(url string, file *os.File) error {
	resp, err := http.Get(url)
	if err != nil {
		return errors.Errorf("could not download cheer file %s: %v", file.Name(), err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return errors.Errorf("could not save cheer: %v", err)
	}
	return nil
}
