package main

import (
	"testing"

	"github.com/mxschmitt/playwright-go"
)

const (
	url        = "http://localhost:8089/chat"
	nameInput  = `//input[@id="name"]`
	joinButton = `//input[@id="join"]`
	msgDiv     = `//div[@id="messages"]`
)

func TestAccessFirefox(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Error(err)
	}

	browser, err := pw.Firefox.Launch()
	if err != nil {
		t.Error(err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		t.Error(err)
	}

	_, err = page.Goto(url)
	if err != nil {
		t.Error(err)
	}

	_, err = page.WaitForSelector(nameInput)
	if err != nil {
		t.Error(err)
	}

	err = page.Type(nameInput, "testUser")
	if err != nil {
		t.Error(err)
	}

	err = page.Click(joinButton)
	if err != nil {
		t.Error(err)
	}

	_, err = page.WaitForSelector(msgDiv)
	if err != nil {
		t.Error(err)
	}
}
