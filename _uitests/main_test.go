package main

import (
	"fmt"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

const (
	url              = "http://localhost:8089/chat"
	nameInput        = `//input[@id="name"]`
	joinButton       = `//input[@id="join"]`
	msgOuputDiv      = `//div[@id="messages"]`
	msgInputTextarea = `//textarea[@id="msg"]`
	sendInput        = `//input[@id="send"]`
	nameSpan         = `//span[contains(@class, "name") and text()="%s"]`
	msgSpan          = `//span[contains(@class, "msg") and text()="%s"]`
)

func openBrowser() (playwright.Browser, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not run playwright: %w", err)
	}

	browser, err := pw.Firefox.Launch()
	if err != nil {
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}

	return browser, nil
}

func openChat(t *testing.T, browser playwright.Browser, username string) (playwright.Page, error) {
	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not open new browser page: %w", err)
	}

	_, err = page.Goto(url)
	if err != nil {
		return nil, fmt.Errorf("could not open chat page: %w", err)
	}

	_, err = page.WaitForSelector(nameInput)
	if err != nil {
		return nil, fmt.Errorf("could not find name input: %w", err)
	}

	err = page.Type(nameInput, username)
	if err != nil {
		return nil, fmt.Errorf("an error occured when typing the username: %w", err)
	}

	err = page.Click(joinButton)
	if err != nil {
		return nil, fmt.Errorf("could not click join button: %w", err)
	}

	_, err = page.WaitForSelector(msgOuputDiv)
	if err != nil {
		return nil, fmt.Errorf("chat window did not show: %w", err)
	}

	_, err = page.WaitForSelector(fmt.Sprintf(nameSpan, username))
	if err != nil {
		return nil, fmt.Errorf("could not get join message span: %w", err)
	}

	return page, nil
}

func TestAccessFirefox(t *testing.T) {
	browser, err := openBrowser()
	if err != nil {
		t.Error(err)
	}
	defer browser.Close()

	_, err = openChat(t, browser, "testUser")
	if err != nil {
		t.Error(err)
	}
}

func TestSeeingChat(t *testing.T) {
	browser, err := openBrowser()
	if err != nil {
		t.Error(err)
	}
	defer browser.Close()

	page1, err := openChat(t, browser, "testUser1")
	if err != nil {
		t.Errorf("could not open chat for user 1: %v", err)
	}

	page2, err := openChat(t, browser, "testUser2")
	if err != nil {
		t.Errorf("could not open chat for user 2: %v", err)
	}

	msg := "testing 1 2 3"
	err = page1.Type(msgInputTextarea, msg)
	if err != nil {
		t.Errorf("could not find msg box: %v", err)
	}

	err = page1.Click(sendInput)
	if err != nil {
		t.Errorf("could not send message: %v", err)
	}

	_, err = page2.WaitForSelector(fmt.Sprintf(msgSpan, msg))
	if err != nil {
		t.Errorf("did not find testing msg: %v", err)
	}
}
