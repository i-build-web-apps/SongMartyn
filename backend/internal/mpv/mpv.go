package mpv

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dexterlb/mpvipc"
	"songmartyn/pkg/models"
)

// DisplaySettings configures which display to use for the player
type DisplaySettings struct {
	TargetDisplay  string // Name of display to use (empty = auto/primary)
	ScreenIndex    int    // Screen index for mpv (0-based, -1 = auto)
	AutoFullscreen bool   // Automatically fullscreen on startup
}

// Controller manages the mpv media player instance
type Controller struct {
	conn       *mpvipc.Connection
	cmd        *exec.Cmd
	socketPath string
	pidFile    string
	executable string
	mu         sync.RWMutex
	adopted    bool // true if we adopted an existing MPV instance

	// Display settings
	displaySettings DisplaySettings

	// Track content type for end-file handling
	playingSong       bool    // true when playing a song (vs holding screen/image)
	currentPlaylistID int64   // playlist_entry_id of current content
	songDuration      float64 // duration of current song in seconds
	lastPosition      float64 // last known position for end detection
	stopMonitor       chan struct{} // channel to stop playback monitor

	// Callbacks
	onStateChange func(state models.PlayerState)
	onTrackEnd    func()
}

// NewController creates a new mpv controller
// executable is the path to the mpv binary (default: "mpv")
func NewController(executable string) *Controller {
	socketPath := getSocketPath()
	pidFile := getPidFilePath()
	if executable == "" {
		executable = "mpv"
	}
	return &Controller{
		socketPath: socketPath,
		pidFile:    pidFile,
		executable: executable,
		displaySettings: DisplaySettings{
			ScreenIndex:    -1,   // Auto
			AutoFullscreen: true, // Default to fullscreen
		},
	}
}

// SetDisplaySettings configures which display to use for the player
func (c *Controller) SetDisplaySettings(settings DisplaySettings) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.displaySettings = settings
	log.Printf("[MPV] Display settings updated: screen=%d, fullscreen=%v, target=%s",
		settings.ScreenIndex, settings.AutoFullscreen, settings.TargetDisplay)
}

// GetDisplaySettings returns the current display settings
func (c *Controller) GetDisplaySettings() DisplaySettings {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.displaySettings
}

// getPidFilePath returns the path to store the MPV process ID
func getPidFilePath() string {
	return filepath.Join(os.TempDir(), "songmartyn-mpv.pid")
}

