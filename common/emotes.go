package common

import (
	"fmt"
	"path"
	"strings"
)

var Emotes map[string]EmotePath

type EmotePath struct {
	Dir  string
	File string
}

func (e EmotePath) path() string {
	return path.Join(e.Dir, e.File)
}

func EmoteToHtml(file, title string) string {
	return fmt.Sprintf(`<img src="%s" height="28px" title="%s" />`, file, title)
}

func ParseEmotesArray(words []string) []string {
	newWords := []string{}
	for _, word := range words {
		// make :emote: and [emote] valid for replacement.
		wordTrimmed := strings.Trim(word, ":[]")

		found := false
		for key, val := range Emotes {
			if key == wordTrimmed {
				newWords = append(newWords, EmoteToHtml(val.File, key))
				found = true
			}
		}
		if !found {
			newWords = append(newWords, word)
		}
	}
	return newWords
}

func ParseEmotes(msg string) string {
	words := ParseEmotesArray(strings.Split(msg, " "))
	return strings.Join(words, " ")
}
