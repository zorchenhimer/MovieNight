package main

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nareix/joy4/av/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zorchenhimer/MovieNight/common"
)

func TestDetectDeviceCapabilities(t *testing.T) {
	tests := []struct {
		name          string
		userAgent     string
		expectedIOS   bool
		expectedHLS   bool
		expectedCodec string
	}{
		{
			name:          "iPhone user agent",
			userAgent:     "Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15",
			expectedIOS:   true,
			expectedHLS:   true,
			expectedCodec: "hls",
		},
		{
			name:          "iPad user agent",
			userAgent:     "Mozilla/5.0 (iPad; CPU OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15",
			expectedIOS:   true,
			expectedHLS:   true,
			expectedCodec: "hls",
		},
		{
			name:          "Desktop Chrome",
			userAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/93.0.4577.63 Safari/537.36",
			expectedIOS:   false,
			expectedHLS:   true,
			expectedCodec: "flv",
		},
		{
			name:          "Android device",
			userAgent:     "Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/93.0.4577.62 Mobile Safari/537.36",
			expectedIOS:   false,
			expectedHLS:   true,
			expectedCodec: "hls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("User-Agent", tt.userAgent)

			capabilities := DetectDeviceCapabilities(req)

			assert.Equal(t, tt.expectedIOS, capabilities.IsIOS)
			assert.Equal(t, tt.expectedHLS, capabilities.SupportsHLS)
			assert.Equal(t, tt.expectedCodec, capabilities.PreferredCodec)
			assert.Equal(t, tt.userAgent, capabilities.UserAgent)
		})
	}
}

func TestDetectDeviceCapabilities_NilRequest(t *testing.T) {
	capabilities := DetectDeviceCapabilities(nil)

	assert.False(t, capabilities.IsIOS)
	assert.False(t, capabilities.SupportsHLS)
	assert.Equal(t, "flv", capabilities.PreferredCodec)
	assert.Equal(t, "", capabilities.UserAgent)
}

func TestShouldUseHLS(t *testing.T) {
	tests := []struct {
		name       string
		userAgent  string
		queryParam string
		expected   bool
	}{
		{
			name:      "iOS device should use HLS",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15",
			expected:  true,
		},
		{
			name:      "Desktop Chrome should use FLV by default",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expected:  false,
		},
		{
			name:       "Force HLS via query parameter",
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			queryParam: "hls",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/"
			if tt.queryParam != "" {
				url += "?format=" + tt.queryParam
			}

			req := httptest.NewRequest("GET", url, nil)
			req.Header.Set("User-Agent", tt.userAgent)

			result := ShouldUseHLS(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldUseHLS_NilRequest(t *testing.T) {
	result := ShouldUseHLS(nil)
	assert.False(t, result)
}

func TestIsValidSegmentURI(t *testing.T) {
	tests := []struct {
		uri      string
		expected bool
	}{
		// Legacy numeric format (backward compatibility)
		{"segment_0.ts", true},
		{"segment_123.ts", true},
		{"segment_999999.ts", true},
		{"/live/segment_0.ts", true},
		{"/live/segment_123.ts", true},
		{"/hls/stream/segment_999.ts", true},

		// New UUID format
		{"segment_a1b2c3d4e5f6789012345678abcdef90.ts", true},
		{"/live/segment_1234567890abcdef1234567890abcdef.ts", true},
		{"segment_ABCDEF1234567890abcdef1234567890.ts", true},

		// Invalid formats
		{"invalid.ts", false},
		{"segment_0.mp4", false},
		{"", false},
		{"segment_.ts", false},
		{"_0.ts", false},
		{"/live/invalid.ts", false},
		{"/live/segment_.ts", false},
		{"segment_invalid_uuid.ts", false}, // Invalid UUID length
		{"segment_xyz123.ts", false},       // Invalid hex characters
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			result := IsValidSegmentURI(tt.uri)
			assert.Equal(t, tt.expected, result, "URI: %s", tt.uri)
		})
	}
}

func TestParseSequenceFromURI(t *testing.T) {
	tests := []struct {
		uri           string
		expectedSeq   uint64
		expectedError bool
	}{
		// Legacy numeric format (still supported)
		{"segment_0.ts", 0, false},
		{"segment_123.ts", 123, false},
		{"segment_999999.ts", 999999, false},
		{"/live/segment_0.ts", 0, false},
		{"/live/segment_123.ts", 123, false},
		{"/hls/stream/segment_999.ts", 999, false},

		// UUID format (should return error as sequence not available)
		{"segment_a1b2c3d4e5f6789012345678abcdef90.ts", 0, true},
		{"/live/segment_1234567890abcdef1234567890abcdef.ts", 0, true},

		// Invalid formats
		{"invalid.ts", 0, true},
		{"segment_.ts", 0, true},
		{"", 0, true},
		{"segment_abc.ts", 0, true},
		{"/live/invalid.ts", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			seq, err := ParseSequenceFromURI(tt.uri)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSeq, seq)
			}
		})
	}
}

