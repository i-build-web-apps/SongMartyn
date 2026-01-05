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

// Controller manages the mpv media player instance
type Controller struct {
	conn       *mpvipc.Connection
	cmd        *exec.Cmd
	socketPath string
	pidFile    string
	executable string
	mu         sync.RWMutex
	adopted    bool // true if we adopted an existing MPV instance

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
	}
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
		"--fullscreen=no",
		"--osc=no",        // Disable on-screen controller
		"--osd-level=0",   // Minimal OSD
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
	_, err := c.conn.Call("loadfile", path)
	return err
}

// LoadImage loads an image file and displays it indefinitely
// Used for holding screens between songs
func (c *Controller) LoadImage(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	// First set the properties for image display
	c.conn.Set("image-display-duration", "inf")
	c.conn.Set("loop-file", "inf")

	// Then load the image
	_, err := c.conn.Call("loadfile", path, "replace")
	return err
}

// LoadCDG loads a CDG file with its paired audio file
// CDG files contain karaoke graphics that sync with the audio
func (c *Controller) LoadCDG(cdgPath, audioPath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	// Reset loop settings from image display
	c.conn.Set("loop-file", "no")

	// Load CDG as video with external audio file
	// Note: audio-files (plural) is the correct mpv option name
	_, err := c.conn.Call("loadfile", cdgPath,
		"replace",
		fmt.Sprintf("audio-files=%s", audioPath),
	)
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
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	if vocalGain <= 0 {
		// No vocals needed, just load instrumental
		return c.LoadFile(instrumentalPath)
	}

	// Build lavfi-complex filter for mixing
	// [aid1] = instrumental, [aid2] = vocals
	filter := fmt.Sprintf(
		"[aid1]volume=1.0[instr];[aid2]volume=%.2f[vox];[instr][vox]amix=inputs=2:duration=longest[ao]",
		vocalGain,
	)

	// Load with external audio file and filter
	_, err := c.conn.Call("loadfile", instrumentalPath,
		"replace",
		fmt.Sprintf("audio-files=%s", vocalPath),
		fmt.Sprintf("lavfi-complex=%s", filter),
	)
	return err
}

// OnStateChange sets the callback for state changes
func (c *Controller) OnStateChange(fn func(state models.PlayerState)) {
	c.onStateChange = fn
}

// OnTrackEnd sets the callback for when a track finishes
func (c *Controller) OnTrackEnd(fn func()) {
	c.onTrackEnd = fn
}

// listenEvents listens for mpv events
func (c *Controller) listenEvents() {
	events, stopListening := c.conn.NewEventListener()
	defer close(stopListening)

	for event := range events {
		switch event.Name {
		case "end-file":
			// Check the reason for end-file
			// Reason can be: eof, stop, quit, error, redirect, unknown
			reason := "unknown"
			if r, ok := event.ExtraData["reason"].(string); ok {
				reason = r
			}
			fmt.Printf("[MPV EVENT] end-file (reason: %s, data: %+v)\n", reason, event.ExtraData)

			// Only trigger onTrackEnd for natural end of file (eof)
			// Skip for errors or other reasons
			if reason == "eof" && c.onTrackEnd != nil {
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
