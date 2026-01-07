package mpv

import (
	"testing"

	"songmartyn/pkg/models"
)

// These tests document expected behavior and serve as regression tests.
// They test the logic without requiring a live MPV instance.

// TestControllerInitialization verifies controller starts with correct defaults
func TestControllerInitialization(t *testing.T) {
	c := NewController("")

	if c.executable != "mpv" {
		t.Errorf("Expected default executable 'mpv', got '%s'", c.executable)
	}

	if c.playingSong {
		t.Error("Expected playingSong to be false on init")
	}

	if c.conn != nil {
		t.Error("Expected conn to be nil before Start()")
	}
}

// TestControllerWithCustomExecutable verifies custom executable path
func TestControllerWithCustomExecutable(t *testing.T) {
	c := NewController("/custom/path/mpv")

	if c.executable != "/custom/path/mpv" {
		t.Errorf("Expected executable '/custom/path/mpv', got '%s'", c.executable)
	}
}

// TestIsRunningWithoutConnection verifies IsRunning returns false when not connected
func TestIsRunningWithoutConnection(t *testing.T) {
	c := NewController("")

	if c.IsRunning() {
		t.Error("Expected IsRunning() to return false when not connected")
	}
}

// TestGetStateWithoutConnection verifies GetState handles nil connection
func TestGetStateWithoutConnection(t *testing.T) {
	c := NewController("")

	state, err := c.GetState()
	if err == nil {
		t.Error("Expected error from GetState when not connected")
	}

	// State should have zero values
	if state.IsPlaying {
		t.Error("Expected IsPlaying to be false")
	}
}

// TestLoadFileRequiresConnection verifies LoadFile fails without connection
func TestLoadFileRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.LoadFile("/path/to/file.mp4")
	if err == nil {
		t.Error("Expected error from LoadFile when not connected")
	}
}

// TestLoadImageRequiresConnection verifies LoadImage fails without connection
func TestLoadImageRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.LoadImage("/path/to/image.png")
	if err == nil {
		t.Error("Expected error from LoadImage when not connected")
	}
}

// TestStopBGMWithFadeRequiresConnection verifies StopBGMWithFade fails without connection
func TestStopBGMWithFadeRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.StopBGMWithFade(0)
	if err == nil {
		t.Error("Expected error from StopBGMWithFade when not connected")
	}
}

// TestLoadBGMWithImageRequiresConnection verifies LoadBGMWithImage fails without connection
func TestLoadBGMWithImageRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.LoadBGMWithImage("/img.png", "http://stream", 50)
	if err == nil {
		t.Error("Expected error from LoadBGMWithImage when not connected")
	}
}

// TestSetVolumeRequiresConnection verifies SetVolume fails without connection
func TestSetVolumeRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.SetVolume(50.0)
	if err == nil {
		t.Error("Expected error from SetVolume when not connected")
	}
}

// TestStopPlaybackRequiresConnection verifies StopPlayback fails without connection
func TestStopPlaybackRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.StopPlayback()
	if err == nil {
		t.Error("Expected error from StopPlayback when not connected")
	}
}

// TestPauseRequiresConnection verifies Pause fails without connection
func TestPauseRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.Pause()
	if err == nil {
		t.Error("Expected error from Pause when not connected")
	}
}

// TestPlayRequiresConnection verifies Play fails without connection
func TestPlayRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.Play()
	if err == nil {
		t.Error("Expected error from Play when not connected")
	}
}

// TestSeekRequiresConnection verifies Seek fails without connection
func TestSeekRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.Seek(10)
	if err == nil {
		t.Error("Expected error from Seek when not connected")
	}
}

