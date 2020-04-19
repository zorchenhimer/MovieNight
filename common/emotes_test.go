package common

import (
	"os"
	"testing"
)

var data_good = map[string]string{
	"one":   `<img src="/emotes/one.png" height="28px" title="one" />`,
	"two":   `<img src="/emotes/two.png" height="28px" title="two" />`,
	"three": `<img src="/emotes/three.gif" height="28px" title="three" />`,

	":one:":   `<img src="/emotes/one.png" height="28px" title="one" />`,
	":two:":   `<img src="/emotes/two.png" height="28px" title="two" />`,
	":three:": `<img src="/emotes/three.gif" height="28px" title="three" />`,

	":one::one:": `<img src="/emotes/one.png" height="28px" title="one" /><img src="/emotes/one.png" height="28px" title="one" />`,
	":one:one:":  `<img src="/emotes/one.png" height="28px" title="one" />one:`,
	"oneone":     "oneone",
	"one:one:":   `one<img src="/emotes/one.png" height="28px" title="one" />`,

	"[one]":   `<img src="/emotes/one.png" height="28px" title="one" />`,
	"[two]":   `<img src="/emotes/two.png" height="28px" title="two" />`,
	"[three]": `<img src="/emotes/three.gif" height="28px" title="three" />`,

	"[one][one]": `<img src="/emotes/one.png" height="28px" title="one" /><img src="/emotes/one.png" height="28px" title="one" />`,
	"[one]one":   `<img src="/emotes/one.png" height="28px" title="one" />one`,

	":one: two [three]": `<img src="/emotes/one.png" height="28px" title="one" /> <img src="/emotes/two.png" height="28px" title="two" /> <img src="/emotes/three.gif" height="28px" title="three" />`,

	"nope one what":     `nope <img src="/emotes/one.png" height="28px" title="one" /> what`,
	"nope :two: what":   `nope <img src="/emotes/two.png" height="28px" title="two" /> what`,
	"nope [three] what": `nope <img src="/emotes/three.gif" height="28px" title="three" /> what`,
}

var data_wrapped = map[string]string{
	"one":   `one`,
	"two":   `two`,
	"three": `three`,

	":one:":   `<img src="/emotes/one.png" height="28px" title="one" />`,
	":two:":   `<img src="/emotes/two.png" height="28px" title="two" />`,
	":three:": `<img src="/emotes/three.gif" height="28px" title="three" />`,

	":one::one:": `<img src="/emotes/one.png" height="28px" title="one" /><img src="/emotes/one.png" height="28px" title="one" />`,
	":one:one:":  `<img src="/emotes/one.png" height="28px" title="one" />one:`,
	"oneone":     "oneone",
	"one:one:":   `one<img src="/emotes/one.png" height="28px" title="one" />`,

	"[one]":   `<img src="/emotes/one.png" height="28px" title="one" />`,
	"[two]":   `<img src="/emotes/two.png" height="28px" title="two" />`,
	"[three]": `<img src="/emotes/three.gif" height="28px" title="three" />`,

	"[one][one]": `<img src="/emotes/one.png" height="28px" title="one" /><img src="/emotes/one.png" height="28px" title="one" />`,
	"[one]one":   `<img src="/emotes/one.png" height="28px" title="one" />one`,

	":one: two [three]": `<img src="/emotes/one.png" height="28px" title="one" /> two <img src="/emotes/three.gif" height="28px" title="three" />`,

	"nope one what":     `nope one what`,
	"nope :two: what":   `nope <img src="/emotes/two.png" height="28px" title="two" /> what`,
	"nope [three] what": `nope <img src="/emotes/three.gif" height="28px" title="three" /> what`,
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

func TestEmotes_ParseEmotes_WrappedOnly(t *testing.T) {
	for input, expected := range data_good {
		got := ParseEmotes(input)
		if got != expected {
			t.Errorf("%s failed to parse into %q. Received: %q", input, expected, got)
		}
	}
}
