/// <reference path='./both.js' />

/**
 * HLS Debug Logging Management
 * 
 * Debug logging is controlled via sessionStorage and URL parameters:
 * - Enable debug: Add ?debug=enable to URL, or run `sessionStorage.setItem('MovieNight-HLS-Debug', 'true')` in console
 * - Disable debug: Add ?debug=disable to URL, or run `sessionStorage.removeItem('MovieNight-HLS-Debug')` in console
 * - Check status: Run `sessionStorage.getItem('MovieNight-HLS-Debug')` in console
 * 
 * Debug logging includes:
 * - HLS player initialization and configuration
 * - Fragment loading and buffering events
 * - Quality switching and adaptive bitrate events
 * - Device detection and format selection
 * - Buffer status and statistics
 * 
 * Note: Errors and critical warnings are always logged regardless of debug setting.
 */

// Debug logging management
const DEBUG_STORAGE_KEY = 'MovieNight-HLS-Debug';

// Check URL parameters for debug control
function initializeDebugMode() {
    const urlParams = new URLSearchParams(window.location.search);
    const debugParam = urlParams.get('debug');
    
    if (debugParam === 'enable') {
        sessionStorage.setItem(DEBUG_STORAGE_KEY, 'true');
        console.log('HLS debug logging enabled via URL parameter');
    } else if (debugParam === 'disable') {
        sessionStorage.removeItem(DEBUG_STORAGE_KEY);
        console.log('HLS debug logging disabled via URL parameter');
    }
}

// Check if debug logging is enabled
function isDebugEnabled() {
    return sessionStorage.getItem(DEBUG_STORAGE_KEY) === 'true';
}

// Debug logging wrapper
function debugLog(...args) {
    if (isDebugEnabled()) {
        console.log('[HLS Debug]', ...args);
    }
}

function debugWarn(...args) {
    if (isDebugEnabled()) {
        console.warn('[HLS Debug]', ...args);
    }
}

function debugError(...args) {
    if (isDebugEnabled()) {
        console.error('[HLS Debug]', ...args);
    }
}

// Initialize debug mode on page load
document.addEventListener('DOMContentLoaded', initializeDebugMode);

// Device detection utilities - prioritizing User Agent string
function isIOS() {
    const userAgent = navigator.userAgent;
    
    // Primary detection via User Agent string (prioritized as requested)
    if (/iPad|iPhone|iPod/.test(userAgent) && !window.MSStream) {
        return true;
    }
    
    // Additional iOS detection patterns
    if (/Macintosh.*Safari/.test(userAgent) && /Version\//.test(userAgent)) {
        // macOS Safari - also supports native HLS
        return true;
    }
    
    return false;
}

function isMobile() {
    const userAgent = navigator.userAgent;
    
    // Primary detection via User Agent string (prioritized as requested)
    return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini|Mobile|Tablet/i.test(userAgent);
}

function isAndroid() {
    const userAgent = navigator.userAgent;
    
    // Primary detection via User Agent string (prioritized as requested)
    return /Android/i.test(userAgent);
}

function supportsHLS() {
    // First check User Agent for known HLS support
    const userAgent = navigator.userAgent;
    if (/iPad|iPhone|iPod|Macintosh.*Safari/.test(userAgent)) {
        return true; // iOS and macOS Safari have native HLS support
    }
    
    // Fallback to feature detection only if User Agent doesn't give clear answer
    // Gracefully handle browsers that don't support HLS
    try {
        const video = document.createElement('video');
        return video.canPlayType('application/vnd.apple.mpegurl') !== '';
    } catch (e) {
        // Silently handle the error - this is expected for browsers without HLS support
        // No need to log since we have a fallback strategy
        return false;
    }
}

function shouldUseHLS() {
    // Check URL parameters first (explicit override)
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('format') === 'hls') {
        return true;
    }
    
    // Force HLS for iOS devices (prioritizing User Agent detection)
    if (isIOS()) {
        debugLog('iOS detected via User Agent, using HLS');
        return true;
    }
    
    // For Android devices, also prefer HLS if supported
    if (isAndroid() && typeof Hls !== 'undefined' && Hls.isSupported()) {
        debugLog('Android detected with HLS.js support, using HLS');
        return true;
    }
    
    // Default to FLV for desktop for better performance
    return false;
}

