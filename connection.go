package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type chatConnection struct {
	*websocket.Conn
	mutex sync.Mutex
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
