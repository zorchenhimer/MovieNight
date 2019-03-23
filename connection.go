package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

type chatConnection struct {
	*websocket.Conn
	mutex        sync.Mutex
	forwardedFor string
}

func (cc *chatConnection) ReadData(data interface{}) error {
	defer cc.mutex.Unlock()
	cc.mutex.Lock()
	stats.msgInInc()
	return cc.ReadJSON(data)
}

func (cc *chatConnection) WriteData(data interface{}) error {
	defer cc.mutex.Unlock()
	cc.mutex.Lock()
	stats.msgOutInc()
	err := cc.WriteJSON(data)
	if err != nil {
		return fmt.Errorf("Error writing data to %s: %v", cc.Host(), err)
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
