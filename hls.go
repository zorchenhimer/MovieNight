package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Eyevinn/hls-m3u8/m3u8"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format/ts"
	"github.com/zorchenhimer/MovieNight/common"
)

// HLSConfig represents configuration for HLS streaming
type HLSConfig struct {
	HLSVersion            int 		    // HLS version to use
	SegmentDuration       time.Duration // Duration of each segment
	MaxSegments           int           // Maximum number of segments to keep in memory
	TargetDuration        time.Duration // Target duration for playlist
	BitrateReduction      float64       // Bitrate reduction factor for HLS (0.0-1.0)
	EnableLowLatency      bool          // Enable low latency optimizations
	MaxConcurrentSegments int           // Maximum number of segments to generate concurrently
	SegmentBufferSize     int           // Buffer size for segment data
	QualityAdaptation     bool          // Enable adaptive quality based on device capabilities
}

// DefaultHLSConfig returns the default HLS configuration
func DefaultHLSConfig() HLSConfig {
	return HLSConfig{
		HLSVersion:            6,               // Use HLS version 6 for better support
		SegmentDuration:       4 * time.Second, // Shorter segments for lower latency
		MaxSegments:           6,               // Fewer segments for faster processing
		TargetDuration:        4 * time.Second, // Match segment duration
		BitrateReduction:      0.7,             // 30% reduction for HLS efficiency
		EnableLowLatency:      true,
		MaxConcurrentSegments: 4,          // More concurrent processing
		SegmentBufferSize:     512 * 1024, // Smaller buffer for faster processing
		QualityAdaptation:     true,
	}
}

// HLSQualitySettings represents quality settings for different device types
type HLSQualitySettings struct {
	BitrateMultiplier float64 // Multiplier for bitrate (1.0 = original, 0.7 = 30% reduction)
	Resolution        string  // Target resolution
	FrameRate         int     // Target frame rate
	KeyFrameInterval  int     // Key frame interval in seconds
}

// GetQualitySettings returns appropriate quality settings based on device capabilities
func GetQualitySettings(capabilities DeviceCapabilities) HLSQualitySettings {
	if capabilities.IsIOS {
		if capabilities.IsMobile {
			// iOS Mobile - optimize for battery and bandwidth
			return HLSQualitySettings{
				BitrateMultiplier: 0.7, // 30% reduction
				Resolution:        "720p",
				FrameRate:         30,
				KeyFrameInterval:  2,
			}
		} else {
			// iOS Desktop (macOS) - higher quality
			return HLSQualitySettings{
				BitrateMultiplier: 0.85, // 15% reduction
				Resolution:        "1080p",
				FrameRate:         60,
				KeyFrameInterval:  2,
			}
		}
	} else if capabilities.IsAndroid {
		// Android devices - balance quality and performance
		return HLSQualitySettings{
			BitrateMultiplier: 0.75, // 25% reduction
			Resolution:        "720p",
			FrameRate:         30,
			KeyFrameInterval:  2,
		}
	}

	// Desktop/Other - use default HLS settings with moderate reduction
	return HLSQualitySettings{
		BitrateMultiplier: 0.8, // 20% reduction for HLS overhead
		Resolution:        "1080p",
		FrameRate:         60,
		KeyFrameInterval:  2,
	}
}

// HLSViewerInfo represents viewer session information with activity tracking
type HLSViewerInfo struct {
	SessionID    string
	LastActivity time.Time
	FirstSeen    time.Time
	IsNew        bool // True if this is the first playlist request for this session
}

// HLSChannel represents an HLS stream with playlist and segments
type HLSChannel struct {
	que             *pubsub.Queue
	playlist        *m3u8.MediaPlaylist
	segments        []HLSSegment
	targetDuration  time.Duration
	sequenceNumber  uint64
	mutex           sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	segmentDuration time.Duration
	maxSegments     int
	viewers         map[string]*HLSViewerInfo // Track HLS viewers with timestamps
	viewersMutex    sync.RWMutex
	config          HLSConfig
	cleanupTicker   *time.Ticker // Background cleanup ticker
}