func TestGenerateSegmentID(t *testing.T) {
	// Test that generateSegmentID produces valid UUID format
	id1 := generateSegmentID()
	id2 := generateSegmentID()

	// Should be 32 hex characters
	assert.Len(t, id1, 32)
	assert.Len(t, id2, 32)

	// Should be different each time
	assert.NotEqual(t, id1, id2)

	// Should contain only hex characters
	for _, c := range id1 {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"ID should contain only hex characters, got: %c", c)
	}

	// Test that generated IDs create valid segment URIs
	uri := fmt.Sprintf("/live/segment_%s.ts", id1)
	assert.True(t, IsValidSegmentURI(uri))
}

func TestNewHLSChannel(t *testing.T) {
	// Initialize logging for tests
	common.SetupLogging(common.LLDebug, "")

	// Test with nil queue
	_, err := NewHLSChannel(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue cannot be nil")

	// Test with valid queue
	queue := pubsub.NewQueue()
	hlsChan, err := NewHLSChannel(queue)
	require.NoError(t, err)
	require.NotNil(t, hlsChan)

	assert.Equal(t, queue, hlsChan.que)
	assert.NotNil(t, hlsChan.playlist)
	assert.Equal(t, 4*time.Second, hlsChan.targetDuration)
	assert.Equal(t, 4*time.Second, hlsChan.segmentDuration)
	assert.Equal(t, 6, hlsChan.maxSegments)
	assert.Equal(t, uint64(0), hlsChan.sequenceNumber)
	assert.NotNil(t, hlsChan.viewers)
	assert.NotNil(t, hlsChan.ctx)
	assert.NotNil(t, hlsChan.cancel)

	// Clean up
	hlsChan.Stop()
}

func TestHLSChannel_ViewerTracking(t *testing.T) {
	// Initialize logging for tests
	common.SetupLogging(common.LLDebug, "")

	queue := pubsub.NewQueue()
	hlsChan, err := NewHLSChannel(queue)
	require.NoError(t, err)
	defer hlsChan.Close() // Use Close() instead of Stop()

	// Test adding viewers - should return true for new viewers
	isNew1 := hlsChan.AddViewer("user1")
	assert.True(t, isNew1, "First time adding user1 should return true")

	isNew2 := hlsChan.AddViewer("user2")
	assert.True(t, isNew2, "First time adding user2 should return true")

	assert.Equal(t, 2, hlsChan.GetViewerCount())

	// Test adding same viewer again - should return false
	isNew1Again := hlsChan.AddViewer("user1")
	assert.False(t, isNew1Again, "Adding user1 again should return false")
	assert.Equal(t, 2, hlsChan.GetViewerCount(), "Count should remain the same")

	// Test removing viewers
	hlsChan.RemoveViewer("user1")
	assert.Equal(t, 1, hlsChan.GetViewerCount())

	hlsChan.RemoveViewer("user2")
	assert.Equal(t, 0, hlsChan.GetViewerCount())

	// Test removing non-existent viewer
	hlsChan.RemoveViewer("nonexistent")
	assert.Equal(t, 0, hlsChan.GetViewerCount())
}

func TestHLSChannel_HasSegments(t *testing.T) {
	// Initialize logging for tests
	common.SetupLogging(common.LLDebug, "")

	// Test with nil channel
	var nilChan *HLSChannel
	require.False(t, nilChan.HasSegments())

	// Create HLS channel
	queue := pubsub.NewQueue()
	hlsChan, err := NewHLSChannel(queue)
	require.NoError(t, err)
	require.NotNil(t, hlsChan)
	defer hlsChan.Stop()

	// Initially should have no segments
	require.False(t, hlsChan.HasSegments())

	// Add a segment manually for testing
	hlsChan.mutex.Lock()
	hlsChan.segments = append(hlsChan.segments, HLSSegment{
		URI:      "segment_0.ts",
		Duration: 5.0,
		Data:     []byte("test data"),
		Sequence: 0,
	})
	hlsChan.mutex.Unlock()

	// Now should have segments
	require.True(t, hlsChan.HasSegments())
}

func TestHLSChannel_SlidingWindow(t *testing.T) {
	// Setup logging for test
	common.SetupLogging(common.LLError, "/dev/null")

	// Create HLS channel with small window for testing
	queue := pubsub.NewQueue()
	hlsChan, err := NewHLSChannel(queue)
	require.NoError(t, err)
	require.NotNil(t, hlsChan)
	defer hlsChan.Stop()

	// Manually set small maxSegments for testing
	hlsChan.maxSegments = 3

	// Test adding segments beyond window size
	testData := []byte("test segment data")

	// Add segments one by one and verify sliding window behavior
	for i := 0; i < 5; i++ {
		// Create segment manually for testing
		currentSeq := hlsChan.sequenceNumber
		hlsChan.sequenceNumber++

		segmentURI := fmt.Sprintf("/live/segment_%d.ts", currentSeq)
		durationSeconds := 2.0

		segment := HLSSegment{
			URI:      segmentURI,
			Duration: durationSeconds,
			Data:     testData,
			Sequence: currentSeq,
		}

		// Add segment with proper sliding window management
		hlsChan.addGeneratedSegment(segment)

		// Check that we never exceed maxSegments in our local storage
		hlsChan.mutex.RLock()
		segmentCount := len(hlsChan.segments)
		playlistCount := int(hlsChan.playlist.Count())
		hlsChan.mutex.RUnlock()

		expectedSegments := i + 1
		if expectedSegments > hlsChan.maxSegments {
			expectedSegments = hlsChan.maxSegments
		}

		assert.LessOrEqual(t, segmentCount, hlsChan.maxSegments,
			"Local segment count should not exceed maxSegments")
		assert.LessOrEqual(t, playlistCount, hlsChan.maxSegments,
			"Playlist count should not exceed maxSegments")
		assert.Equal(t, expectedSegments, segmentCount,
			"Local segment count should match expected after sliding window")

		t.Logf("Added segment %d: local_count=%d, playlist_count=%d, max=%d",
			i, segmentCount, playlistCount, hlsChan.maxSegments)
	}

	// Final verification: should have exactly maxSegments
	hlsChan.mutex.RLock()
	finalSegmentCount := len(hlsChan.segments)
	finalPlaylistCount := int(hlsChan.playlist.Count())
	hlsChan.mutex.RUnlock()

	assert.Equal(t, hlsChan.maxSegments, finalSegmentCount,
		"Should have exactly maxSegments in local storage")
	assert.Equal(t, hlsChan.maxSegments, finalPlaylistCount,
		"Should have exactly maxSegments in playlist")
}

func TestHLSChannel_GetPlaylist(t *testing.T) {
	// Initialize logging for tests
	common.SetupLogging(common.LLDebug, "")

	queue := pubsub.NewQueue()
	hlsChan, err := NewHLSChannel(queue)
	require.NoError(t, err)
	defer hlsChan.Stop()

	playlist := hlsChan.GetPlaylist()
	assert.NotEmpty(t, playlist)
	assert.Contains(t, playlist, "#EXTM3U")
	assert.Contains(t, playlist, "#EXT-X-VERSION:6")
}

func TestHLSChannel_GetPlaylist_Nil(t *testing.T) {
	var hlsChan *HLSChannel
	playlist := hlsChan.GetPlaylist()
	assert.Empty(t, playlist)
}

func TestHLSChannel_NilChecks(t *testing.T) {
	var hlsChan *HLSChannel

	// Test nil checks
	hlsChan.AddViewer("test")    // Should not panic
	hlsChan.RemoveViewer("test") // Should not panic
	assert.Equal(t, 0, hlsChan.GetViewerCount())

	_, err := hlsChan.GetSegment(0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HLS channel is nil")

	_, err = hlsChan.GetSegmentByURI("segment_0.ts")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HLS channel is nil")
}

func TestGetContentTypeForFormat(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"hls", "application/vnd.apple.mpegurl"},
		{"ts", "video/mp2t"},
		{"flv", "video/x-flv"},
		{"unknown", "video/x-flv"},
		{"", "video/x-flv"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := GetContentTypeForFormat(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsHLSPlaylistRequest(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		accept   string
		expected bool
	}{
		{"m3u8 extension", "/live/playlist.m3u8", "", true},
		{"M3U8 extension uppercase", "/live/PLAYLIST.M3U8", "", true},
		{"playlist in path", "/live/playlist", "", true},
		{"Accept header", "/live", "application/vnd.apple.mpegurl", true},
		{"format=hls parameter", "/live?format=hls", "", true},
		{"regular path", "/live", "", false},
		{"ts segment", "/live/segment_0.ts", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}

			result := IsHLSPlaylistRequest(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsHLSSegmentRequest(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"ts segment", "/live/segment_0.ts", true},
		{"TS segment uppercase", "/live/SEGMENT_0.TS", true},
		{"playlist", "/live/playlist.m3u8", false},
		{"regular path", "/live", false},
		{"other ts file", "/live/other.ts", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)

			result := IsHLSSegmentRequest(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateUserAgent(t *testing.T) {
	tests := []struct {
		userAgent string
		expected  bool
	}{
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X)", true},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/93.0", true},
		{"", false},
		{"curl/7.68.0", false},
		{"wget/1.20.3", false},
		{"python-requests/2.25.1", false},
		{"Googlebot/2.1", false},
		{string(make([]byte, 1001)), false}, // Too long
	}

	setupSettings("", "")

	for _, tt := range tests {
		t.Run(tt.userAgent, func(t *testing.T) {
			result := ValidateUserAgent(tt.userAgent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration test demonstrating the full HLS workflow
func TestHLSIntegration(t *testing.T) {
	t.Run("iOS Device HLS Flow", func(t *testing.T) {
		// Simulate iOS Safari request
		req := httptest.NewRequest("GET", "/live", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15")

		// Test device detection
		shouldUseHLS := ShouldUseHLS(req)
		assert.True(t, shouldUseHLS, "iOS devices should use HLS")

		format := GetStreamingFormat(req)
		assert.Equal(t, "hls", format, "iOS should get HLS format")

		capabilities := DetectDeviceCapabilities(req)
		assert.True(t, capabilities.IsIOS, "Should detect iOS")
		assert.True(t, capabilities.SupportsHLS, "iOS should support HLS")
		assert.Equal(t, "hls", capabilities.PreferredCodec, "iOS should prefer HLS")

		// Test quality settings
		qualitySettings := GetQualitySettings(capabilities)
		assert.Equal(t, 0.7, qualitySettings.BitrateMultiplier, "iOS mobile should have 30% bitrate reduction")
		assert.Equal(t, "720p", qualitySettings.Resolution, "iOS mobile should use 720p")
	})

	t.Run("Desktop Chrome FLV Flow", func(t *testing.T) {
		// Simulate Desktop Chrome request
		req := httptest.NewRequest("GET", "/live", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/93.0.4577.63 Safari/537.36")

		// Test device detection
		shouldUseHLS := ShouldUseHLS(req)
		assert.False(t, shouldUseHLS, "Desktop should use FLV by default")

		format := GetStreamingFormat(req)
		assert.Equal(t, "flv", format, "Desktop should get FLV format")

		capabilities := DetectDeviceCapabilities(req)
		assert.False(t, capabilities.IsIOS, "Should not detect iOS")
		assert.True(t, capabilities.SupportsHLS, "Desktop should support HLS via hls.js")
		assert.Equal(t, "flv", capabilities.PreferredCodec, "Desktop should prefer FLV")
	})

	t.Run("Force HLS via Query Parameter", func(t *testing.T) {
		// Simulate desktop request with HLS forced
		req := httptest.NewRequest("GET", "/live?format=hls", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		shouldUseHLS := ShouldUseHLS(req)
		assert.True(t, shouldUseHLS, "Should use HLS when forced via query parameter")

		format := GetStreamingFormat(req)
		assert.Equal(t, "hls", format, "Should get HLS format when requested")
	})
}