function initPlayer() {
    debugLog('initPlayer: User Agent:', navigator.userAgent);
    debugLog('initPlayer: isIOS():', isIOS());
    debugLog('initPlayer: isAndroid():', isAndroid());
    debugLog('initPlayer: isMobile():', isMobile());
    debugLog('initPlayer: supportsHLS():', supportsHLS());
    
    const useHLS = shouldUseHLS();
    debugLog('initPlayer: shouldUseHLS():', useHLS);
    
    if (useHLS) {
        initHLSPlayer();
    } else {
        initMPEGTSPlayer();
    }
}

function initHLSPlayer() {
    debugLog('Initializing HLS player');
    
    let videoElement = document.querySelector('#videoElement');
    const hlsSource = '/live?format=hls';
    
    // Check for native HLS support (iOS Safari)
    if (supportsHLS()) {
        debugLog('Using native HLS support');
        videoElement.src = hlsSource;
        videoElement.addEventListener('loadedmetadata', function() {
            videoElement.play().catch(e => {
                debugWarn('Autoplay failed:', e);
            });
        });
        
        // Add error handling for native HLS
        videoElement.addEventListener('error', function(e) {
            console.error('Native HLS error:', e); // Keep error always visible
            // Fallback to hls.js or MPEG-TS
            if (typeof Hls !== 'undefined' && Hls.isSupported()) {
                debugLog('Falling back to hls.js');
                initHLSWithLibrary(videoElement, hlsSource);
            } else {
                debugLog('Falling back to MPEG-TS');
                initMPEGTSPlayer();
            }
        });
    } 
    // Use hls.js for browsers without native HLS support
    else if (typeof Hls !== 'undefined' && Hls.isSupported()) {
        debugLog('Using hls.js');
        debugLog('Hls object:', Hls);
        debugLog('Hls.isSupported():', Hls.isSupported());
        debugLog('Hls version:', Hls.version || 'version unknown');
        initHLSWithLibrary(videoElement, hlsSource);
    } else {
        console.warn('HLS not supported, falling back to MPEG-TS'); // Keep warning always visible
        if (typeof Hls === 'undefined') {
            console.error('HLS.js library not loaded');
        } else {
            console.error('HLS.js not supported on this browser');
        }
        initMPEGTSPlayer();
    }
    
    setupVideoOverlay();
}

