package common

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type EmotesMap map[string]string

var Emotes EmotesMap

var reStripStatic = regexp.MustCompile(`^(\\|/)?static`)

func init() {
	Emotes = map[string]string{}
}

func (em EmotesMap) Add(fullpath string) {
	fullpath = reStripStatic.ReplaceAllLiteralString(fullpath, "")

	base := filepath.Base(fullpath)
	code := base[0 : len(base)-len(filepath.Ext(base))]

	_, exists := em[code]

	num := 0
	for exists {
		num += 1
		_, exists = em[fmt.Sprintf("%s-%d", code, num)]
	}

	if num > 0 {
		code = fmt.Sprintf("%s-%d", code, num)
	}

	Emotes[code] = fullpath
	fmt.Printf("Added emote %s at path %q\n", code, fullpath)
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
				newWords = append(newWords, EmoteToHtml(val, key))
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
