package common

import (
	"fmt"
	"strings"
)

var Emotes map[string]string

func ParseEmotesArray(words []string) []string {
	newWords := []string{}
	for _, word := range words {
		word = strings.Trim(word, "[]")

		found := false
		for key, val := range Emotes {
			if key == word {
				newWords = append(newWords, fmt.Sprintf("<img src=\"/emotes/%s\" title=\"%s\" />", val, key))
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