function initHLSWithLibrary(videoElement, hlsSource) {
    debugLog('Initializing HLS.js with source:', hlsSource);
    
    // Clean up any existing HLS instance first
    if (window.hlsPlayer) {
        debugLog('Cleaning up existing HLS instance');
        try {
            window.hlsPlayer.destroy();
        } catch (e) {
            debugWarn('Error destroying previous HLS instance:', e);
        }
        window.hlsPlayer = null;
    }
    
    // Clear video element state to prevent cache-related issues
    try {
        videoElement.pause();
        videoElement.removeAttribute('src');
        videoElement.load();
        // Clear any existing source buffers
        if (videoElement.srcObject) {
            videoElement.srcObject = null;
        }
    } catch (e) {
        debugWarn('Error clearing video element state:', e);
    }
    
    // Check if debug mode is enabled
    const debugModeEnabled = isDebugEnabled();
    
    // Modern HLS.js configuration using policy-based approach (v1.6.12+)
    const hlsConfig = {
        // Core settings - only use well-supported options
        debug: debugModeEnabled,               // Enable debug only when explicitly requested
        enableWorker: true,                 // Enable worker for better performance
        lowLatencyMode: true,               // Enable low latency mode
        
        // Buffer management - conservative settings
        maxBufferLength: 30,                // Maximum forward buffer (30s)
        maxMaxBufferLength: 60,             // Absolute maximum buffer (60s)
        
        // Live streaming optimizations - basic settings
        liveSyncDurationCount: 3,           // Segments to keep from live edge
        liveMaxLatencyDurationCount: 5,     // Max latency in segments
        
        // Modern policy-based network configuration (replaces deprecated timeout settings)
        manifestLoadPolicy: {
            default: {
                maxTimeToFirstByteMs: 10000,
                maxLoadTimeMs: 10000,
                timeoutRetry: {
                    maxNumRetry: 3,
                    retryDelayMs: 0,
                    maxRetryDelayMs: 0
                },
                errorRetry: {
                    maxNumRetry: 3,
                    retryDelayMs: 1000,
                    maxRetryDelayMs: 8000
                }
            }
        },
        
        playlistLoadPolicy: {
            default: {
                maxTimeToFirstByteMs: 10000,
                maxLoadTimeMs: 10000,
                timeoutRetry: {
                    maxNumRetry: 2,
                    retryDelayMs: 0,
                    maxRetryDelayMs: 0
                },
                errorRetry: {
                    maxNumRetry: 2,
                    retryDelayMs: 1000,
                    maxRetryDelayMs: 8000
                }
            }
        },
        
        fragLoadPolicy: {
            default: {
                maxTimeToFirstByteMs: 20000,
                maxLoadTimeMs: 20000,
                timeoutRetry: {
                    maxNumRetry: 4,
                    retryDelayMs: 0,
                    maxRetryDelayMs: 0
                },
                errorRetry: {
                    maxNumRetry: 6,
                    retryDelayMs: 1000,
                    maxRetryDelayMs: 8000
                }
            }
        },
        
        // Quality settings
        startLevel: -1,                     // Auto start level
        capLevelToPlayerSize: true,         // Cap quality to player size
        
        // Basic features
        autoStartLoad: true,                // Auto start loading
        startPosition: -1                   // Start from live edge
    };

    var hls;
    try {
        // Try with our configuration first
        debugLog('Creating HLS instance with config:', hlsConfig);
        hls = new Hls(hlsConfig);
    } catch (e) {
        console.error('Failed to create HLS with config, trying minimal config:', e); // Keep error always visible
        // Fallback to minimal configuration
        try {
            hls = new Hls({
                debug: isDebugEnabled,
                lowLatencyMode: true,
                autoStartLoad: true
            });
        } catch (e2) {
            console.error('Failed to create HLS with minimal config, trying default:', e2); // Keep error always visible
            // Last resort - use completely default configuration
            hls = new Hls({
                debug: isDebugEnabled
            });
        }
    }
    
    // Store reference for cleanup
    window.hlsPlayer = hls;
    
    // Load and attach media - proper order is important
    debugLog('Attaching HLS to video element');
    hls.attachMedia(videoElement);
    
    // Wait for media to be attached before loading source
    hls.on(Hls.Events.MEDIA_ATTACHED, function() {
        debugLog('Media attached, now loading source:', hlsSource);
        hls.loadSource(hlsSource);
    });
    
    // Event listeners for comprehensive error handling and monitoring
    
    // Manifest events
    hls.on(Hls.Events.MANIFEST_LOADING, function(event, data) {
        debugLog('Loading HLS manifest from:', data.url);
    });
    
    hls.on(Hls.Events.MANIFEST_LOADED, function(event, data) {
        debugLog('HLS manifest loaded, levels:', data.levels.length);
        logAvailableQualities(data.levels);
    });
    
    hls.on(Hls.Events.MANIFEST_PARSED, function(event, data) {
        debugLog('HLS manifest parsed, starting playback');
        debugLog('Available levels:', data.levels.length);
        debugLog('Audio tracks:', data.audioTracks?.length || 0);
        debugLog('Subtitle tracks:', data.subtitleTracks?.length || 0);
        
        // Auto-play with better error handling
        videoElement.play().catch(e => {
            debugWarn('Autoplay failed:', e);
            // Show click-to-play overlay if autoplay fails
            showPlayButton();
        });
    });
    
    // Add comprehensive error handling
    hls.on(Hls.Events.ERROR, function(event, data) {
        console.error('HLS error:', data); // Keep errors always visible
        
        if (data.fatal) {
            handleFatalError(hls, data, videoElement);
        } else {
            handleNonFatalError(hls, data);
        }
    });
    
    // Level (quality) events
    hls.on(Hls.Events.LEVEL_SWITCHING, function(event, data) {
        debugLog('Switching to level:', data.level);
    });
    
    hls.on(Hls.Events.LEVEL_SWITCHED, function(event, data) {
        debugLog('Switched to level:', data.level);
        updateQualityIndicator(data.level);
    });
    
    // Fragment events for monitoring
    hls.on(Hls.Events.FRAG_LOADING, function(event, data) {
        debugLog('Loading fragment:', data.frag.sn);
    });
    
    hls.on(Hls.Events.FRAG_LOADED, function(event, data) {
        debugLog('Fragment loaded:', data.frag.sn, 'duration:', data.frag.duration);
    });
    
    hls.on(Hls.Events.FRAG_BUFFERED, function(event, data) {
        debugLog('Fragment buffered:', data.frag.sn);
        updateBufferIndicator();
    });
    
    // Add quality control interface
    addQualityControls(hls);
    
    // Add buffer monitoring
    addBufferMonitoring(hls, videoElement);
}

