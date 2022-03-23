package common

import (
	"os"
	"testing"
)

var data_good = map[string]string{
	"one":   `<img src="/emotes/one.png" class="emote" title="one" />`,
	"two":   `<img src="/emotes/two.png" class="emote" title="two" />`,
	"three": `<img src="/emotes/three.gif" class="emote" title="three" />`,

	":one:":   `<img src="/emotes/one.png" class="emote" title="one" />`,
	":two:":   `<img src="/emotes/two.png" class="emote" title="two" />`,
	":three:": `<img src="/emotes/three.gif" class="emote" title="three" />`,

	":one::one:": `<img src="/emotes/one.png" class="emote" title="one" /><img src="/emotes/one.png" class="emote" title="one" />`,
	":one:one:":  `<img src="/emotes/one.png" class="emote" title="one" />one:`,
	"oneone":     "oneone",
	"one:one:":   `one<img src="/emotes/one.png" class="emote" title="one" />`,

	"[one]":   `<img src="/emotes/one.png" class="emote" title="one" />`,
	"[two]":   `<img src="/emotes/two.png" class="emote" title="two" />`,
	"[three]": `<img src="/emotes/three.gif" class="emote" title="three" />`,

	"[one][one]": `<img src="/emotes/one.png" class="emote" title="one" /><img src="/emotes/one.png" class="emote" title="one" />`,
	"[one]one":   `<img src="/emotes/one.png" class="emote" title="one" />one`,

	":one: two [three]": `<img src="/emotes/one.png" class="emote" title="one" /> <img src="/emotes/two.png" class="emote" title="two" /> <img src="/emotes/three.gif" class="emote" title="three" />`,

	"nope one what":     `nope <img src="/emotes/one.png" class="emote" title="one" /> what`,
	"nope :two: what":   `nope <img src="/emotes/two.png" class="emote" title="two" /> what`,
	"nope [three] what": `nope <img src="/emotes/three.gif" class="emote" title="three" /> what`,
}

func TestMain(m *testing.M) {
	Emotes = map[string]string{
		"one":   "/emotes/one.png",
		"two":   "/emotes/two.png",
		"three": "/emotes/three.gif",
	}
	os.Exit(m.Run())
}

func TestEmotes_ParseEmotes(t *testing.T) {
	for input, expected := range data_good {
		got := ParseEmotes(input)
		if got != expected {
			t.Errorf("%s failed to parse into %q. Received: %q", input, expected, got)
		}
	}
}
