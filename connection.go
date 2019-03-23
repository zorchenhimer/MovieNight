package main

import (
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

type chatConnection struct {
	*websocket.Conn
	mutex        sync.RWMutex
	forwardedFor string
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
	return cc.WriteJSON(data)
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