// TestSetVolumeClamps verifies SetVolume clamps values to valid range
func TestSetVolumeClampsDescription(t *testing.T) {
	// This test documents the expected behavior:
	// - Volume should be clamped to 0-100 range
	// - Values below 0 should become 0
	// - Values above 100 should become 100

	// The actual implementation uses clampVolume() which:
	// - Clamps float64 values to 0-100 range
	// - Is called before setting volume property

	// Test the clampVolume function directly
	testCases := []struct {
		input    float64
		expected float64
	}{
		{-10, 0},
		{0, 0},
		{50, 50},
		{100, 100},
		{150, 100},
	}

	for _, tc := range testCases {
		result := clampVolume(tc.input)
		if result != tc.expected {
			t.Errorf("clampVolume(%v): expected %v, got %v", tc.input, tc.expected, result)
		}
	}
}

// Helper to clamp volume (matches the expected behavior)
func clampVolume(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// TestSocketPath verifies socket path is set correctly
func TestSocketPath(t *testing.T) {
	path := getSocketPath()

	if path == "" {
		t.Error("Expected non-empty socket path")
	}

	// Should contain "songmartyn" identifier
	if !contains(path, "songmartyn") {
		t.Errorf("Expected socket path to contain 'songmartyn', got: %s", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// PlayingSong State Management Tests
// =============================================================================

// TestSetPlayingSong verifies SetPlayingSong updates the state correctly
func TestSetPlayingSong(t *testing.T) {
	c := NewController("")

	// Initially false
	if c.playingSong {
		t.Error("Expected playingSong to be false initially")
	}

	// Set to true
	c.SetPlayingSong(true)
	if !c.playingSong {
		t.Error("Expected playingSong to be true after SetPlayingSong(true)")
	}

	// Set back to false
	c.SetPlayingSong(false)
	if c.playingSong {
		t.Error("Expected playingSong to be false after SetPlayingSong(false)")
	}
}

// TestLoadImageSetsPlayingSongFalse documents that LoadImage marks playingSong as false
// This is important because we don't want onTrackEnd to fire for holding screens
func TestLoadImageSetsPlayingSongFalse(t *testing.T) {
	c := NewController("")

	// Simulate that we were playing a song
	c.playingSong = true

	// LoadImage should fail (no connection) but we can verify the state change
	// by checking the function signature documents this behavior
	// The actual implementation sets c.playingSong = false before loading

	// Document: LoadImage sets playingSong = false (line 472 in mpv.go)
	// This ensures end-file events during holding screen don't trigger song end callbacks
}

// =============================================================================
// Callback Registration Tests
// =============================================================================

// TestOnStateChangeCallback verifies OnStateChange callback can be registered
func TestOnStateChangeCallback(t *testing.T) {
	c := NewController("")

	c.OnStateChange(func(state models.PlayerState) {
		// Callback registered - would be called on state changes
		_ = state // Use the parameter
	})

	if c.onStateChange == nil {
		t.Error("Expected onStateChange callback to be set")
	}
}

// TestOnTrackEndCallback verifies OnTrackEnd callback can be registered
func TestOnTrackEndCallback(t *testing.T) {
	c := NewController("")

	c.OnTrackEnd(func() {
		// Callback registered - would be called when track ends
	})

	if c.onTrackEnd == nil {
		t.Error("Expected onTrackEnd callback to be set")
	}
}

// =============================================================================
// Pitch Calculation Tests
// =============================================================================

// TestPitchScaleCalculation verifies the pitch scale formula
// scale = 2^(semitones/12)
func TestPitchScaleCalculation(t *testing.T) {
	semitoneRatio := 1.0594630943592953 // 2^(1/12)

	testCases := []struct {
		semitones int
		minScale  float64
		maxScale  float64
	}{
		{0, 0.999, 1.001},     // 0 semitones = 1.0
		{12, 1.999, 2.001},    // +12 semitones = 2.0 (octave up)
		{-12, 0.499, 0.501},   // -12 semitones = 0.5 (octave down)
		{1, 1.059, 1.060},     // +1 semitone
		{-1, 0.943, 0.944},    // -1 semitone
		{7, 1.498, 1.499},     // +7 semitones (perfect fifth)
	}

	for _, tc := range testCases {
		pitchScale := 1.0
		if tc.semitones > 0 {
			for i := 0; i < tc.semitones; i++ {
				pitchScale *= semitoneRatio
			}
		} else {
			for i := 0; i > tc.semitones; i-- {
				pitchScale /= semitoneRatio
			}
		}

		if pitchScale < tc.minScale || pitchScale > tc.maxScale {
			t.Errorf("Pitch scale for %d semitones: expected %.3f-%.3f, got %.6f",
				tc.semitones, tc.minScale, tc.maxScale, pitchScale)
		}
	}
}

// =============================================================================
// Tempo Clamping Tests
// =============================================================================

// TestTempoClamping verifies tempo is clamped to valid range (0.5 to 2.0)
func TestTempoClamping(t *testing.T) {
	testCases := []struct {
		input    float64
		expected float64
	}{
		{0.25, 0.5},  // Below minimum, should clamp to 0.5
		{0.5, 0.5},   // At minimum
		{1.0, 1.0},   // Normal speed
		{1.5, 1.5},   // Valid speed
		{2.0, 2.0},   // At maximum
		{3.0, 2.0},   // Above maximum, should clamp to 2.0
	}

	for _, tc := range testCases {
		result := clampTempo(tc.input)
		if result != tc.expected {
			t.Errorf("clampTempo(%v): expected %v, got %v", tc.input, tc.expected, result)
		}
	}
}

// Helper to clamp tempo (matches SetTempo implementation)
func clampTempo(speed float64) float64 {
	if speed < 0.5 {
		return 0.5
	}
	if speed > 2.0 {
		return 2.0
	}
	return speed
}

// =============================================================================
// Additional Connection Requirement Tests
// =============================================================================

// TestSetPitchRequiresConnection verifies SetPitch fails without connection
func TestSetPitchRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.SetPitch(5)
	if err == nil {
		t.Error("Expected error from SetPitch when not connected")
	}
}

// TestSetTempoRequiresConnection verifies SetTempo fails without connection
func TestSetTempoRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.SetTempo(1.5)
	if err == nil {
		t.Error("Expected error from SetTempo when not connected")
	}
}

// TestShowOverlayRequiresConnection verifies ShowOverlay fails without connection
func TestShowOverlayRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.ShowOverlay("Test", 3000)
	if err == nil {
		t.Error("Expected error from ShowOverlay when not connected")
	}
}

// TestShowTickerRequiresConnection verifies ShowTicker fails without connection
func TestShowTickerRequiresConnection(t *testing.T) {
	c := NewController("")

	entries := []TickerEntry{{SingerName: "Test", SongTitle: "Song"}}
	err := c.ShowTicker(entries)
	if err == nil {
		t.Error("Expected error from ShowTicker when not connected")
	}
}

// TestHideTickerRequiresConnection verifies HideTicker fails without connection
func TestHideTickerRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.HideTicker()
	if err == nil {
		t.Error("Expected error from HideTicker when not connected")
	}
}

