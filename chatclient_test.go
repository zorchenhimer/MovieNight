package main

import "testing"

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
