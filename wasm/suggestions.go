// +build js,wasm

package main

import (
	"strings"

	"syscall/js"

	"github.com/zorchenhimer/MovieNight/common"
)

const (
	keyTab          = 9
	keyEnter        = 13
	keyEsc          = 27
	keySpace        = 32
	keyUp           = 38
	keyDown         = 40
	suggestionName  = '@'
	suggestionEmote = ':'
)

var (
	currentSugType rune
	currentSug     string
	filteredSug    []string
	names          []string
	emoteNames     []string
	emotes         map[string]string
)

// The returned value is a bool deciding to prevent the event from propagating
func processMessageKey(this js.Value, v []js.Value) interface{} {
	startIdx := v[0].Get("target").Get("selectionStart").Int()
	keyCode := v[0].Get("keyCode").Int()
	ctrl := v[0].Get("ctrlKey").Bool()

	if ctrl && keyCode == keySpace {
		processMessage(nil)
		return true
	}

	if len(filteredSug) == 0 || currentSug == "" {
		return false
	}

	switch keyCode {
	case keyEsc:
		filteredSug = nil
		currentSug = ""
		currentSugType = 0
	case keyUp, keyDown:
		newidx := 0
		for i, n := range filteredSug {
			if n == currentSug {
				newidx = i
				if keyCode == keyDown {
					newidx = i + 1
					if newidx == len(filteredSug) {
						newidx--
					}
				} else if keyCode == keyUp {
					newidx = i - 1
					if newidx < 0 {
						newidx = 0
					}
				}
				break
			}
		}
		currentSug = filteredSug[newidx]
	case keyTab, keyEnter:
		msg := global.Get("msg")
		val := msg.Get("value").String()
		newval := val[:startIdx]
		wrap := string(suggestionEmote)

		if i := strings.LastIndex(newval, string(currentSugType)); i != -1 {
			var offset int
			if currentSugType == suggestionName {
				offset = 1
				wrap = ""
			}

			newval = newval[:i+offset] + wrap + currentSug + wrap
		}

		endVal := val[startIdx:]
		if len(val) == startIdx || val[startIdx:][0] != ' ' {
			// insert a space into val so selection indexing can be one line
			endVal = " " + endVal
		}
		msg.Set("value", newval+endVal)
		msg.Set("selectionStart", len(newval)+1)
		msg.Set("selectionEnd", len(newval)+1)

		// Clear out filtered names since it is no longer needed
		filteredSug = nil
	default:
		// We only want to handle the caught keys, so return early
		return false
	}

	updateSuggestionDiv()
	return true
}

func processMessage(v []js.Value) {
	msg := global.Get("msg")
	text := strings.ToLower(msg.Get("value").String())
	startIdx := msg.Get("selectionStart").Int()

	filteredSug = nil
	if len(text) != 0 {
		if len(names) > 0 || len(emoteNames) > 0 {
			var caretIdx int
			textParts := strings.Split(text, " ")

			for i, word := range textParts {
				// Increase caret index at beginning if not first word to account for spaces
				if i != 0 {
					caretIdx++
				}

				// It is possible to have a double space "  ", which will lead to an
				// empty string element in the slice. Also check that the index of the
				// cursor is between the start of the word and the end
				if len(word) > 0 && caretIdx <= startIdx && startIdx <= caretIdx+len(word) {
					var suggestions []string
					if word[0] == suggestionName {
						currentSugType = suggestionName
						suggestions = names
					} else if word[0] == suggestionEmote {
						suggestions = emoteNames
						currentSugType = suggestionEmote
					}

					for _, s := range suggestions {
						if len(word) == 1 || strings.Contains(strings.ToLower(s), word[1:]) {
							filteredSug = append(filteredSug, s)
						}
					}
				}

				if len(filteredSug) > 0 {
					currentSug = ""
					break
				}

				caretIdx += len(word)
			}
		}
	}

	updateSuggestionDiv()
}

func updateSuggestionDiv() {
	const selectedClass = ` class="selectedName"`

	var divs []string
	if len(filteredSug) > 0 {
		// set current name to first if not set already
		if currentSug == "" {
			currentSug = filteredSug[len(filteredSug)-1]
		}

		var hascurrentSuggestion bool
		divs = make([]string, len(filteredSug))

		// Create inner body of html
		for i := range filteredSug {
			divs[i] = "<div"

			sug := filteredSug[i]
			if sug == currentSug {
				hascurrentSuggestion = true
				divs[i] += selectedClass
			}
			divs[i] += ">"

			if currentSugType == suggestionEmote {
				divs[i] += common.EmoteToHtml(emotes[sug], sug)
			}

			divs[i] += sug + "</div>"
		}

		if !hascurrentSuggestion {
			divs[0] = divs[0][:4] + selectedClass + divs[0][4:]
		}
	}
	// The \n is so it's easier to read th source in web browsers for the dev
	global.Get("suggestions").Set("innerHTML", strings.Join(divs, "\n"))
	global.Call("updateSuggestionScroll")
}