// getSocketPath returns the appropriate IPC socket path for the OS
func getSocketPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\songmartyn-mpv`
	}
	return filepath.Join(os.TempDir(), "songmartyn-mpv.sock")
}

// tryReconnect attempts to connect to an existing MPV instance
// Returns true if successfully connected and the instance is healthy
func (c *Controller) tryReconnect() bool {
	// Check if socket exists (Unix) or try to connect (Windows named pipe)
	if runtime.GOOS != "windows" {
		if _, err := os.Stat(c.socketPath); os.IsNotExist(err) {
			return false
		}
	}

	log.Printf("[MPV] Attempting to reconnect to existing instance at %s", c.socketPath)

	conn := mpvipc.NewConnection(c.socketPath)
	if err := conn.Open(); err != nil {
		log.Printf("[MPV] Failed to connect to existing socket: %v", err)
		return false
	}

	// Test the connection by getting a property
	_, err := conn.Get("mpv-version")
	if err != nil {
		log.Printf("[MPV] Existing connection unhealthy: %v", err)
		conn.Close()
		return false
	}

	log.Printf("[MPV] Successfully reconnected to existing MPV instance")
	c.conn = conn
	c.adopted = true

	// Start event listener for adopted instance
	go c.listenEvents()

	return true
}

// savePid saves the MPV process ID to a file
func (c *Controller) savePid(pid int) {
	os.WriteFile(c.pidFile, []byte(strconv.Itoa(pid)), 0644)
}

// readPid reads the saved MPV process ID
func (c *Controller) readPid() (int, error) {
	data, err := os.ReadFile(c.pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// cleanupOrphans finds and kills orphaned MPV processes that belong to us
func (c *Controller) cleanupOrphans() {
	log.Printf("[MPV] Cleaning up orphaned processes...")

	// First, try to gracefully quit via socket if it exists
	if c.tryGracefulQuit() {
		time.Sleep(500 * time.Millisecond)
	}

	// Then kill by saved PID
	if pid, err := c.readPid(); err == nil && pid > 0 {
		c.killProcessByPid(pid)
	}

	// Platform-specific cleanup for any remaining orphans using our socket
	c.killOrphansBySocket()

	// Clean up files
	os.Remove(c.socketPath)
	os.Remove(c.pidFile)

	time.Sleep(200 * time.Millisecond)
}

// tryGracefulQuit attempts to send a quit command via the socket
func (c *Controller) tryGracefulQuit() bool {
	// Check if socket exists
	if runtime.GOOS != "windows" {
		if _, err := os.Stat(c.socketPath); os.IsNotExist(err) {
			return false
		}
	}

	conn := mpvipc.NewConnection(c.socketPath)
	if err := conn.Open(); err != nil {
		return false
	}
	defer conn.Close()

	log.Printf("[MPV] Sending graceful quit command to existing instance")
	conn.Call("quit")
	return true
}

// killProcessByPid kills a process by its PID
func (c *Controller) killProcessByPid(pid int) {
	log.Printf("[MPV] Attempting to kill process with PID %d", pid)

	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}

	// Check if process is still running
	if runtime.GOOS == "windows" {
		// On Windows, FindProcess always succeeds, so we try to kill directly
		proc.Kill()
	} else {
		// On Unix, send signal 0 to check if process exists
		if err := proc.Signal(os.Signal(nil)); err == nil {
			proc.Kill()
			proc.Wait()
		}
	}
}

// killOrphansBySocket finds and kills MPV processes using our socket (platform-specific)
func (c *Controller) killOrphansBySocket() {
	switch runtime.GOOS {
	case "darwin", "linux":
		c.killOrphansUnix()
	case "windows":
		c.killOrphansWindows()
	}
}

// killOrphansUnix finds MPV processes using our socket on macOS/Linux
func (c *Controller) killOrphansUnix() {
	// Method 1: Use lsof to find process using our socket
	if pids := c.findPidsByLsof(); len(pids) > 0 {
		for _, pid := range pids {
			log.Printf("[MPV] Killing orphaned process %d (found via lsof)", pid)
			c.killProcessByPid(pid)
		}
		return
	}

	// Method 2: Use pgrep to find mpv processes with our socket in args
	if pids := c.findPidsByPgrep(); len(pids) > 0 {
		for _, pid := range pids {
			log.Printf("[MPV] Killing orphaned process %d (found via pgrep)", pid)
			c.killProcessByPid(pid)
		}
	}
}

// findPidsByLsof uses lsof to find processes using our socket
func (c *Controller) findPidsByLsof() []int {
	cmd := exec.Command("lsof", "-t", c.socketPath)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var pids []int
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		if pid, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

// findPidsByPgrep uses pgrep to find mpv processes with our socket
func (c *Controller) findPidsByPgrep() []int {
	cmd := exec.Command("pgrep", "-f", c.socketPath)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var pids []int
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		if pid, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

// killOrphansWindows finds and kills MPV processes on Windows
func (c *Controller) killOrphansWindows() {
	// Use tasklist/taskkill with window title or command line matching
	// Find mpv processes and check their command line for our pipe name
	cmd := exec.Command("wmic", "process", "where", "name='mpv.exe'", "get", "processid,commandline", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, c.socketPath) {
			// Extract PID from CSV line (format: Node,CommandLine,ProcessId)
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				if pid, err := strconv.Atoi(strings.TrimSpace(parts[len(parts)-1])); err == nil {
					log.Printf("[MPV] Killing orphaned Windows process %d", pid)
					exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid)).Run()
				}
			}
		}
	}
}

// Start launches the mpv process with IPC enabled
// It first attempts to reconnect to an existing instance, then cleans up orphans if needed
func (c *Controller) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Strategy 1: Try to reconnect to existing healthy instance
	if c.tryReconnect() {
		log.Printf("[MPV] Adopted existing MPV instance")
		return nil
	}

	// Strategy 2: Clean up any orphaned processes
	c.cleanupOrphans()

	// Strategy 3: Start fresh MPV instance
	log.Printf("[MPV] Starting fresh MPV instance")

	args := []string{
		"--idle=yes",
		"--force-window=yes",
		"--keep-open=yes",
		"--input-ipc-server=" + c.socketPath,
		"--hwdec=auto",    // Hardware acceleration
		"--volume=100",
		"--osc=no",        // Disable on-screen controller
		"--osd-level=0",   // Minimal OSD
	}

	// Display/screen selection
	if c.displaySettings.ScreenIndex >= 0 {
		screenArg := fmt.Sprintf("--screen=%d", c.displaySettings.ScreenIndex)
		fsScreenArg := fmt.Sprintf("--fs-screen=%d", c.displaySettings.ScreenIndex)
		args = append(args, screenArg, fsScreenArg)
		log.Printf("[MPV] Using screen index: %d", c.displaySettings.ScreenIndex)
	}

	// Fullscreen setting
	if c.displaySettings.AutoFullscreen {
		args = append(args, "--fullscreen=yes")
		log.Printf("[MPV] Starting in fullscreen mode")
	} else {
		args = append(args, "--fullscreen=no")
	}

	// Platform-specific audio output
	switch runtime.GOOS {
	case "darwin":
		args = append(args, "--ao=coreaudio")
	case "windows":
		args = append(args, "--ao=wasapi")
	default: // Linux and others
		args = append(args, "--ao=pipewire,pulse,alsa")
	}

	c.cmd = exec.Command(c.executable, args...)
	c.adopted = false

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mpv: %w", err)
	}

	// Save PID for future cleanup
	c.savePid(c.cmd.Process.Pid)
	log.Printf("[MPV] Started with PID %d", c.cmd.Process.Pid)

	// Wait for socket to be ready
	for i := 0; i < 50; i++ {
		if runtime.GOOS == "windows" {
			// On Windows, try to connect to named pipe
			conn := mpvipc.NewConnection(c.socketPath)
			if err := conn.Open(); err == nil {
				conn.Close()
				break
			}
		} else {
			if _, err := os.Stat(c.socketPath); err == nil {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Connect to IPC
	conn := mpvipc.NewConnection(c.socketPath)
	if err := conn.Open(); err != nil {
		c.cmd.Process.Kill()
		os.Remove(c.pidFile)
		return fmt.Errorf("failed to connect to mpv IPC: %w", err)
	}
	c.conn = conn

	// Start event listener
	go c.listenEvents()

	return nil
}

// Stop terminates the mpv process
func (c *Controller) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Call("quit")
		c.conn.Close()
		c.conn = nil
	}

	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd = nil
	}

	// Clean up files
	os.Remove(c.socketPath)
	os.Remove(c.pidFile)
	c.adopted = false
	return nil
}

// StopPlayback stops the current playback without terminating MPV
// This stops playback and clears the playlist
func (c *Controller) StopPlayback() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	// Use the stop command which stops playback and clears playlist
	_, err := c.conn.Call("stop")
	return err
}

// IsRunning returns true if mpv is running and connected
func (c *Controller) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// For adopted instances, we may not have cmd but still have conn
	if c.adopted {
		return c.conn != nil
	}
	return c.conn != nil && c.cmd != nil && c.cmd.Process != nil
}

// Restart restarts the mpv process
func (c *Controller) Restart() error {
	// Stop if running
	if c.IsRunning() {
		c.Stop()
		time.Sleep(200 * time.Millisecond) // Give time for cleanup
	}
	return c.Start()
}

// Play starts or resumes playback
func (c *Controller) Play() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}
	return c.conn.Set("pause", false)
}

// Pause pauses playback
func (c *Controller) Pause() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}
	return c.conn.Set("pause", true)
}

// LoadFile loads a media file for playback
func (c *Controller) LoadFile(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	// Reset volume to 100 (may have been set to 0 by BGM fade-out)
	c.conn.Set("volume", 100)

	// Reset loop settings from image display
	c.conn.Set("loop-file", "no")

	_, err := c.conn.Call("loadfile", path)
	return err
}

// LoadImage loads an image file and displays it indefinitely
// Used for holding screens between songs
func (c *Controller) LoadImage(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	// Mark that we're NOT playing a song (showing image/holding screen)
	c.playingSong = false

	// First set the properties for image display
	c.conn.Set("image-display-duration", "inf")
	c.conn.Set("loop-file", "inf")

	// Then load the image
	_, err := c.conn.Call("loadfile", path, "replace")
	return err
}

// LoadBGMWithImage loads BGM audio while keeping a static image displayed
func (c *Controller) LoadBGMWithImage(imagePath, audioURL string, targetVolume float64) error {
	c.mu.Lock()

	if c.conn == nil {
		c.mu.Unlock()
		return fmt.Errorf("mpv not connected")
	}

	log.Printf("[MPV] Loading BGM with image: %s, audio: %s", imagePath, audioURL)

	// Set volume to 0 for fade-in
	c.conn.Set("volume", 0)

	// Set image to display infinitely and loop
	c.conn.Set("image-display-duration", "inf")
	c.conn.Set("loop-file", "inf")

	// Load the image first
	_, err := c.conn.Call("loadfile", imagePath, "replace")
	if err != nil {
		c.mu.Unlock()
		log.Printf("[MPV] Failed to load image: %v", err)
		return err
	}

	c.mu.Unlock()

	// Wait for image to start loading (outside mutex)
	time.Sleep(500 * time.Millisecond)

	// Now add the audio track
	c.mu.Lock()
	if c.conn != nil {
		_, err = c.conn.Call("audio-add", audioURL, "select")
		if err != nil {
			log.Printf("[MPV] audio-add failed: %v - BGM will play without holding screen", err)
			// Fallback: just load the audio directly
			c.conn.Call("loadfile", audioURL, "replace")
		} else {
			log.Printf("[MPV] Successfully added audio track")
		}
	}
	c.mu.Unlock()

	// Fade in the volume over 2 seconds
	go c.fadeVolume(0, targetVolume, 2*time.Second)

	return nil
}

// UpdateBGMImage updates the displayed image while keeping BGM audio playing
func (c *Controller) UpdateBGMImage(imagePath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	log.Printf("[MPV] Updating BGM image to: %s", imagePath)

	// Get current volume to preserve it
	vol, _ := c.conn.Get("volume")
	currentVol, ok := vol.(float64)
	if !ok {
		currentVol = 50
	}

	// Get current audio URL - we need to re-add it after loading new image
	// We'll use the audio-reload command or just load the image as external file

	// Use video-add to replace the video track while keeping audio
	// This loads the image into the video slot without affecting audio
	_, err := c.conn.Call("video-add", imagePath, "select")
	if err != nil {
		log.Printf("[MPV] video-add failed: %v, trying loadfile approach", err)
		// Fallback: The image update will happen when BGM stops
		return err
	}

	// Restore volume in case it was affected
	c.conn.Set("volume", currentVol)

	return nil
}

// StopBGMWithFade stops BGM with a fade out effect
func (c *Controller) StopBGMWithFade(fadeDuration time.Duration) error {
	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return fmt.Errorf("mpv not connected")
	}

	// Get current volume
	vol, err := c.conn.Get("volume")
	c.mu.Unlock()

	if err != nil {
		log.Printf("[MPV] Failed to get volume for fade: %v, using default", err)
		vol = 50.0
	}

	currentVol, ok := vol.(float64)
	if !ok {
		currentVol = 50
	}

	log.Printf("[MPV] Stopping BGM with fade from volume %.0f", currentVol)

	// Fade out (blocking)
	c.fadeVolume(currentVol, 0, fadeDuration)

	// Stop playback after fade completes
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err = c.conn.Call("stop")
	return err
}

// fadeVolume smoothly transitions volume from start to end over duration
func (c *Controller) fadeVolume(start, end float64, duration time.Duration) {
	steps := 20
	stepDuration := duration / time.Duration(steps)
	volumeStep := (end - start) / float64(steps)

	for i := 0; i <= steps; i++ {
		vol := start + volumeStep*float64(i)
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Set("volume", vol)
		}
		c.mu.Unlock()
		if i < steps {
			time.Sleep(stepDuration)
		}
	}
}

// LoadCDG loads a CDG file with its paired audio file
// CDG files contain karaoke graphics that sync with the audio
func (c *Controller) LoadCDG(cdgPath, audioPath string) error {
	c.mu.Lock()

	if c.conn == nil {
		c.mu.Unlock()
		return fmt.Errorf("mpv not connected")
	}

	// Mark that we're playing a song
	c.playingSong = true

	// Reset loop settings from image display
	c.conn.Set("loop-file", "no")

	// Load CDG as video with external audio file
	// Note: audio-files (plural) is the correct mpv option name
	_, err := c.conn.Call("loadfile", cdgPath,
		"replace",
		fmt.Sprintf("audio-files=%s", audioPath),
	)
	c.mu.Unlock()

	// Start playback monitor to detect song end
	if err == nil {
		c.StartPlaybackMonitor()
	}
	return err
}

// SetVolume sets the playback volume (0-100)
func (c *Controller) SetVolume(volume float64) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}
	return c.conn.Set("volume", volume)
}

// Seek seeks to a position in seconds
func (c *Controller) Seek(position float64) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}
	_, err := c.conn.Call("seek", position, "absolute")
	return err
}

// SetPitch sets the pitch shift in semitones (-12 to +12)
// Uses rubberband audio filter for high-quality pitch shifting
func (c *Controller) SetPitch(semitones int) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	if semitones == 0 {
		// Remove pitch filter
		_, err := c.conn.Call("af", "remove", "@pitch")
		return err
	}

	// Convert semitones to pitch scale: scale = 2^(semitones/12)
	// +12 semitones = 2.0 (one octave up)
	// -12 semitones = 0.5 (one octave down)
	// Using precomputed constant: 2^(1/12) = 1.0594630943592953
	pitchScale := 1.0
	semitoneRatio := 1.0594630943592953
	if semitones > 0 {
		for i := 0; i < semitones; i++ {
			pitchScale *= semitoneRatio
		}
	} else {
		for i := 0; i > semitones; i-- {
			pitchScale /= semitoneRatio
		}
	}

	// Apply rubberband filter with pitch scale
	// @pitch is a label so we can update/remove it later
	filter := fmt.Sprintf("@pitch:rubberband=pitch-scale=%.6f", pitchScale)
	_, err := c.conn.Call("af", "add", filter)
	return err
}

// SetTempo sets the playback speed (0.5 to 2.0, 1.0 = normal)
// This affects both audio and video speed
func (c *Controller) SetTempo(speed float64) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	// Clamp speed to valid range
	if speed < 0.5 {
		speed = 0.5
	}
	if speed > 2.0 {
		speed = 2.0
	}

	return c.conn.Set("speed", speed)
}

// ShowOverlay displays text on screen for a specified duration
// Used for singer name announcements at song start
func (c *Controller) ShowOverlay(text string, durationMs int) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	// Use show-text command: show-text <text> [duration_ms] [level]
	// Level 1 = normal OSD message
	_, err := c.conn.Call("show-text", text, durationMs, 1)
	return err
}

// tickerSubPath is the path to the ticker subtitle file
var tickerSubPath = filepath.Join(os.TempDir(), "songmartyn-ticker.ass")

// ShowTicker displays a scrolling ticker with upcoming singer information
// The ticker appears at the bottom of the screen and scrolls horizontally
func (c *Controller) ShowTicker(entries []TickerEntry) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	if len(entries) == 0 {
		return c.HideTicker()
	}

	// Build the ticker text
	var tickerText strings.Builder
	tickerText.WriteString("Up Next: ")
	for i, entry := range entries {
		if i > 0 {
			tickerText.WriteString("  •  ")
		}
		tickerText.WriteString(entry.SingerName)
		tickerText.WriteString(" - ")
		tickerText.WriteString(entry.SongTitle)
	}

	// Create ASS subtitle with scrolling effect
	// Banner effect scrolls text from right to left
	assContent := fmt.Sprintf(`[Script Info]
