package main

import (
	"testing"

	"github.com/zorchenhimer/MovieNight/common"
)

func TestClient_addSpoilerTag(t *testing.T) {
	data := [][]string{
		{"||||", spoilerStart + spoilerEnd},
		{"|||||", spoilerStart + spoilerEnd + "|"},
		{"||||||", spoilerStart + spoilerEnd + "||"},
		{"|||||||", spoilerStart + spoilerEnd + "|||"},
		{"||||||||", spoilerStart + spoilerEnd + spoilerStart + spoilerEnd},
		{"||test||", spoilerStart + "test" + spoilerEnd},
		{"|| ||", spoilerStart + " " + spoilerEnd},
		{"|s|||", "|s|||"},
	}

	for i := range data {
		s := addSpoilerTags(data[i][0])
		if s != data[i][1] {
			t.Errorf("expected %#v, got %#v with %#v", data[i][1], s, data[i][0])
		}
	}
}

// Name highlighting should not interfere with emotes
func TestClient_emoteHighlight(t *testing.T) {
	data := [][]string{
		{"zorchenhimer", `<span class="mention">zorchenhimer</span>`},
		{"@zorchenhimer", `<span class="mention">@zorchenhimer</span>`},
		{"Zorchenhimer", `<span class="mention">Zorchenhimer</span>`},
		{"@Zorchenhimer", `<span class="mention">@Zorchenhimer</span>`},
		{"hello zorchenhimer", `hello <span class="mention">zorchenhimer</span>`},
		{"hello zorchenhimer ass", `hello <span class="mention">zorchenhimer</span> ass`},
		{`<img src="/emotes/twitch/zorchenhimer/zorcheWhat.png" height="28px" title="zorcheWhat">`, `<img src="/emotes/twitch/zorchenhimer/zorcheWhat.png" height="28px" title="zorcheWhat">`},
		{`zorchenhimer <img src="/emotes/twitch/zorchenhimer/zorcheWhat.png" height="28px" title="zorcheWhat">`, `<span class="mention">zorchenhimer</span> <img src="/emotes/twitch/zorchenhimer/zorcheWhat.png" height="28px" title="zorcheWhat">`},
	}

	client, err := NewClient(nil, nil, "Zorchenhimer", "#9547ff")
	if err != nil {
		t.Errorf("Client init error: %v", err)
	}

	for _, d := range data {
		chatData := client.replaceColorizedName(common.NewChatMessage(client.name, client.color, d[0], common.CmdlUser, common.MsgChat))
		if chatData.Data.(common.DataMessage).Message != d[1] {
			t.Errorf("\nExpected:\n\t%s\nReceived\n\t%s", d[1], chatData.Data.(common.DataMessage).Message)
		} else {
			t.Logf("Passed %s", d[0])
		}
	}
}
