package mpv

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	executable string
	mu         sync.RWMutex

	// Callbacks
	onStateChange func(state models.PlayerState)
	onTrackEnd    func()
}

// NewController creates a new mpv controller
// executable is the path to the mpv binary (default: "mpv")
func NewController(executable string) *Controller {
	socketPath := getSocketPath()
	if executable == "" {
		executable = "mpv"
	}
	return &Controller{
		socketPath: socketPath,
		executable: executable,
	}
}

// getSocketPath returns the appropriate IPC socket path for the OS
func getSocketPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\songmartyn-mpv`
	}
	return filepath.Join(os.TempDir(), "songmartyn-mpv.sock")
}

// Start launches the mpv process with IPC enabled
func (c *Controller) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Kill any existing mpv processes to avoid conflicts
	exec.Command("pkill", "-x", "mpv").Run()
	time.Sleep(200 * time.Millisecond)

	// Remove stale socket
	os.Remove(c.socketPath)

	// Start mpv with required options
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

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mpv: %w", err)
	}

	// Wait for socket to be ready
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(c.socketPath); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Connect to IPC
	conn := mpvipc.NewConnection(c.socketPath)
	if err := conn.Open(); err != nil {
		c.cmd.Process.Kill()
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
	}

	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}

	os.Remove(c.socketPath)
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
