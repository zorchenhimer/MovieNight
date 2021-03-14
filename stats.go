package main

import (
	"sync"
	"time"

	"github.com/zorchenhimer/MovieNight/common"
)

type streamStats struct {
	messageIn   int
	messageOut  int
	maxUsers    int
	start       time.Time
	mutex       sync.Mutex
	streamStart time.Time
	streamLive  bool // True if live
	viewers     map[string]string
	maxViewers  int
}

func (s *streamStats) addViewer(ip string) {
	s.mutex.Lock()
	s.viewers[ip] = ip
	s.updateMaxViewers(len(s.viewers))
	s.mutex.Unlock()

	common.LogDebugf("Viewer connect from: %s\n", ip)
}
func (s *streamStats) removeViewer(ip string) {
	s.mutex.Lock()
	delete(s.viewers, ip)
	s.mutex.Unlock()

	common.LogDebugf("Viewer left from: %s\n", ip)
}

func (s *streamStats) updateMaxViewers(size int) {
	if s.maxViewers < size {
		s.maxViewers = size
	}
}

func (s *streamStats) resetViewers() {
	s.viewers = make(map[string]string)
}

func newStreamStats() streamStats {
	return streamStats{start: time.Now(), streamLive: false, viewers: make(map[string]string)}
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
	common.LogInfof("Max Stream Viewer: %d\n", s.maxViewers)
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

func (s *streamStats) getViewerCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return len(s.viewers)
}

func (s *streamStats) getMaxViewerCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.maxViewers
}

func (s *streamStats) getViewers() map[string]string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.viewers
}
