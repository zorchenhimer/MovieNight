package main

import (
	"testing"
	"time"

	"github.com/nareix/joy4/av/pubsub"
	"github.com/zorchenhimer/MovieNight/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHLSChannel_ViewerTimeout(t *testing.T) {
	// Initialize logging for tests
	common.SetupLogging(common.LLDebug, "")

	queue := pubsub.NewQueue()
	hlsChan, err := NewHLSChannel(queue)
	require.NoError(t, err)
	defer hlsChan.Close()

	// Add a viewer
	isNew := hlsChan.AddViewer("test-session")
	assert.True(t, isNew, "Should be a new viewer")
	assert.Equal(t, 1, hlsChan.GetViewerCount())

	// Adding the same viewer again should not be new
	isNew = hlsChan.AddViewer("test-session")
	assert.False(t, isNew, "Should not be a new viewer")
	assert.Equal(t, 1, hlsChan.GetViewerCount())

	// Manually trigger cleanup (to avoid waiting 30 seconds in test)
	hlsChan.cleanupInactiveViewers()

	// Should still have viewer since it's been active recently
	assert.Equal(t, 1, hlsChan.GetViewerCount())

	// Modify the viewer's last activity to simulate timeout
	hlsChan.viewersMutex.Lock()
	if viewer, exists := hlsChan.viewers["test-session"]; exists {
		viewer.LastActivity = time.Now().Add(-35 * time.Second) // Simulate 35 seconds ago
	}
	hlsChan.viewersMutex.Unlock()

	// Trigger cleanup again
	hlsChan.cleanupInactiveViewers()

	// Should now have no viewers due to timeout
	assert.Equal(t, 0, hlsChan.GetViewerCount())
}

func TestHLSChannel_ViewerActivityUpdate(t *testing.T) {
	// Initialize logging for tests
	common.SetupLogging(common.LLDebug, "")

	queue := pubsub.NewQueue()
	hlsChan, err := NewHLSChannel(queue)
	require.NoError(t, err)
	defer hlsChan.Close()

	// Add a viewer
	isNew := hlsChan.AddViewer("test-session")
	assert.True(t, isNew, "Should be a new viewer")

	// Get initial activity time
	hlsChan.viewersMutex.RLock()
	initialActivity := hlsChan.viewers["test-session"].LastActivity
	hlsChan.viewersMutex.RUnlock()

	// Wait a small amount and add again (simulating playlist refresh)
	time.Sleep(10 * time.Millisecond)
	isNew = hlsChan.AddViewer("test-session")
	assert.False(t, isNew, "Should not be a new viewer")

	// Check that activity was updated
	hlsChan.viewersMutex.RLock()
	updatedActivity := hlsChan.viewers["test-session"].LastActivity
	hlsChan.viewersMutex.RUnlock()

	assert.True(t, updatedActivity.After(initialActivity), "Activity should be updated")
}

func TestHLSChannel_BackgroundCleanup(t *testing.T) {
	// Initialize logging for tests
	common.SetupLogging(common.LLDebug, "")

	queue := pubsub.NewQueue()
	hlsChan, err := NewHLSChannel(queue)
	require.NoError(t, err)

	// Add a viewer
	hlsChan.AddViewer("test-session")
	assert.Equal(t, 1, hlsChan.GetViewerCount())

	// Manually set the viewer's activity to past the timeout
	hlsChan.viewersMutex.Lock()
	if viewer, exists := hlsChan.viewers["test-session"]; exists {
		viewer.LastActivity = time.Now().Add(-35 * time.Second)
	}
	hlsChan.viewersMutex.Unlock()

	// The background cleanup should run every 10 seconds, but we'll trigger it manually
	hlsChan.cleanupInactiveViewers()

	// Viewer should be cleaned up
	assert.Equal(t, 0, hlsChan.GetViewerCount())

	// Close the channel to test proper cleanup
	hlsChan.Close()
}