Title: SongMartyn Ticker
ScriptType: v4.00+
PlayResX: 1920
PlayResY: 1080

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Ticker,Arial,48,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,0,0,0,0,100,100,0,0,3,2,0,2,50,50,40,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Dialogue: 0,0:00:00.00,9:59:59.00,Ticker,,0,0,0,Banner;50;0;50,{\pos(960,1050)}%s
`, tickerText.String())

	// Write the ASS file
	if err := os.WriteFile(tickerSubPath, []byte(assContent), 0644); err != nil {
		return fmt.Errorf("failed to write ticker subtitle: %w", err)
	}

	// Load the subtitle as a secondary track
	_, err := c.conn.Call("sub-add", tickerSubPath, "auto", "ticker", "eng")
	return err
}

// HideTicker removes the scrolling ticker from the display
func (c *Controller) HideTicker() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	// Remove any subtitle tracks titled "ticker"
	// First get the count of sub tracks
	count, err := c.conn.Get("track-list/count")
	if err != nil {
		return nil // No tracks, nothing to remove
	}

	trackCount, ok := count.(float64)
	if !ok {
		return nil
	}

	// Find and remove ticker tracks
	for i := int(trackCount) - 1; i >= 0; i-- {
		titlePath := fmt.Sprintf("track-list/%d/title", i)
		title, err := c.conn.Get(titlePath)
		if err != nil {
			continue
		}
		if titleStr, ok := title.(string); ok && titleStr == "ticker" {
			idPath := fmt.Sprintf("track-list/%d/id", i)
			id, err := c.conn.Get(idPath)
			if err != nil {
				continue
			}
			if idNum, ok := id.(float64); ok {
				c.conn.Call("sub-remove", int(idNum))
			}
		}
	}

	return nil
}

// TickerEntry represents a single entry in the scrolling ticker
type TickerEntry struct {
	SingerName string
	SongTitle  string
}

// GetState returns the current player state
func (c *Controller) GetState() (models.PlayerState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	state := models.PlayerState{}

	if c.conn == nil {
		return state, fmt.Errorf("mpv not connected")
	}

	// Get current position
	if pos, err := c.conn.Get("time-pos"); err == nil && pos != nil {
		if f, ok := pos.(float64); ok {
			state.Position = f
		}
	}

	// Get duration
	if dur, err := c.conn.Get("duration"); err == nil && dur != nil {
		if f, ok := dur.(float64); ok {
			state.Duration = f
		}
	}

	// Get pause state
	if pause, err := c.conn.Get("pause"); err == nil && pause != nil {
		if b, ok := pause.(bool); ok {
			state.IsPlaying = !b
		}
	}

	// Get volume
	if vol, err := c.conn.Get("volume"); err == nil && vol != nil {
		if f, ok := vol.(float64); ok {
			state.Volume = f
		}
	}

	return state, nil
}

// SetVocalMix configures the audio filter for vocal assist mixing
// Uses lavfi-complex to mix instrumental and vocal tracks
func (c *Controller) SetVocalMix(instrumentalPath, vocalPath string, vocalGain float64) error {
	c.mu.Lock()

	if c.conn == nil {
		c.mu.Unlock()
		return fmt.Errorf("mpv not connected")
	}

	// Mark that we're playing a song
	c.playingSong = true

	var err error
	if vocalGain <= 0 {
		// No vocals needed, just load instrumental
		_, err = c.conn.Call("loadfile", instrumentalPath)
	} else {
		// Build lavfi-complex filter for mixing
		// [aid1] = instrumental, [aid2] = vocals
		filter := fmt.Sprintf(
			"[aid1]volume=1.0[instr];[aid2]volume=%.2f[vox];[instr][vox]amix=inputs=2:duration=longest[ao]",
			vocalGain,
		)

		// Load with external audio file and filter
		_, err = c.conn.Call("loadfile", instrumentalPath,
			"replace",
			fmt.Sprintf("audio-files=%s", vocalPath),
			fmt.Sprintf("lavfi-complex=%s", filter),
		)
	}
	c.mu.Unlock()

	// Start playback monitor to detect song end
	if err == nil {
		c.StartPlaybackMonitor()
	}
	return err
}

// SetPlayingSong marks whether we're currently playing a song (vs holding screen or BGM)
// Used to determine if onTrackEnd should fire when playback ends
func (c *Controller) SetPlayingSong(playing bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.playingSong = playing
}

// OnStateChange sets the callback for state changes
func (c *Controller) OnStateChange(fn func(state models.PlayerState)) {
	c.onStateChange = fn
}

// OnTrackEnd sets the callback for when a track finishes
func (c *Controller) OnTrackEnd(fn func()) {
	c.onTrackEnd = fn
}

// StartPlaybackMonitor starts a goroutine to monitor playback position
// and detect when a song ends (position reaches duration)
func (c *Controller) StartPlaybackMonitor() {
	// Stop any existing monitor
	c.stopPlaybackMonitor()

	c.mu.Lock()
	c.stopMonitor = make(chan struct{})
	c.lastPosition = 0
	c.mu.Unlock()

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		// Wait a moment for MPV to load the file and report duration
		time.Sleep(1 * time.Second)

		// Get initial duration
		c.mu.Lock()
		if c.conn != nil {
			if dur, err := c.conn.Get("duration"); err == nil && dur != nil {
				if f, ok := dur.(float64); ok && f > 0 {
					c.songDuration = f
					log.Printf("[MPV Monitor] Song duration: %.1f seconds", f)
				}
			}
		}
		stopChan := c.stopMonitor
		c.mu.Unlock()

		for {
			select {
			case <-stopChan:
				log.Println("[MPV Monitor] Stopped")
				return
			case <-ticker.C:
				c.mu.RLock()
				if c.conn == nil || !c.playingSong {
					c.mu.RUnlock()
					continue
				}

				// Get current position
				var position float64
				var duration float64
				var paused bool

				if pos, err := c.conn.Get("time-pos"); err == nil && pos != nil {
					if f, ok := pos.(float64); ok {
						position = f
					}
				}
				if dur, err := c.conn.Get("duration"); err == nil && dur != nil {
					if f, ok := dur.(float64); ok {
						duration = f
					}
				}
				if p, err := c.conn.Get("pause"); err == nil && p != nil {
					if b, ok := p.(bool); ok {
						paused = b
					}
				}

				lastPos := c.lastPosition
				c.mu.RUnlock()

				// Update duration if we got a valid one
				if duration > 0 {
					c.mu.Lock()
					c.songDuration = duration
					c.mu.Unlock()
				}

				// Log position every 10 seconds for debugging
				if int(position)%10 == 0 && position > 0 && int(position) != int(lastPos) {
					log.Printf("[MPV Monitor] Position: %.1f / %.1f (paused: %v)", position, duration, paused)
				}

				// Detect song end: position near duration OR position reset (looped)
				// Check BEFORE pause skip - song might be paused at end due to --keep-open
				if duration > 0 && position > 0 {
					// Song ended if we're within 1 second of the end (or paused at end)
					if position >= duration-1.0 {
						log.Printf("[MPV Monitor] Song ended (position: %.1f, duration: %.1f)", position, duration)
						c.mu.Lock()
						c.playingSong = false
						c.mu.Unlock()
						c.stopPlaybackMonitor()
						if c.onTrackEnd != nil {
							c.onTrackEnd()
						}
						return
					}

					// Detect if position jumped backwards (indicating loop restart)
					if lastPos > 5 && position < 2 && lastPos > position+5 {
						log.Printf("[MPV Monitor] Song looped (last: %.1f, now: %.1f) - treating as end", lastPos, position)
						c.mu.Lock()
						c.playingSong = false
						c.mu.Unlock()
						c.stopPlaybackMonitor()
						if c.onTrackEnd != nil {
							c.onTrackEnd()
						}
						return
					}
				}

				c.mu.Lock()
				c.lastPosition = position
				c.mu.Unlock()
			}
		}
	}()
}

// stopPlaybackMonitor stops the playback monitor goroutine
func (c *Controller) stopPlaybackMonitor() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopMonitor != nil {
		close(c.stopMonitor)
		c.stopMonitor = nil
	}
}

// listenEvents listens for mpv events
func (c *Controller) listenEvents() {
	events, stopListening := c.conn.NewEventListener()
	defer close(stopListening)

	for event := range events {
		switch event.Name {
		case "end-file":
			// Check the reason for end-file
			// MPV returns reason as integer: 0=eof, 1=stop, 2=quit, 3=error, 4=redirect, 5=unknown
			reason := "unknown"
			if r, ok := event.ExtraData["reason"].(float64); ok {
				switch int(r) {
				case 0:
					reason = "eof"
				case 1:
					reason = "stop"
				case 2:
					reason = "quit"
				case 3:
					reason = "error"
				case 4:
					reason = "redirect"
				}
			} else if r, ok := event.ExtraData["reason"].(string); ok {
				reason = r
			}

			c.mu.RLock()
			wasPlayingSong := c.playingSong
			c.mu.RUnlock()

			fmt.Printf("[MPV EVENT] end-file (reason: %s, playingSong: %v, data: %+v)\n", reason, wasPlayingSong, event.ExtraData)

			// Trigger onTrackEnd when a song reaches natural end (eof)
			// Must have been playing a song, not just replacing content
			if reason == "eof" && wasPlayingSong && c.onTrackEnd != nil {
				c.mu.Lock()
				c.playingSong = false // Song has ended
				c.mu.Unlock()
				log.Println("[MPV] Song finished naturally - triggering onTrackEnd")
				c.onTrackEnd()
			}
		case "property-change":
			if c.onStateChange != nil {
				if state, err := c.GetState(); err == nil {
					c.onStateChange(state)
				}
			}
		}
	}
}