// Enhanced error handling functions
function handleFatalError(hls, data, videoElement) {
    switch(data.type) {
        case Hls.ErrorTypes.NETWORK_ERROR:
            console.log('Fatal network error, attempting recovery...'); // Keep recovery attempts visible
            // For network errors, try standard recovery first
            try {
                hls.startLoad();
            } catch (e) {
                debugWarn('Standard recovery failed, trying full restart');
                recoverHLSPlayer();
            }
            break;
            
        case Hls.ErrorTypes.MEDIA_ERROR:
            console.log('Fatal media error, attempting recovery...'); // Keep recovery attempts visible
            // For media errors, especially buffer-related issues, try media recovery
            try {
                hls.recoverMediaError();
            } catch (e) {
                debugWarn('Media recovery failed, trying full restart');
                recoverHLSPlayer();
            }
            break;
            
        case Hls.ErrorTypes.MUX_ERROR:
            console.log('Fatal mux error - likely cache-related, restarting player...'); // Keep recovery attempts visible
            // Mux errors often indicate cached/stale content, restart completely
            recoverHLSPlayer();
            break;
            
        case Hls.ErrorTypes.OTHER_ERROR:
            console.error('Fatal other error, cannot recover:', data);
            // Fallback to MPEG-TS
            setTimeout(() => {
                debugLog('Falling back to MPEG-TS player');
                cleanup();
                initMPEGTSPlayer();
            }, 1000);
            break;
            
        default:
            console.error('Unknown fatal error:', data);
            hls.destroy();
            initMPEGTSPlayer();
            break;
    }
}

function handleNonFatalError(hls, data) {
    debugWarn('Non-fatal HLS error:', data.type, data.details);
    
    // Log specific error details for debugging
    switch(data.details) {
        case Hls.ErrorDetails.MANIFEST_LOAD_ERROR:
        case Hls.ErrorDetails.MANIFEST_LOAD_TIMEOUT:
            debugWarn('Manifest loading issue - possibly no active stream');
            // Check if it's a 503 Service Unavailable (no stream active)
            if (data.response && data.response.code === 503) {
                console.info('No active stream available. Waiting for stream to start...'); // Keep stream status visible
                showNoStreamMessage();
            }
            break;
            
        case Hls.ErrorDetails.LEVEL_LOAD_ERROR:
        case Hls.ErrorDetails.LEVEL_LOAD_TIMEOUT:
            debugWarn('Level loading issue');
            break;
            
        case Hls.ErrorDetails.FRAG_LOAD_ERROR:
        case Hls.ErrorDetails.FRAG_LOAD_TIMEOUT:
            debugWarn('Fragment loading issue');
            break;
            
        case Hls.ErrorDetails.BUFFER_APPEND_ERROR:
            debugWarn('Buffer append issue - attempting buffer flush');
            // Try to recover from buffer append errors by flushing buffers
            if (window.hlsPlayer && window.hlsPlayer.media) {
                try {
                    window.hlsPlayer.trigger(Hls.Events.BUFFER_FLUSHED);
                } catch (e) {
                    debugWarn('Error flushing buffer:', e);
                }
            }
            break;
            
        case Hls.ErrorDetails.BUFFER_FULL_ERROR:
            debugWarn('Buffer full error - forcing buffer cleanup');
            // Handle buffer full errors more aggressively
            if (window.hlsPlayer) {
                try {
                    // Try to flush all buffers and restart
                    const videoElement = window.hlsPlayer.media;
                    if (videoElement) {
                        const currentTime = videoElement.currentTime;
                        window.hlsPlayer.trigger(Hls.Events.BUFFER_FLUSHED);
                        // Seek to current position to force buffer refresh
                        videoElement.currentTime = currentTime;
                    }
                } catch (e) {
                    debugWarn('Error handling buffer full:', e);
                }
            }
            break;
            
        default:
            debugWarn('Other non-fatal error:', data.details);
    }
}

