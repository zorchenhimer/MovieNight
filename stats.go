package main

import (
	"fmt"
	"sync"
	"time"
)

type streamStats struct {
	messageIn  int
	messageOut int
	start      time.Time
	mutex      sync.Mutex
}

func newStreamStats() streamStats {
	return streamStats{start: time.Now()}
}

func (s *streamStats) msgInInc() {
	s.mutex.Lock()
	s.messageIn++
	s.mutex.Unlock()
}
func (s *streamStats) msgOutInc() {
	s.mutex.Lock()
	s.messageOut++
	s.mutex.Unlock()
}

func (s *streamStats) Print() {
	common.LogInfof("Messages In: %d\n", s.messageIn)
	common.LogInfof("Messages Out: %d\n", s.messageOut)
	common.LogInfof("Total Time: %s\n", time.Since(s.start))
}
