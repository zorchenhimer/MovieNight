package main

import (
	"net/http"
	"regexp"
	"strings"
)

// DeviceCapabilities represents the streaming capabilities of a device
type DeviceCapabilities struct {
	SupportsHLS    bool
	SupportsMPEGTS bool
	IsMobile       bool
	IsIOS          bool
	IsAndroid      bool
	UserAgent      string
	PreferredCodec string
}

// iOS user agent patterns for detection - prioritizing User Agent string
var iosPatterns = []*regexp.Regexp{
	// Primary iOS device patterns (prioritized)
	regexp.MustCompile(`(?i)iphone`),
	regexp.MustCompile(`(?i)ipad`),
	regexp.MustCompile(`(?i)ipod`),
	// macOS Safari patterns - also supports HLS natively
	regexp.MustCompile(`(?i)macintosh.*safari.*version`),
	regexp.MustCompile(`(?i)mac os x.*safari.*version`),
}

// Android user agent patterns
var androidPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)android`),
}

// Mobile device patterns
var mobilePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)mobile`),
	regexp.MustCompile(`(?i)tablet`),
	regexp.MustCompile(`(?i)iphone`),
	regexp.MustCompile(`(?i)ipad`),
	regexp.MustCompile(`(?i)ipod`),
	regexp.MustCompile(`(?i)android`),
	regexp.MustCompile(`(?i)blackberry`),
	regexp.MustCompile(`(?i)windows phone`),
}

// DetectDeviceCapabilities analyzes the HTTP request to determine device capabilities
// Prioritizes User Agent string detection as requested
func DetectDeviceCapabilities(r *http.Request) DeviceCapabilities {
	if r == nil {
		return DeviceCapabilities{
			SupportsHLS:    false,
			SupportsMPEGTS: true,
			IsMobile:       false,
			IsIOS:          false,
			IsAndroid:      false,
			UserAgent:      "",
			PreferredCodec: "flv",
		}
	}

	userAgent := r.Header.Get("User-Agent")

	capabilities := DeviceCapabilities{
		UserAgent: userAgent,
	}

	// Primary detection via User Agent string (prioritized as requested)
	// Detect iOS devices first
	for _, pattern := range iosPatterns {
		if pattern.MatchString(userAgent) {
			capabilities.IsIOS = true
			// iOS devices have native HLS support
			capabilities.SupportsHLS = true
			capabilities.SupportsMPEGTS = false // iOS Safari doesn't support MPEG-TS well
			capabilities.PreferredCodec = "hls"
			break
		}
	}

	// Detect Android devices
	if !capabilities.IsIOS { // Only check if not already detected as iOS
		for _, pattern := range androidPatterns {
			if pattern.MatchString(userAgent) {
				capabilities.IsAndroid = true
				// Android devices may support HLS via hls.js
				capabilities.SupportsHLS = true
				capabilities.SupportsMPEGTS = true
				capabilities.PreferredCodec = "hls" // Prefer HLS for mobile
				break
			}
		}
	}

	// Set defaults for Desktop or unknown devices
	if !capabilities.IsIOS && !capabilities.IsAndroid {
		// Desktop browsers - prefer MPEG-TS for better performance
		capabilities.SupportsHLS = true    // via hls.js
		capabilities.SupportsMPEGTS = true // via mpegts.js
		capabilities.PreferredCodec = "flv"
	}
	
	// Detect mobile devices -- there do exist some non-iOS/Android mobile devices
	for _, pattern := range mobilePatterns {
		if pattern.MatchString(userAgent) {
			capabilities.IsMobile = true
			break
		}
	}

	return capabilities
}

// ShouldUseHLS determines if HLS should be used for this request
func ShouldUseHLS(r *http.Request) bool {
	if r == nil {
		return false
	}

	capabilities := DetectDeviceCapabilities(r)

	// Use HLS for iOS devices as they have native support and better performance
	if capabilities.IsIOS {
		return true
	}

	// Check if explicitly requested via query parameter
	if r.URL.Query().Get("format") == "hls" {
		return true
	}

	// For other devices, use MPEG-TS by default for better performance
	return false
}

// GetStreamingFormat returns the preferred streaming format for the device
func GetStreamingFormat(r *http.Request) string {
	if ShouldUseHLS(r) {
		return "hls"
	}
	return "flv"
}

// GetAcceptHeader returns the appropriate Accept header for the streaming format
func GetAcceptHeader(format string) string {
	switch format {
	case "hls":
		return "application/vnd.apple.mpegurl"
	default:
		return "video/x-flv"
	}
}

// IsHLSPlaylistRequest checks if the request is for an HLS playlist
func IsHLSPlaylistRequest(r *http.Request) bool {
	if r == nil {
		return false
	}

	path := strings.ToLower(r.URL.Path)

	// Check if it's explicitly an m3u8 file
	if strings.HasSuffix(path, ".m3u8") {
		return true
	}

	// Check if it contains playlist in the path
	if strings.Contains(path, "playlist") {
		return true
	}

	// Check Accept header for HLS content type
	if r.Header.Get("Accept") == "application/vnd.apple.mpegurl" {
		return true
	}

	// Check if format=hls parameter is present (for /live?format=hls)
	if r.URL.Query().Get("format") == "hls" {
		return true
	}

	return false
}

// IsHLSSegmentRequest checks if the request is for an HLS segment
func IsHLSSegmentRequest(r *http.Request) bool {
	if r == nil {
		return false
	}

	path := strings.ToLower(r.URL.Path)
	return strings.HasSuffix(path, ".ts") && strings.Contains(path, "segment")
}

// GetContentTypeForFormat returns the appropriate Content-Type header for the format
func GetContentTypeForFormat(format string) string {
	switch format {
	case "hls":
		return "application/vnd.apple.mpegurl"
	case "ts":
		return "video/mp2t"
	default:
		return "video/x-flv"
	}
}

// ValidateUserAgent performs basic validation on the User-Agent string
func ValidateUserAgent(userAgent string) bool {
	if userAgent == "" {
		return false
	}

	// Basic validation - check for reasonable length and common patterns
	if len(userAgent) > 1000 {
		return false
	}

	lowerUA := strings.ToLower(userAgent)
	for _, pattern := range settings.UABotPatterns {
		if strings.Contains(lowerUA, pattern) {
			return false
		}
	}

	return true
}
