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
	fmt.Printf("Messages In: %d\n", s.messageIn)
	fmt.Printf("Messages Out: %d\n", s.messageOut)
	fmt.Printf("Total Time: %s\n", time.Since(s.start))
}