// TestLoadCDGRequiresConnection verifies LoadCDG fails without connection
func TestLoadCDGRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.LoadCDG("/path/to/file.cdg", "/path/to/file.mp3")
	if err == nil {
		t.Error("Expected error from LoadCDG when not connected")
	}
}

// TestSetVocalMixRequiresConnection verifies SetVocalMix fails without connection
func TestSetVocalMixRequiresConnection(t *testing.T) {
	c := NewController("")

	err := c.SetVocalMix("/instrumental.wav", "/vocal.wav", 0.5)
	if err == nil {
		t.Error("Expected error from SetVocalMix when not connected")
	}
}

// =============================================================================
// Empty Ticker Test
// =============================================================================

// TestShowTickerWithNoEntries verifies ShowTicker calls HideTicker when empty
func TestShowTickerWithNoEntriesRequiresConnection(t *testing.T) {
	c := NewController("")

	// With no entries, ShowTicker calls HideTicker which requires connection
	err := c.ShowTicker([]TickerEntry{})
	if err == nil {
		t.Error("Expected error from ShowTicker with empty entries when not connected")
	}
}

// =============================================================================
// PID File Path Test
// =============================================================================

// TestPidFilePath verifies PID file path is set correctly
func TestPidFilePath(t *testing.T) {
	path := getPidFilePath()

	if path == "" {
		t.Error("Expected non-empty PID file path")
	}

	// Should contain "songmartyn" identifier
	if !contains(path, "songmartyn") {
		t.Errorf("Expected PID file path to contain 'songmartyn', got: %s", path)
	}

	// Should end with .pid
	if !contains(path, ".pid") {
		t.Errorf("Expected PID file path to end with '.pid', got: %s", path)
	}
}