// HLSSegment represents a single HLS segment
type HLSSegment struct {
	URI      string
	Duration float64
	Data     []byte
	Sequence uint64
}

// NewHLSChannel creates a new HLS channel
func NewHLSChannel(que *pubsub.Queue) (*HLSChannel, error) {
	if que == nil {
		return nil, fmt.Errorf("queue cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	config := DefaultHLSConfig()

	// Create playlist with sliding window for live streaming
	// Important: Use the proper pattern for sliding window
	windowSize := uint(config.MaxSegments)
	playlist, err := m3u8.NewMediaPlaylist(windowSize, windowSize)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create playlist: %w", err)
	}

	// Set playlist properties for optimal HLS performance and sliding window
	playlist.SetVersion(config.HLSVersion)
	playlist.Closed = false // Keep playlist open for live streaming (sliding window)

	hls := &HLSChannel{
		que:             que,
		playlist:        playlist,
		segments:        make([]HLSSegment, 0),
		targetDuration:  config.TargetDuration,
		sequenceNumber:  0,
		ctx:             ctx,
		cancel:          cancel,
		segmentDuration: config.SegmentDuration,
		maxSegments:     config.MaxSegments,
		viewers:         make(map[string]*HLSViewerInfo),
		config:          config,
		cleanupTicker:   time.NewTicker(10 * time.Second), // Run cleanup every 10 seconds
	}

	// Start background cleanup routine
	go hls.startViewerCleanup()

	return hls, nil
}

// NewHLSChannelWithDeviceOptimization creates a new HLS channel optimized for specific device capabilities
func NewHLSChannelWithDeviceOptimization(que *pubsub.Queue, r *http.Request) (*HLSChannel, error) {
	if que == nil {
		return nil, fmt.Errorf("queue cannot be nil")
	}

	// Detect device capabilities for optimization
	capabilities := DeviceCapabilities{}
	if r != nil {
		capabilities = DetectDeviceCapabilities(r)
	}

	qualitySettings := GetQualitySettings(capabilities)

	ctx, cancel := context.WithCancel(context.Background())

	config := DefaultHLSConfig()

	// Apply device-specific optimizations
	config.BitrateReduction = qualitySettings.BitrateMultiplier

	if capabilities.IsIOS && capabilities.IsMobile {
		// Optimize for iOS mobile devices - prioritize low latency
		config.SegmentDuration = 3 * time.Second // Very short segments for low latency
		config.MaxSegments = 5                   // Minimal segments for fast processing
		config.EnableLowLatency = true
		config.MaxConcurrentSegments = 3 // Balanced concurrency for mobile
	} else if capabilities.IsAndroid {
		// Optimize for Android devices
		config.SegmentDuration = 4 * time.Second // Short segments for good performance
		config.MaxSegments = 6
		config.MaxConcurrentSegments = 3
	} else {
		// Desktop optimization - can handle slightly longer segments
		config.SegmentDuration = 4 * time.Second // Still short for low latency
		config.MaxSegments = 8
		config.MaxConcurrentSegments = 5
	}

	// Create playlist with device-optimized settings for proper sliding window
	windowSize := uint(config.MaxSegments)
	playlist, err := m3u8.NewMediaPlaylist(windowSize, windowSize)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create optimized playlist: %w", err)
	}
	playlist.SetVersion(config.HLSVersion)
	playlist.Closed = false // Keep playlist open for live streaming (sliding window)

	hls := &HLSChannel{
		que:             que,
		playlist:        playlist,
		segments:        make([]HLSSegment, 0),
		targetDuration:  config.TargetDuration,
		sequenceNumber:  0,
		ctx:             ctx,
		cancel:          cancel,
		segmentDuration: config.SegmentDuration,
		maxSegments:     config.MaxSegments,
		viewers:         make(map[string]*HLSViewerInfo),
		config:          config,
		cleanupTicker:   time.NewTicker(10 * time.Second), // Run cleanup every 10 seconds
	}

	// Start background cleanup routine
	go hls.startViewerCleanup()

	common.LogDebugf("Created HLS channel optimized for device: iOS=%v, Mobile=%v, BitrateReduction=%.2f\n",
		capabilities.IsIOS, capabilities.IsMobile, config.BitrateReduction)

	return hls, nil
}

