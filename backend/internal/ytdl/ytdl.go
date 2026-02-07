package ytdl

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// DownloadStatus represents the state of a download
type DownloadStatus string

const (
	StatusPending     DownloadStatus = "pending"
	StatusDownloading DownloadStatus = "downloading"
	StatusReady       DownloadStatus = "ready"
	StatusFailed      DownloadStatus = "failed"
)

// DownloadState tracks the progress of a single video download
type DownloadState struct {
	VideoID  string         `json:"video_id"`
	Status   DownloadStatus `json:"status"`
	Progress float64        `json:"progress"` // 0-100
	FilePath string         `json:"file_path,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// ProgressCallback is called when download progress changes
type ProgressCallback func(videoID string, state *DownloadState)

// Downloader manages YouTube video downloads via yt-dlp
type Downloader struct {
	cacheDir   string
	states     map[string]*DownloadState
	mu         sync.RWMutex
	onProgress ProgressCallback
}

var progressRegex = regexp.MustCompile(`\[download\]\s+([\d.]+)%`)

// CheckInstalled checks if yt-dlp is available and returns its version
func CheckInstalled() (string, error) {
	out, err := exec.Command("yt-dlp", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp not found: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// NewDownloader creates a new Downloader with the given cache directory
func NewDownloader(cacheDir string) (*Downloader, error) {
	// Check that yt-dlp is installed
	if _, err := CheckInstalled(); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	return &Downloader{
		cacheDir: cacheDir,
		states:   make(map[string]*DownloadState),
	}, nil
}

// OnProgress sets the callback for download progress updates
func (d *Downloader) OnProgress(cb ProgressCallback) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onProgress = cb
}

// GetState returns the current download state for a video
func (d *Downloader) GetState(videoID string) *DownloadState {
	d.mu.RLock()
	state, ok := d.states[videoID]
	d.mu.RUnlock()

	if ok {
		return state
	}

	// Check filesystem for cached file even without in-memory state
	if fp := d.findCachedFile(videoID); fp != "" {
		state = &DownloadState{
			VideoID:  videoID,
			Status:   StatusReady,
			Progress: 100,
			FilePath: fp,
		}
		d.mu.Lock()
		d.states[videoID] = state
		d.mu.Unlock()
		return state
	}

	return nil
}

// GetFilePath returns the file path if the video is downloaded, empty string otherwise
func (d *Downloader) GetFilePath(videoID string) string {
	state := d.GetState(videoID)
	if state != nil && state.Status == StatusReady {
		return state.FilePath
	}
	return ""
}

// StartDownload begins an async download of the given video ID.
// If already downloading or cached, it returns immediately.
func (d *Downloader) StartDownload(videoID string) {
	d.mu.Lock()

	// Already tracked?
	if state, ok := d.states[videoID]; ok {
		if state.Status == StatusReady || state.Status == StatusDownloading || state.Status == StatusPending {
			d.mu.Unlock()
			return
		}
		// If previously failed, allow retry
	}

	// Check cache first
	if fp := d.findCachedFile(videoID); fp != "" {
		d.states[videoID] = &DownloadState{
			VideoID:  videoID,
			Status:   StatusReady,
			Progress: 100,
			FilePath: fp,
		}
		cb := d.onProgress
		state := d.states[videoID]
		d.mu.Unlock()
		if cb != nil {
			cb(videoID, state)
		}
		return
	}

	// Create pending state
	d.states[videoID] = &DownloadState{
		VideoID:  videoID,
		Status:   StatusPending,
		Progress: 0,
	}
	d.mu.Unlock()

	go d.doDownload(videoID)
}

// doDownload runs yt-dlp and parses progress from stdout
func (d *Downloader) doDownload(videoID string) {
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	outputTemplate := filepath.Join(d.cacheDir, videoID+".%(ext)s")

	cmd := exec.Command("yt-dlp",
		"-f", "bestvideo[height<=1080]+bestaudio/best[height<=1080]",
		"--merge-output-format", "mp4",
		"--newline",
		"--no-playlist",
		"-o", outputTemplate,
		videoURL,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		d.setFailed(videoID, fmt.Sprintf("failed to create stdout pipe: %v", err))
		return
	}

	cmd.Stderr = cmd.Stdout // Merge stderr into stdout

	d.setState(videoID, StatusDownloading, 0, "")

	if err := cmd.Start(); err != nil {
		d.setFailed(videoID, fmt.Sprintf("failed to start yt-dlp: %v", err))
		return
	}

	log.Printf("[ytdl] Download started for %s", videoID)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := progressRegex.FindStringSubmatch(line); len(matches) == 2 {
			var pct float64
			fmt.Sscanf(matches[1], "%f", &pct)
			d.setState(videoID, StatusDownloading, pct, "")
		}
	}

	if err := cmd.Wait(); err != nil {
		d.setFailed(videoID, fmt.Sprintf("yt-dlp exited with error: %v", err))
		return
	}

	// Find the downloaded file
	fp := d.findCachedFile(videoID)
	if fp == "" {
		d.setFailed(videoID, "download completed but file not found")
		return
	}

	d.mu.Lock()
	state := d.states[videoID]
	state.Status = StatusReady
	state.Progress = 100
	state.FilePath = fp
	cb := d.onProgress
	d.mu.Unlock()

	log.Printf("[ytdl] Download complete for %s -> %s", videoID, fp)
	if cb != nil {
		cb(videoID, state)
	}
}

// setState updates download state and fires progress callback
func (d *Downloader) setState(videoID string, status DownloadStatus, progress float64, errMsg string) {
	d.mu.Lock()
	state, ok := d.states[videoID]
	if !ok {
		state = &DownloadState{VideoID: videoID}
		d.states[videoID] = state
	}
	state.Status = status
	state.Progress = progress
	if errMsg != "" {
		state.Error = errMsg
	}
	cb := d.onProgress
	d.mu.Unlock()

	if cb != nil {
		cb(videoID, state)
	}
}

// setFailed marks a download as failed
func (d *Downloader) setFailed(videoID string, errMsg string) {
	log.Printf("[ytdl] Download failed for %s: %s", videoID, errMsg)
	d.setState(videoID, StatusFailed, 0, errMsg)
}

// findCachedFile looks for a downloaded file matching the video ID
func (d *Downloader) findCachedFile(videoID string) string {
	pattern := filepath.Join(d.cacheDir, videoID+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}
	// Prefer .mp4, fall back to first match
	for _, m := range matches {
		if strings.HasSuffix(m, ".mp4") {
			return m
		}
	}
	// Skip partial downloads
	for _, m := range matches {
		if strings.HasSuffix(m, ".part") || strings.HasSuffix(m, ".ytdl") {
			continue
		}
		return m
	}
	return ""
}