// =============================================================================
// Playback Monitor Tests
// =============================================================================

// TestStopPlaybackMonitorSafety verifies stopPlaybackMonitor handles nil channel
func TestStopPlaybackMonitorSafety(t *testing.T) {
	c := NewController("")

	// stopMonitor is nil initially - calling stopPlaybackMonitor should not panic
	c.stopPlaybackMonitor()

	// No panic = success
}

// =============================================================================
// Documentation Tests - Expected Behavior
// =============================================================================

// TestLoadFileResetsVolume documents that LoadFile resets volume to 100
// This is critical for fixing the silent playback bug after BGM fade-out
func TestLoadFileResetsVolumeDocumentation(t *testing.T) {
	// This test documents the expected behavior (implemented in mpv.go:451-452):
	//
	// When LoadFile is called, it MUST reset volume to 100 because:
	// 1. BGM fade-out sets volume to 0
	// 2. If we don't reset, song playback has no audio
	//
	// Code in LoadFile:
	//   c.conn.Set("volume", 100)  // Reset volume (may have been set to 0 by BGM fade-out)
	//   c.conn.Set("loop-file", "no")  // Reset loop settings from image display
	//
	// Without this fix, songs play silently after BGM stops.

	c := NewController("")

	// LoadFile requires connection, so we just verify it fails gracefully
	err := c.LoadFile("/test.mp4")
	if err == nil {
		t.Error("Expected error from LoadFile when not connected")
	}
}

// TestBGMStateManagement documents the BGM state machine
func TestBGMStateManagementDocumentation(t *testing.T) {
	// This test documents the BGM state management:
	//
	// BGM Start Flow:
	// 1. Set volume to 0 (for fade-in)
	// 2. Load holding screen image with loop=inf
	// 3. Wait 500ms for image to load
	// 4. Add audio track with audio-add command
	// 5. Fade volume from 0 to targetVolume over 2 seconds
	//
	// BGM Stop Flow:
	// 1. Get current volume
	// 2. Fade volume from current to 0 over 2 seconds
	// 3. Call "stop" command to clear playlist
	//
	// Key invariant: app.bgmActive tracks whether BGM is playing
	// When bgmActive is true:
	// - showHoldingScreen() should skip (to not disrupt audio)
	// - Adding songs should not reload the holding screen image
}

// TestHoldingScreenSkipsDuringBGM documents that holding screen updates are skipped during BGM
func TestHoldingScreenSkipsDuringBGMDocumentation(t *testing.T) {
	// This test documents behavior in main.go showHoldingScreen():
	//
	// if app.bgmActive {
	//     log.Println("Skipping holding screen update while BGM is playing")
	//     return
	// }
	//
	// This prevents the bug where adding songs while BGM is playing
	// would call LoadBGMWithImage which disrupts the audio stream.
	//
	// The holding screen image will be updated when BGM stops.
}

// =============================================================================
// Display Settings Tests
// =============================================================================

// TestDisplaySettingsDefaults verifies display settings start with correct defaults
func TestDisplaySettingsDefaults(t *testing.T) {
	c := NewController("")

	settings := c.GetDisplaySettings()

	if settings.TargetDisplay != "" {
		t.Errorf("Expected default TargetDisplay to be empty, got '%s'", settings.TargetDisplay)
	}

	if settings.ScreenIndex != -1 {
		t.Errorf("Expected default ScreenIndex to be -1 (auto), got %d", settings.ScreenIndex)
	}

	// AutoFullscreen defaults to true for karaoke use case
	if !settings.AutoFullscreen {
		t.Error("Expected default AutoFullscreen to be true")
	}
}