// Start begins HLS segment generation
func (h *HLSChannel) Start() error {
	if h == nil {
		return fmt.Errorf("HLS channel is nil")
	}

	go h.generateSegments()
	return nil
}

// Stop stops HLS segment generation
func (h *HLSChannel) Stop() {
	h.Close()
}

// generateSegments continuously generates HLS segments from the stream using proper TS muxing
func (h *HLSChannel) generateSegments() {
	if h == nil || h.que == nil {
		common.LogErrorln("Cannot generate segments: HLS channel or queue is nil")
		return
	}

	cursor := h.que.Latest()
	if cursor == nil {
		common.LogErrorln("Cannot get latest cursor from queue")
		return
	}

	// Create segments at regular intervals using TS muxer
	segmentTimer := time.NewTicker(h.segmentDuration)
	defer segmentTimer.Stop()

	var currentSegmentBuffer bytes.Buffer
	var segmentStartTime time.Time
	var tsMuxer *ts.Muxer

	// Initialize first segment
	h.startNewSegment(&currentSegmentBuffer, &tsMuxer, &segmentStartTime)

	for {
		select {
		case <-h.ctx.Done():
			// Finalize any pending segment before exiting
			if currentSegmentBuffer.Len() > 0 {
				h.finalizeSegment(&currentSegmentBuffer, time.Since(segmentStartTime))
			}
			return

		case <-segmentTimer.C:
			// Time-based segmentation - create segment every interval
			if currentSegmentBuffer.Len() > 0 {
				h.finalizeSegment(&currentSegmentBuffer, time.Since(segmentStartTime))
				h.startNewSegment(&currentSegmentBuffer, &tsMuxer, &segmentStartTime)
			}

		default:
			// Read and process stream data
			packet, err := cursor.ReadPacket()
			if err != nil {
				if err != io.EOF {
					common.LogErrorf("Error reading from stream cursor: %v\n", err)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Write packet to current segment via TS muxer
			if tsMuxer != nil {
				err = tsMuxer.WritePacket(packet)
				if err != nil {
					common.LogErrorf("Error writing packet to TS muxer: %v\n", err)
					continue
				}
			}

			// Check if segment is getting too large (fallback protection)
			if currentSegmentBuffer.Len() > 2*1024*1024 { // 2MB limit
				common.LogDebugf("Segment size limit reached, creating segment\n")
				h.finalizeSegment(&currentSegmentBuffer, time.Since(segmentStartTime))
				h.startNewSegment(&currentSegmentBuffer, &tsMuxer, &segmentStartTime)
			}
		}
	}
}

// startNewSegment initializes a new segment with TS muxer
func (h *HLSChannel) startNewSegment(buffer *bytes.Buffer, muxer **ts.Muxer, startTime *time.Time) {
	buffer.Reset()
	*startTime = time.Now()

	// Create new TS muxer that writes to our buffer
	newMuxer := ts.NewMuxer(buffer)

	// Get the streams from the original queue to initialize the muxer
	if h.que != nil {
		// Get stream headers from the queue
		cursor := h.que.Latest()
		if cursor != nil {
			// Try to get the stream headers that were written to the queue
			streams, err := cursor.Streams()
			if err == nil && streams != nil && len(streams) > 0 {
				// Write stream headers to the TS muxer
				err := newMuxer.WriteHeader(streams)
				if err != nil {
					common.LogErrorf("Failed to write stream headers to TS muxer: %v\n", err)
					// Fall back to creating muxer without headers, but this may cause issues
				} else {
					common.LogDebugf("TS muxer initialized with %d streams\n", len(streams))
				}
			} else {
				common.LogDebugf("No streams available for TS muxer initialization\n")
			}
		}
	}

	*muxer = newMuxer

	common.LogDebugf("Started new HLS segment\n")
}

// finalizeSegment completes the current segment and adds it to the playlist
func (h *HLSChannel) finalizeSegment(buffer *bytes.Buffer, duration time.Duration) {
	if buffer.Len() == 0 {
		return
	}

	// Create a copy of the buffer data
	segmentData := make([]byte, buffer.Len())
	copy(segmentData, buffer.Bytes())

	currentSeq := h.sequenceNumber
	h.sequenceNumber++

	// Generate unique segment ID to avoid browser caching issues across service restarts
	segmentID := generateSegmentID()
	segmentURI := fmt.Sprintf("/live/segment_%s.ts", segmentID)
	durationSeconds := duration.Seconds()

	segment := HLSSegment{
		URI:      segmentURI,
		Duration: durationSeconds,
		Data:     segmentData,
		Sequence: currentSeq, // Keep sequence for internal ordering
	}

	// Add segment with proper sliding window management
	h.addGeneratedSegment(segment)

	common.LogDebugf("Finalized HLS segment %d with %d bytes, duration %.2fs\n",
		currentSeq, len(segmentData), durationSeconds)
}

// createSegment creates a new HLS segment
// addGeneratedSegment adds a generated segment to the HLS channel with proper sliding window
func (h *HLSChannel) addGeneratedSegment(segment HLSSegment) {
	if h == nil || h.playlist == nil {
		return
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Add segment to our local list with sliding window management
	h.segments = append(h.segments, segment)

	// Remove old segments if we exceed max (manual sliding window for our data)
	if len(h.segments) > h.maxSegments {
		// Remove oldest segments to maintain window size
		excess := len(h.segments) - h.maxSegments
		h.segments = h.segments[excess:]
	}

	// If this is the first segment, set the initial MediaSequence
	if h.playlist.Count() == 0 {
		h.playlist.SeqNo = segment.Sequence
	}

	// For proper sliding window, we need to manually manage the playlist size
	// If the playlist is at max capacity, we need to remove the oldest segment first
	if int(h.playlist.Count()) >= h.maxSegments {
		// Create a new playlist and copy the recent segments
		newPlaylist, err := m3u8.NewMediaPlaylist(uint(h.maxSegments), uint(h.maxSegments))
		if err != nil {
			common.LogErrorf("Failed to create new playlist for sliding window: %v\n", err)
			return
		}
		newPlaylist.SetVersion(config.HLSVersion)
		newPlaylist.Closed = false

		// Add only the segments that should remain (excluding the oldest one)
		segmentsToKeep := h.maxSegments - 1 // Leave room for the new segment
		startIdx := len(h.segments) - segmentsToKeep
		if startIdx < 0 {
			startIdx = 0
		}

		// Set the media sequence to match the first segment that will be in the new playlist
		if startIdx < len(h.segments) {
			newPlaylist.SeqNo = h.segments[startIdx].Sequence
		}

		for i := startIdx; i < len(h.segments)-1; i++ { // -1 because we haven't added the new segment yet
			seg := h.segments[i]
			newPlaylist.Append(seg.URI, seg.Duration, "")
		}

		// Replace the old playlist
		h.playlist = newPlaylist
	}

	// Now add the new segment
	h.playlist.Append(segment.URI, segment.Duration, "")

	// Update target duration if needed
	duration := time.Duration(segment.Duration * float64(time.Second))
	if duration > h.targetDuration {
		h.targetDuration = duration
		h.playlist.TargetDuration = uint(segment.Duration)
	}

	common.LogDebugf("Added generated HLS segment %d with duration %.2fs (playlist count: %d/%d)\n",
		segment.Sequence, segment.Duration, h.playlist.Count(), h.maxSegments)
}

// GetPlaylist returns the current m3u8 playlist
func (h *HLSChannel) GetPlaylist() string {
	if h == nil || h.playlist == nil {
		return ""
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Set final playlist properties
	h.playlist.TargetDuration = uint(h.targetDuration.Seconds())

	return h.playlist.String()
}

// HasSegments returns true if the playlist has any segments
func (h *HLSChannel) HasSegments() bool {
	if h == nil {
		return false
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return len(h.segments) > 0
}

// GetSegment returns a specific segment by sequence number
func (h *HLSChannel) GetSegment(sequence uint64) ([]byte, error) {
	if h == nil {
		return nil, fmt.Errorf("HLS channel is nil")
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for _, segment := range h.segments {
		if segment.Sequence == sequence {
			return segment.Data, nil
		}
	}

	return nil, fmt.Errorf("segment %d not found", sequence)
}

// GetSegmentByURI returns a segment by its URI
func (h *HLSChannel) GetSegmentByURI(uri string) ([]byte, error) {
	if h == nil {
		return nil, fmt.Errorf("HLS channel is nil")
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for _, segment := range h.segments {
		if segment.URI == uri {
			return segment.Data, nil
		}
	}

	return nil, fmt.Errorf("segment with URI %s not found", uri)
}

// AddViewer adds an HLS viewer for tracking, returns true if this is a new viewer
func (h *HLSChannel) AddViewer(sessionID string) bool {
	if h == nil {
		return false
	}

	h.viewersMutex.Lock()
	defer h.viewersMutex.Unlock()

	now := time.Now()
	isNew := false

	if existing, exists := h.viewers[sessionID]; exists {
		// Update last activity for existing viewer
		existing.LastActivity = now
		existing.IsNew = false
	} else {
		// New viewer
		h.viewers[sessionID] = &HLSViewerInfo{
			SessionID:    sessionID,
			LastActivity: now,
			FirstSeen:    now,
			IsNew:        true,
		}
		isNew = true
		common.LogInfof("[HLS] New viewer added: %s\n", sessionID)
	}

	return isNew
}

// RemoveViewer removes an HLS viewer
func (h *HLSChannel) RemoveViewer(sessionID string) {
	if h == nil {
		return
	}

	h.viewersMutex.Lock()
	defer h.viewersMutex.Unlock()

	if _, exists := h.viewers[sessionID]; exists {
		delete(h.viewers, sessionID)
		common.LogInfof("[HLS] Viewer removed: %s\n", sessionID)
	}
}

// GetViewerCount returns the number of HLS viewers
func (h *HLSChannel) GetViewerCount() int {
	if h == nil {
		return 0
	}

	h.viewersMutex.RLock()
	defer h.viewersMutex.RUnlock()

	return len(h.viewers)
}

// startViewerCleanup runs background cleanup to remove inactive viewers
func (h *HLSChannel) startViewerCleanup() {
	if h == nil || h.cleanupTicker == nil {
		return
	}

	for {
		select {
		case <-h.ctx.Done():
			h.cleanupTicker.Stop()
			return
		case <-h.cleanupTicker.C:
			h.cleanupInactiveViewers()
		}
	}
}

// cleanupInactiveViewers removes viewers that have been inactive for more than 30 seconds
func (h *HLSChannel) cleanupInactiveViewers() {
	if h == nil {
		return
	}

	h.viewersMutex.Lock()
	defer h.viewersMutex.Unlock()

	now := time.Now()
	inactiveThreshold := 30 * time.Second
	removedViewers := make([]string, 0)

	for sessionID, viewer := range h.viewers {
		if now.Sub(viewer.LastActivity) > inactiveThreshold {
			removedViewers = append(removedViewers, sessionID)
			delete(h.viewers, sessionID)
		}
	}

	// Log cleanup results
	if len(removedViewers) > 0 {
		common.LogInfof("[HLS] Cleaned up %d inactive viewers: %v\n", len(removedViewers), removedViewers)

		// Remove from global stats as well
		for _, sessionID := range removedViewers {
			stats.removeViewer(sessionID)
		}
	}
}

// Close properly shuts down the HLS channel and cleanup routines
func (h *HLSChannel) Close() {
	if h == nil {
		return
	}

	if h.cancel != nil {
		h.cancel()
	}

	if h.cleanupTicker != nil {
		h.cleanupTicker.Stop()
	}

	// Clean up all viewers
	h.viewersMutex.Lock()
	for sessionID := range h.viewers {
		stats.removeViewer(sessionID)
	}
	h.viewers = make(map[string]*HLSViewerInfo)
	h.viewersMutex.Unlock()

	common.LogInfof("[HLS] Channel closed and all viewers cleaned up\n")
}

// generateSegmentID creates a unique segment identifier to avoid browser caching issues
// Uses UUID4 format for maximum collision avoidance across service restarts
func generateSegmentID() string {
	// Generate 16 random bytes for UUID4
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Set version (4) and variant bits for proper UUID4 format
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant 10

	// Format as UUID string (shortened for segment names)
	return fmt.Sprintf("%x%x%x%x%x%x%x%x",
		uuid[0:2], uuid[2:4], uuid[4:6], uuid[6:8],
		uuid[8:10], uuid[10:12], uuid[12:14], uuid[14:16])
}

// IsValidSegmentURI checks if a segment URI is valid
func IsValidSegmentURI(uri string) bool {
	if uri == "" {
		return false
	}

	// Check if it's a .ts segment
	if !strings.HasSuffix(uri, ".ts") {
		return false
	}

	// Extract the filename from the URI (handle both relative and absolute paths)
	filename := uri
	if strings.Contains(uri, "/") {
		parts := strings.Split(uri, "/")
		filename = parts[len(parts)-1]
	}

	// Check if filename matches segment pattern
	if !strings.HasPrefix(filename, "segment_") {
		return false
	}

	// Extract identifier and validate
	name := strings.TrimSuffix(filename, ".ts")
	parts := strings.Split(name, "_")
	if len(parts) != 2 {
		return false
	}

	identifier := parts[1]

	// Support both old numeric format (for backward compatibility) and new UUID format
	if _, err := strconv.ParseUint(identifier, 10, 64); err == nil {
		// Valid numeric sequence (legacy format)
		return true
	}

	// Check for UUID format: 32 hex characters
	if len(identifier) == 32 {
		for _, c := range identifier {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return true
	}

	return false
}

// ParseSequenceFromURI extracts sequence number from segment URI
// Note: This function is deprecated for UUID-based segments and maintained for backward compatibility
func ParseSequenceFromURI(uri string) (uint64, error) {
	if !IsValidSegmentURI(uri) {
		return 0, fmt.Errorf("invalid segment URI: %s", uri)
	}

	// Extract the filename from the URI (handle both relative and absolute paths)
	filename := uri
	if strings.Contains(uri, "/") {
		parts := strings.Split(uri, "/")
		filename = parts[len(parts)-1]
	}

	// Extract identifier from "segment_IDENTIFIER.ts"
	name := strings.TrimSuffix(filename, ".ts")
	parts := strings.Split(name, "_")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid segment URI format: %s", uri)
	}

	identifier := parts[1]

	// Try to parse as numeric sequence (legacy format)
	if sequence, err := strconv.ParseUint(identifier, 10, 64); err == nil {
		return sequence, nil
	}

	// For UUID-based segments, sequence number is not available from URI
	return 0, fmt.Errorf("sequence number not available for UUID-based segment URI: %s", uri)
}