// Utility functions for enhanced features
function logAvailableQualities(levels) {
    debugLog('Available quality levels:');
    levels.forEach((level, index) => {
        debugLog(`  ${index}: ${level.width}x${level.height} @ ${Math.round(level.bitrate/1000)}kbps`);
    });
}

function updateQualityIndicator(levelIndex) {
    // Update UI to show current quality
    const indicator = document.querySelector('#qualityIndicator');
    if (indicator && window.hlsPlayer) {
        const levels = window.hlsPlayer.levels;
        if (levels && levels[levelIndex]) {
            const level = levels[levelIndex];
            indicator.textContent = `${level.height}p`;
        }
    }
}

function updateBufferIndicator() {
    const videoElement = document.querySelector('#videoElement');
    if (videoElement && videoElement.buffered.length > 0) {
        const buffered = videoElement.buffered.end(videoElement.buffered.length - 1);
        const current = videoElement.currentTime;
        const bufferAhead = buffered - current;
        
        debugLog(`Buffer: ${bufferAhead.toFixed(1)}s ahead`);
        
        // Update buffer indicator in UI
        const indicator = document.querySelector('#bufferIndicator');
        if (indicator) {
            indicator.textContent = `Buffer: ${bufferAhead.toFixed(1)}s`;
        }
    }
}

function showPlayButton() {
    // Show a play button overlay for manual playback initiation
    const overlay = document.querySelector('#videoOverlay');
    if (overlay) {
        overlay.style.display = 'block';
        overlay.innerHTML = '<div class="play-button">▶ Click to Play</div>';
    }
}

function showNoStreamMessage() {
    // Show a message when no stream is active
    const overlay = document.querySelector('#videoOverlay');
    if (overlay) {
        overlay.style.display = 'block';
        overlay.innerHTML = '<div class="no-stream-message">📺 No active stream<br><small>Waiting for stream to start...</small></div>';
        
        // Add some basic styling
        const style = overlay.style;
        style.display = 'flex';
        style.alignItems = 'center';
        style.justifyContent = 'center';
        style.backgroundColor = 'rgba(0, 0, 0, 0.8)';
        style.color = 'white';
        style.fontSize = '18px';
        style.textAlign = 'center';
    }
}

function addQualityControls(hls) {
    // Add quality selection controls
    const videoWrapper = document.querySelector('#videoWrapper');
    if (videoWrapper && hls.levels.length > 1) {
        const qualitySelect = document.createElement('select');
        qualitySelect.id = 'qualitySelect';
        qualitySelect.style.position = 'absolute';
        qualitySelect.style.top = '10px';
        qualitySelect.style.right = '10px';
        qualitySelect.style.zIndex = '1000';
        
        // Add auto option
        const autoOption = document.createElement('option');
        autoOption.value = '-1';
        autoOption.textContent = 'Auto';
        qualitySelect.appendChild(autoOption);
        
        // Add quality options
        hls.levels.forEach((level, index) => {
            const option = document.createElement('option');
            option.value = index;
            option.textContent = `${level.height}p (${Math.round(level.bitrate/1000)}k)`;
            qualitySelect.appendChild(option);
        });
        
        qualitySelect.addEventListener('change', (e) => {
            const selectedLevel = parseInt(e.target.value);
            hls.currentLevel = selectedLevel;
            debugLog('Manual quality change to:', selectedLevel === -1 ? 'auto' : selectedLevel);
        });
        
        videoWrapper.appendChild(qualitySelect);
    }
}