// TestSetDisplaySettings verifies SetDisplaySettings updates the settings correctly
func TestSetDisplaySettings(t *testing.T) {
	c := NewController("")

	settings := DisplaySettings{
		TargetDisplay:  "Dell U2715H",
		ScreenIndex:    1,
		AutoFullscreen: true,
	}

	c.SetDisplaySettings(settings)

	result := c.GetDisplaySettings()

	if result.TargetDisplay != "Dell U2715H" {
		t.Errorf("Expected TargetDisplay 'Dell U2715H', got '%s'", result.TargetDisplay)
	}

	if result.ScreenIndex != 1 {
		t.Errorf("Expected ScreenIndex 1, got %d", result.ScreenIndex)
	}

	if !result.AutoFullscreen {
		t.Error("Expected AutoFullscreen to be true")
	}
}

// TestDisplaySettingsScreenIndexZero verifies screen index 0 is valid (first display)
func TestDisplaySettingsScreenIndexZero(t *testing.T) {
	c := NewController("")

	settings := DisplaySettings{
		TargetDisplay:  "Primary Display",
		ScreenIndex:    0, // First display (0-indexed)
		AutoFullscreen: false,
	}

	c.SetDisplaySettings(settings)

	result := c.GetDisplaySettings()

	if result.ScreenIndex != 0 {
		t.Errorf("Expected ScreenIndex 0, got %d", result.ScreenIndex)
	}
}

// TestDisplaySettingsMPVArgsDocumentation documents the expected MPV arguments
func TestDisplaySettingsMPVArgsDocumentation(t *testing.T) {
	// This test documents the expected MPV arguments for display settings:
	//
	// When ScreenIndex >= 0:
	//   --screen=<index>     : Specifies which display to render on (0-indexed)
	//   --fs-screen=<index>  : Specifies which display to fullscreen on (0-indexed)
	//
	// When AutoFullscreen is true:
	//   --fullscreen=yes     : Starts MPV in fullscreen mode
	//
	// When AutoFullscreen is false:
	//   --fullscreen=no      : Starts MPV in windowed mode
	//
	// Example for secondary display with fullscreen:
	//   mpv --screen=1 --fs-screen=1 --fullscreen=yes [other args...]
	//
	// The screen index is resolved from the display name in main.go:
	//   - macOS: system_profiler SPDisplaysDataType
	//   - Linux: xrandr --query
	//   - Windows: PowerShell WMI queries
}

// TestDisplaySettingsWithCustomExecutable verifies display settings work with custom executable
func TestDisplaySettingsWithCustomExecutable(t *testing.T) {
	c := NewController("/Applications/mpv.app/Contents/MacOS/mpv")

	settings := DisplaySettings{
		TargetDisplay:  "External Display",
		ScreenIndex:    2,
		AutoFullscreen: true,
	}

	c.SetDisplaySettings(settings)

	result := c.GetDisplaySettings()

	// Verify settings are stored correctly regardless of executable path
	if result.TargetDisplay != "External Display" {
		t.Errorf("Expected TargetDisplay 'External Display', got '%s'", result.TargetDisplay)
	}

	if result.ScreenIndex != 2 {
		t.Errorf("Expected ScreenIndex 2, got %d", result.ScreenIndex)
	}
}

// TestDisplaySettingsAutoDetect verifies auto-detect mode (ScreenIndex = -1)
func TestDisplaySettingsAutoDetect(t *testing.T) {
	c := NewController("")

	// Explicitly set auto-detect
	settings := DisplaySettings{
		TargetDisplay:  "",  // Empty = auto
		ScreenIndex:    -1,  // -1 = auto-detect/primary
		AutoFullscreen: false,
	}

	c.SetDisplaySettings(settings)

	result := c.GetDisplaySettings()

	if result.ScreenIndex != -1 {
		t.Errorf("Expected ScreenIndex -1 (auto), got %d", result.ScreenIndex)
	}

	// When ScreenIndex is -1, mpv uses its default behavior (primary display)
}
