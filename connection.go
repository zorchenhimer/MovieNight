package main

import (
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

	return cc.ReadJSON(data)
}

func (cc *chatConnection) WriteData(data interface{}) error {
	defer cc.mutex.Unlock()
	cc.mutex.Lock()

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
