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
	CDGPath      string           `json:"cdg_path,omitempty"`      // Path to CDG graphics file
	AudioPath    string           `json:"audio_path,omitempty"`    // Path to audio file (for CDG)
	VocalAssist  VocalAssistLevel `json:"vocal_assist"`
	KeyChange    int              `json:"key_change"`              // Semitones (-12 to +12)
	TempoChange  float64          `json:"tempo_change"`            // Speed multiplier (0.5 to 2.0, 1.0 = normal)
	AddedBy      string           `json:"added_by"`                // MartynKey of who added it
	AddedAt      time.Time        `json:"added_at"`
}

// AvatarColors represents custom color overrides for avatar parts
type AvatarColors struct {
	Env   string `json:"env,omitempty"`   // Background color
	Clo   string `json:"clo,omitempty"`   // Clothes primary color
	Head  string `json:"head,omitempty"`  // Skin color
	Mouth string `json:"mouth,omitempty"` // Mouth color
	Eyes  string `json:"eyes,omitempty"`  // Eyes primary color
	Top   string `json:"top,omitempty"`   // Hair/top color
}

// AvatarConfig represents the avatar customization settings
type AvatarConfig struct {
	Env    int           `json:"env"`
	Clo    int           `json:"clo"`
	Head   int           `json:"head"`
	Mouth  int           `json:"mouth"`
	Eyes   int           `json:"eyes"`
	Top    int           `json:"top"`
	Colors *AvatarColors `json:"colors,omitempty"` // Optional custom colors
}

// Session represents a connected client session (The Martyn Handshake)
type Session struct {
	MartynKey      string           `json:"martyn_key"` // UUID
	DisplayName    string           `json:"display_name"`
	AvatarID       string           `json:"avatar_id,omitempty"`     // Legacy pixel avatar identifier
	AvatarConfig   *AvatarConfig    `json:"avatar_config,omitempty"` // Multiavatar configuration
	VocalAssist    VocalAssistLevel `json:"vocal_assist"`
	SearchHistory  []string         `json:"search_history"`
	Favorites      []string         `json:"favorites"`               // Favorite song IDs
	CurrentSongID  string           `json:"current_song_id,omitempty"`
	ConnectedAt    time.Time        `json:"connected_at"`
	LastSeenAt     time.Time        `json:"last_seen_at"`
	// Admin panel fields
	IPAddress      string           `json:"ip_address"`
	DeviceName     string           `json:"device_name"`     // Auto-detected or custom
	UserAgent      string           `json:"user_agent"`
	IsAdmin        bool             `json:"is_admin"`
	IsOnline       bool             `json:"is_online"`       // Currently connected
	IsAFK          bool             `json:"is_afk"`          // Away from keyboard
	NameLocked     bool             `json:"name_locked"`     // Admin locked the display name
}

// PlayerState represents the current playback state
type PlayerState struct {
	CurrentSong   *Song            `json:"current_song"`
	Position      float64          `json:"position"`      // seconds
	Duration      float64          `json:"duration"`      // seconds
	IsPlaying     bool             `json:"is_playing"`
	Volume        float64          `json:"volume"`        // 0-100
	VocalAssist   VocalAssistLevel `json:"vocal_assist"`
	BGMActive     bool             `json:"bgm_active"`    // Background music playing
	BGMEnabled    bool             `json:"bgm_enabled"`   // BGM feature enabled
	Idle          bool             `json:"idle"`          // Showing holding screen (not playing a song)
}

// BGMSourceType represents the type of background music source
type BGMSourceType string

const (
	BGMSourceYouTube BGMSourceType = "youtube"
	BGMSourceIcecast BGMSourceType = "icecast"
)

// BGMSettings holds background music configuration
type BGMSettings struct {
	Enabled    bool          `json:"enabled"`
	SourceType BGMSourceType `json:"source_type"` // "youtube" or "icecast"
	URL        string        `json:"url"`         // YouTube URL or Icecast stream URL
	Volume     float64       `json:"volume"`      // 0-100
}

// IcecastStream represents an Icecast radio stream
type IcecastStream struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Genre       string `json:"genre"`
	Description string `json:"description"`
	Bitrate     int    `json:"bitrate"`
	Format      string `json:"format"`
}

// QueueState represents the song queue
type QueueState struct {
	Songs    []Song `json:"songs"`
	Position int    `json:"position"` // Current position in queue
	Autoplay bool   `json:"autoplay"` // Auto-advance to next song when current ends
}

// CountdownState represents the inter-song countdown
type CountdownState struct {
	Active           bool   `json:"active"`             // Countdown is running
	SecondsRemaining int    `json:"seconds_remaining"`  // Seconds until auto-play
	NextSongID       string `json:"next_song_id"`       // ID of next song
	NextSingerKey    string `json:"next_singer_key"`    // MartynKey of next singer
	RequiresApproval bool   `json:"requires_approval"`  // Admin must start (different user)
}

// RoomState represents the entire room state
type RoomState struct {
	Player    PlayerState    `json:"player"`
	Queue     QueueState     `json:"queue"`
	Sessions  []Session      `json:"sessions"`  // Connected clients
	Countdown CountdownState `json:"countdown"` // Inter-song countdown
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
	CDGPath      string    `json:"cdg_path,omitempty"`   // Path to CDG graphics file
	AudioPath    string    `json:"audio_path,omitempty"` // Path to audio file (for CDG)
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

// QueueMode represents how songs are ordered in the queue
type QueueMode string

const (
	QueueModeFIFO     QueueMode = "fifo"     // First in, first out (default)
	QueueModeRotation QueueMode = "rotation" // Fair rotation by singer
)

// FeatureSettings holds toggleable feature flags
type FeatureSettings struct {
	PitchControlEnabled   bool      `json:"pitch_control_enabled"`   // Allow key/pitch changes
	TempoControlEnabled   bool      `json:"tempo_control_enabled"`   // Allow tempo/speed changes
	FairRotationEnabled   bool      `json:"fair_rotation_enabled"`   // Use round-robin queue instead of FIFO
	ScrollingTickerEnabled bool     `json:"scrolling_ticker_enabled"` // Show upcoming singers ticker on display
	SingerNameOverlay     bool      `json:"singer_name_overlay"`      // Show singer name at start of songs
}