function addBufferMonitoring(hls, videoElement) {
    // Add buffer level monitoring
    setInterval(() => {
        if (videoElement && window.hlsPlayer) {
            const bufferInfo = hls.bufferLength;
            if (bufferInfo < 5) {
                console.warn('Low buffer warning:', bufferInfo);
            }
        }
    }, 5000);
}

function initMPEGTSPlayer() {
    if (!mpegts.isSupported()) {
        console.warn('mpegts not supported'); // Keep compatibility warnings visible
        return;
    }
    
    debugLog('Initializing MPEG-TS player');

    let videoElement = document.querySelector('#videoElement');
    let flvPlayer = mpegts.createPlayer({
        type: 'flv',
        url: '/live'
    }, {
        isLive: true,
        liveBufferLatencyChasing: true,
        autoCleanupSourceBuffer: true,
    });
    
    flvPlayer.attachMediaElement(videoElement);
    flvPlayer.load();
    flvPlayer.play();
    
    // Store player instance for cleanup
    window.flvPlayer = flvPlayer;
    
    setupVideoOverlay();
}

function setupVideoOverlay() {
    let overlay = document.querySelector('#videoOverlay');
    if (overlay) {
        overlay.onclick = () => {
            overlay.style.display = 'none';
            let videoElement = document.querySelector('#videoElement');
            if (videoElement) {
                videoElement.muted = false;
                videoElement.play().catch(e => {
                    debugWarn('Manual play failed:', e);
                });
            }
        };
    }
}

// Cleanup function for page unload
function cleanup() {
    if (window.hlsPlayer) {
        debugLog('Cleaning up HLS player');
        try {
            window.hlsPlayer.destroy();
        } catch (e) {
            debugWarn('Error destroying HLS player:', e);
        }
        window.hlsPlayer = null;
    }
    if (window.flvPlayer) {
        debugLog('Cleaning up FLV player');
        try {
            window.flvPlayer.destroy();
        } catch (e) {
            debugWarn('Error destroying FLV player:', e);
        }
        window.flvPlayer = null;
    }
}

// Recovery function for severe HLS issues
function recoverHLSPlayer() {
    debugLog('Attempting HLS player recovery');
    const videoElement = document.querySelector('#videoElement');
    if (!videoElement) return;
    
    // Get current source URL
    const currentSource = window.hlsPlayer?.url || '/live?format=hls';
    
    // Cleanup existing player
    cleanup();
    
    // Wait a moment then reinitialize
    setTimeout(() => {
        debugLog('Reinitializing HLS player after recovery');
        initHLSWithLibrary(videoElement, currentSource);
    }, 1000);
}

// Enhanced statistics collection
function collectPlaybackStats() {
    if (window.hlsPlayer) {
        const stats = {
            currentLevel: window.hlsPlayer.currentLevel,
            autoLevelEnabled: window.hlsPlayer.autoLevelEnabled,
            levels: window.hlsPlayer.levels?.length || 0,
            loadLevel: window.hlsPlayer.loadLevel,
            nextLevel: window.hlsPlayer.nextLevel,
            bufferLength: window.hlsPlayer.bufferLength
        };
        debugLog('HLS Stats:', stats);
        return stats;
    }
    return null;
}

// Expose stats collection for debugging
window.getHLSStats = collectPlaybackStats;

window.addEventListener('load', initPlayer);
window.addEventListener('beforeunload', cleanup);

// Add periodic stats logging in debug mode
if (window.location.search.includes('debug=1')) {
    setInterval(() => {
        collectPlaybackStats();
    }, 10000);
}
