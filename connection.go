package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/zorchenhimer/MovieNight/common"
)

type chatConnection struct {
	*websocket.Conn
	mutex        sync.RWMutex
	forwardedFor string
	clientName   string
}

func (cc *chatConnection) ReadData(data interface{}) error {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	stats.msgInInc()
	return cc.ReadJSON(data)
}

func (cc *chatConnection) WriteData(data interface{}) error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	stats.msgOutInc()
	err := cc.WriteJSON(data)
	if err != nil {
		if operr, ok := err.(*net.OpError); ok {
			common.LogDebugln("OpError: " + operr.Err.Error())
		}
		return fmt.Errorf("Error writing data to %s %s: %w", cc.clientName, cc.Host(), err)
	}
	return nil
}

func (cc *chatConnection) Host() string {
	if len(cc.forwardedFor) > 0 {
		return cc.forwardedFor
	}

	host, _, err := net.SplitHostPort(cc.RemoteAddr().String())
	if err != nil {
		return cc.RemoteAddr().String()
	}
	return host
}
