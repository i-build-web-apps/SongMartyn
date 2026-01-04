package models

import "time"

// VocalAssistLevel represents the Chortle vocal assist intensity
type VocalAssistLevel string

const (
	VocalOff  VocalAssistLevel = "OFF"  // 0% gain
	VocalLow  VocalAssistLevel = "LOW"  // 15% gain - pitch reference
	VocalMed  VocalAssistLevel = "MED"  // 45% gain - melody support
	VocalHigh VocalAssistLevel = "HIGH" // 80% gain - full vocal lead
)

// VocalGainMap maps assist levels to their gain percentages
var VocalGainMap = map[VocalAssistLevel]float64{
	VocalOff:  0.0,
	VocalLow:  0.15,
	VocalMed:  0.45,
	VocalHigh: 0.80,
}

// Song represents a queued karaoke track
type Song struct {
	ID           string           `json:"id"`
	Title        string           `json:"title"`
	Artist       string           `json:"artist"`
	Duration     int              `json:"duration"` // seconds
	ThumbnailURL string           `json:"thumbnail_url"`
	VideoURL     string           `json:"video_url"`
	VocalPath    string           `json:"vocal_path,omitempty"`    // Path to extracted vocals
	InstrPath    string           `json:"instr_path,omitempty"`    // Path to instrumental
	VocalAssist  VocalAssistLevel `json:"vocal_assist"`
	AddedBy      string           `json:"added_by"` // MartynKey of who added it
	AddedAt      time.Time        `json:"added_at"`
}

// Session represents a connected client session (The Martyn Handshake)
type Session struct {
	MartynKey      string           `json:"martyn_key"` // UUID
	DisplayName    string           `json:"display_name"`
	VocalAssist    VocalAssistLevel `json:"vocal_assist"`
	SearchHistory  []string         `json:"search_history"`
	CurrentSongID  string           `json:"current_song_id,omitempty"`
	ConnectedAt    time.Time        `json:"connected_at"`
	LastSeenAt     time.Time        `json:"last_seen_at"`
	// Admin panel fields
	IPAddress      string           `json:"ip_address"`
	DeviceName     string           `json:"device_name"`     // Auto-detected or custom
	UserAgent      string           `json:"user_agent"`
	IsAdmin        bool             `json:"is_admin"`
	IsOnline       bool             `json:"is_online"`       // Currently connected
}

// PlayerState represents the current playback state
type PlayerState struct {
	CurrentSong   *Song            `json:"current_song"`
	Position      float64          `json:"position"`      // seconds
	Duration      float64          `json:"duration"`      // seconds
	IsPlaying     bool             `json:"is_playing"`
	Volume        float64          `json:"volume"`        // 0-100
	VocalAssist   VocalAssistLevel `json:"vocal_assist"`
	BGMActive     bool             `json:"bgm_active"`    // Dawnsong mode
}

// QueueState represents the song queue
type QueueState struct {
	Songs    []Song `json:"songs"`
	Position int    `json:"position"` // Current position in queue
}

// RoomState represents the entire room state
type RoomState struct {
	Player    PlayerState  `json:"player"`
	Queue     QueueState   `json:"queue"`
	Sessions  []Session    `json:"sessions"` // Connected clients
}

// LibraryLocation represents a folder containing media files
type LibraryLocation struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`      // Friendly name
	SongCount int       `json:"song_count"`
	AddedAt   time.Time `json:"added_at"`
	LastScan  time.Time `json:"last_scan"`
}

// LibrarySong represents a song in the local library
type LibrarySong struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Artist       string    `json:"artist"`
	Album        string    `json:"album,omitempty"`
	Duration     int       `json:"duration"`      // seconds
	FilePath     string    `json:"file_path"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	VocalPath    string    `json:"vocal_path,omitempty"`
	InstrPath    string    `json:"instr_path,omitempty"`
	LibraryID    int64     `json:"library_id"`
	// Stats
	TimesSung    int       `json:"times_sung"`
	LastSungAt   *time.Time `json:"last_sung_at,omitempty"`
	LastSungBy   string    `json:"last_sung_by,omitempty"` // MartynKey
	AddedAt      time.Time `json:"added_at"`
}

// SongHistory tracks when a user sang a song
type SongHistory struct {
	ID        int64     `json:"id"`
	SongID    string    `json:"song_id"`
	MartynKey string    `json:"martyn_key"`
	SungAt    time.Time `json:"sung_at"`
	// Denormalized for easy display
	SongTitle  string `json:"song_title"`
	SongArtist string `json:"song_artist"`
}
