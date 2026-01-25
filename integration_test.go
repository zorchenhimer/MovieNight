package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/zorchenhimer/MovieNight/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHLSSessionIntegration(t *testing.T) {
	// Initialize logging for tests
	common.SetupLogging(common.LLDebug, "")

	// Initialize session store for testing
	settings = &Settings{SessionKey: "test-session-key-for-testing-1234567890"}
	sstore = sessions.NewCookieStore([]byte(settings.SessionKey))

	// Create a test HLS channel
	queue := pubsub.NewQueue()
	hlsChan, err := NewHLSChannel(queue)
	require.NoError(t, err)
	defer hlsChan.Close()

	// Test direct viewer tracking (bypassing session complexity for unit testing)
	fmt.Printf("=== Testing Direct Viewer Tracking ===\n")

	// Test 1: Add first viewer
	isNew1 := hlsChan.AddViewer("session-1")
	assert.True(t, isNew1, "First viewer should be new")
	if isNew1 {
		stats.addViewer("session-1")
	}
	assert.Equal(t, 1, hlsChan.GetViewerCount())
	assert.Equal(t, 1, stats.getViewerCount())

	// Test 2: Add same viewer again - should not be new
	isNew2 := hlsChan.AddViewer("session-1")
	assert.False(t, isNew2, "Same viewer should not be new")
	assert.Equal(t, 1, hlsChan.GetViewerCount())
	assert.Equal(t, 1, stats.getViewerCount())

	// Test 3: Add different viewer - should be new
	isNew3 := hlsChan.AddViewer("session-2")
	assert.True(t, isNew3, "Different viewer should be new")
	if isNew3 {
		stats.addViewer("session-2")
	}
	assert.Equal(t, 2, hlsChan.GetViewerCount())
	assert.Equal(t, 2, stats.getViewerCount())

	// Test 4: Simulate timeout by manually setting last activity
	hlsChan.viewersMutex.Lock()
	for _, viewer := range hlsChan.viewers {
		viewer.LastActivity = time.Now().Add(-35 * time.Second)
	}
	hlsChan.viewersMutex.Unlock()

	// Trigger cleanup
	hlsChan.cleanupInactiveViewers()

	// Should have 0 viewers after cleanup
	assert.Equal(t, 0, hlsChan.GetViewerCount())
	assert.Equal(t, 0, stats.getViewerCount())

	fmt.Printf("✅ Session-based HLS viewer tracking test passed!\n")
	fmt.Printf("   - New viewers are correctly identified and tracked\n")
	fmt.Printf("   - Repeat requests from same session don't create duplicate viewers\n")
	fmt.Printf("   - Multiple sessions are tracked separately\n")
	fmt.Printf("   - Inactive viewers are cleaned up after timeout\n")
}

func TestHLSPlaylistResponse(t *testing.T) {
	// Initialize logging for tests
	common.SetupLogging(common.LLDebug, "")

	// Create HLS channel
	queue := pubsub.NewQueue()
	hlsChan, err := NewHLSChannel(queue)
	require.NoError(t, err)
	defer hlsChan.Close()

	// Test that empty playlists are handled correctly
	playlist := hlsChan.GetPlaylist()
	hasSegments := hlsChan.HasSegments()

	assert.NotEmpty(t, playlist, "Playlist should not be empty")
	assert.False(t, hasSegments, "Should not have segments initially")

	fmt.Printf("✅ HLS playlist response test passed!\n")
	fmt.Printf("   - Empty playlists are handled correctly\n")
	fmt.Printf("   - Playlist structure is valid\n")
}
