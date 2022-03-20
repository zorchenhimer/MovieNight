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
	viewers     map[string]int
	maxViewers  int
}

func (s *streamStats) addViewer(id string) {
	s.mutex.Lock()
	s.viewers[id] = len(s.viewers)
	size := len(s.viewers)
	s.updateMaxViewers(size)
	s.mutex.Unlock()

	common.LogDebugf("[stats] %d viewer(s) connected\n", size)
}
func (s *streamStats) removeViewer(id string) {
	s.mutex.Lock()
	delete(s.viewers, id)
	s.mutex.Unlock()

	common.LogDebugf("[stats] One viewer left the stream\n")
}

func (s *streamStats) updateMaxViewers(size int) {
	if s.maxViewers < size {
		s.maxViewers = size
	}
}

func (s *streamStats) resetViewers() {
	s.viewers = sessionsMapNew()
}

func sessionsMapNew() map[string]int {
	return make(map[string]int)
}

func newStreamStats() streamStats {
	return streamStats{start: time.Now(), streamLive: false, viewers: sessionsMapNew()}
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

	common.LogInfof("[stats] Messages In: %d\n", s.messageIn)
	common.LogInfof("[stats] Messages Out: %d\n", s.messageOut)
	common.LogInfof("[stats] Max users in chat: %d\n", s.maxUsers)
	common.LogInfof("[stats] Total Time: %s\n", time.Since(s.start))
	common.LogInfof("[stats] Max Stream Viewer: %d\n", s.maxViewers)
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
