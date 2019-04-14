package main

import (
	"sync"
	"time"

	"github.com/zorchenhimer/MovieNight/common"
)

type streamStats struct {
	messageIn  int
	messageOut int
	maxUsers   int
	start      time.Time
	mutex      sync.Mutex

	streamStart time.Time
	streamLive  bool // True if live
}

func newStreamStats() streamStats {
	return streamStats{start: time.Now(), streamLive: false}
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

func (s *streamStats) updateMaxUsers(count int) {
	s.mutex.Lock()
	if count > s.maxUsers {
		s.maxUsers = count
	}
	s.mutex.Unlock()
}

func (s *streamStats) getMaxUsers() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.maxUsers
}

func (s *streamStats) Print() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	common.LogInfof("Messages In: %d\n", s.messageIn)
	common.LogInfof("Messages Out: %d\n", s.messageOut)
	common.LogInfof("Max users in chat: %d\n", s.maxUsers)
	common.LogInfof("Total Time: %s\n", time.Since(s.start))
}

func (s *streamStats) startStream() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.streamLive = true
	s.streamStart = time.Now()
}

func (s *streamStats) endStream() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.streamLive = false
}

func (s *streamStats) getStreamLength() time.Duration {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.streamLive {
		return 0
	}
	return time.Since(s.streamStart)
}
