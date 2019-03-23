package main

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	ErrConnectionClosed = errors.New("Connection has been closed")
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
	err := cc.WriteJSON(data)
	if err != nil {
		switch t := err.(type) {
		case *net.OpError:
			// only handle the close connection.
			// net.OpError.Op provides more information
			return ErrConnectionClosed
		default:
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
