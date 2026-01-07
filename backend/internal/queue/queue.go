package queue

import (
	"database/sql"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"songmartyn/pkg/models"
)

// Manager handles the song queue with persistence
type Manager struct {
	db           *sql.DB
	songs        []models.Song
	position     int
	autoplay     bool
	fairRotation bool  // Use round-robin queue instead of FIFO
	mu           sync.RWMutex

	// Callbacks
	onChange func()
}

// NewManager creates a new queue manager with SQLite persistence
func NewManager(dbPath string) (*Manager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create queue table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS queue (
			id TEXT PRIMARY KEY,
			title TEXT,
			artist TEXT,
			duration INTEGER,
			thumbnail_url TEXT,
			video_url TEXT,
			vocal_path TEXT,
			instr_path TEXT,
			vocal_assist TEXT DEFAULT 'OFF',
			added_by TEXT,
			added_at DATETIME,
			queue_order INTEGER
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create queue state table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS queue_state (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			position INTEGER DEFAULT 0,
			autoplay INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		return nil, err
	}

	// Add autoplay column if it doesn't exist (migration for existing DBs)
	db.Exec(`ALTER TABLE queue_state ADD COLUMN autoplay INTEGER DEFAULT 0`)

	// Initialize state if not exists (autoplay defaults to OFF)
	db.Exec(`INSERT OR IGNORE INTO queue_state (id, position, autoplay) VALUES (1, 0, 0)`)

	m := &Manager{
		db:    db,
		songs: []models.Song{},
	}

	// Load existing queue
	if err := m.loadQueue(); err != nil {
		return nil, err
	}

	// Always start with autoplay OFF - requires manual toggle each session
	m.autoplay = false
	m.db.Exec(`UPDATE queue_state SET autoplay = 0 WHERE id = 1`)

	return m, nil
}

// loadQueue loads the queue from SQLite
func (m *Manager) loadQueue() error {
	// Load position and autoplay
	var autoplayInt int
	row := m.db.QueryRow(`SELECT position, COALESCE(autoplay, 0) FROM queue_state WHERE id = 1`)
	row.Scan(&m.position, &autoplayInt)
	m.autoplay = autoplayInt == 1

	// Load songs
	rows, err := m.db.Query(`
		SELECT id, title, artist, duration, thumbnail_url, video_url,
		       vocal_path, instr_path, vocal_assist, added_by, added_at
		FROM queue
		ORDER BY queue_order ASC
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	m.songs = []models.Song{}
	for rows.Next() {
		var song models.Song
		var vocalPath, instrPath sql.NullString
		var addedAt string

		err := rows.Scan(
			&song.ID,
			&song.Title,
			&song.Artist,
			&song.Duration,
			&song.ThumbnailURL,
			&song.VideoURL,
			&vocalPath,
			&instrPath,
			&song.VocalAssist,
			&song.AddedBy,
			&addedAt,
		)
		if err != nil {
			continue
		}

		if vocalPath.Valid {
			song.VocalPath = vocalPath.String
		}
		if instrPath.Valid {
			song.InstrPath = instrPath.String
		}
		song.AddedAt, _ = time.Parse(time.RFC3339, addedAt)

		m.songs = append(m.songs, song)
	}

	return nil
}

// Add adds a song to the end of the queue
func (m *Manager) Add(song models.Song) error {
	m.mu.Lock()
	song.AddedAt = time.Now()

	if m.fairRotation {
		// Fair rotation: insert at position that gives fair turns
		insertPos := m.findFairInsertPosition(song.AddedBy)
		if insertPos < len(m.songs) {
			// Insert in the middle
			m.songs = append(m.songs[:insertPos], append([]models.Song{song}, m.songs[insertPos:]...)...)
		} else {
			// Insert at end
			m.songs = append(m.songs, song)
		}
		// Reorder all songs in DB
		err := m.reorderQueue()
		onChange := m.onChange
		m.mu.Unlock()
		if onChange != nil {
			onChange()
		}
		return err
	}

	// Standard FIFO: append to end
	m.songs = append(m.songs, song)
	err := m.saveSong(song, len(m.songs)-1)
	onChange := m.onChange // Capture callback before unlocking
	m.mu.Unlock()

	// Call onChange AFTER releasing lock to avoid deadlock
	if onChange != nil {
		onChange()
	}
	return err
}

// findFairInsertPosition finds the optimal position to insert a song for fair rotation
// The algorithm ensures singers take turns fairly
func (m *Manager) findFairInsertPosition(singerKey string) int {
	// Only consider upcoming songs (after current position)
	upcomingStart := m.position
	if upcomingStart < 0 {
		upcomingStart = 0
	}

	// Count songs per singer in upcoming queue
	singerCounts := make(map[string]int)
	for i := upcomingStart; i < len(m.songs); i++ {
		singerCounts[m.songs[i].AddedBy]++
	}

	// How many songs does this singer already have?
	thisSingerCount := singerCounts[singerKey]

	// Find the maximum songs any singer has
	maxCount := 0
	for _, count := range singerCounts {
		if count > maxCount {
			maxCount = count
		}
	}

	// If this singer already has the most songs, insert at end
	if thisSingerCount >= maxCount && maxCount > 0 {
		return len(m.songs)
	}

	// Otherwise, insert after the point where all singers have at least thisSingerCount songs
	// This ensures fair rotation
	insertPos := len(m.songs)
	countAtPos := make(map[string]int)

	for i := upcomingStart; i < len(m.songs); i++ {
		countAtPos[m.songs[i].AddedBy]++

		// Check if this singer now has more songs than we do
		if countAtPos[singerKey] > thisSingerCount {
			// Insert before this position
			insertPos = i
			break
		}
	}

	return insertPos
}

// Remove removes a song from the queue by ID
// Returns (currentRemoved, error) - currentRemoved is true if the currently playing song was removed
func (m *Manager) Remove(songID string) (bool, error) {
	m.mu.Lock()
	var err error
	var found bool
	var currentRemoved bool
	for i, song := range m.songs {
		if song.ID == songID {
			// Check if this is the current song
			if i == m.position {
				currentRemoved = true
			}

			m.songs = append(m.songs[:i], m.songs[i+1:]...)

			// Adjust position if needed
			if i < m.position {
				m.position--
			}

			// If position is now out of bounds, adjust to last song (or 0 if empty)
			if m.position >= len(m.songs) && len(m.songs) > 0 {
				m.position = len(m.songs) - 1
			} else if len(m.songs) == 0 {
				m.position = 0
			}

			_, err = m.db.Exec(`DELETE FROM queue WHERE id = ?`, songID)
			m.savePosition()
			found = true
			break
		}
	}
	onChange := m.onChange
	m.mu.Unlock()

	// Call onChange AFTER releasing lock to avoid deadlock
	if found && onChange != nil {
		onChange()
	}
	return currentRemoved, err
}

// Current returns the current song
func (m *Manager) Current() *models.Song {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.position >= 0 && m.position < len(m.songs) {
		return &m.songs[m.position]
	}
	return nil
}

// Next advances to the next song and returns it
func (m *Manager) Next() *models.Song {
	m.mu.Lock()
	var result *models.Song
	var changed bool
	if m.position < len(m.songs)-1 {
		m.position++
		m.savePosition()
		result = &m.songs[m.position]
		changed = true
	}
	onChange := m.onChange
	m.mu.Unlock()

	// Call onChange AFTER releasing lock to avoid deadlock
	if changed && onChange != nil {
		onChange()
	}
	return result
}

// Skip moves the current song to history (advances position even if at the last song)
// Returns the next song if there is one, nil otherwise
func (m *Manager) Skip() *models.Song {
	m.mu.Lock()
	var result *models.Song
	var changed bool

	if m.position < len(m.songs) {
		m.position++
		m.savePosition()
		changed = true
		// Return the new current song if there is one
		if m.position < len(m.songs) {
			result = &m.songs[m.position]
		}
	}

	onChange := m.onChange
	m.mu.Unlock()

	if changed && onChange != nil {
		onChange()
	}
	return result
}

// Previous goes back to the previous song
func (m *Manager) Previous() *models.Song {
	m.mu.Lock()
	var result *models.Song
	var changed bool
	if m.position > 0 {
		m.position--
		m.savePosition()
		result = &m.songs[m.position]
		changed = true
	}
	onChange := m.onChange
	m.mu.Unlock()

	// Call onChange AFTER releasing lock to avoid deadlock
	if changed && onChange != nil {
		onChange()
	}
	return result
}

// Move moves a song from one position to another
func (m *Manager) Move(fromIndex, toIndex int) error {
	m.mu.Lock()
	if fromIndex < 0 || fromIndex >= len(m.songs) ||
		toIndex < 0 || toIndex >= len(m.songs) {
		m.mu.Unlock()
		return nil
	}

	song := m.songs[fromIndex]
	m.songs = append(m.songs[:fromIndex], m.songs[fromIndex+1:]...)

	// Insert at new position
	m.songs = append(m.songs[:toIndex], append([]models.Song{song}, m.songs[toIndex:]...)...)

	// Save all positions
	for i, s := range m.songs {
		m.db.Exec(`UPDATE queue SET queue_order = ? WHERE id = ?`, i, s.ID)
	}
	onChange := m.onChange
	m.mu.Unlock()

	// Call onChange AFTER releasing lock to avoid deadlock
	if onChange != nil {
		onChange()
	}
	return nil
}

// Shuffle randomizes the order of songs after the current position
// The current song stays in place, only upcoming songs are shuffled
func (m *Manager) Shuffle() {
	m.mu.Lock()

	// Only shuffle songs after current position
	if m.position >= len(m.songs)-1 {
		m.mu.Unlock()
		return // Nothing to shuffle
	}

	// Get the upcoming songs (after current position)
	upcoming := m.songs[m.position+1:]

	// Fisher-Yates shuffle
	for i := len(upcoming) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		upcoming[i], upcoming[j] = upcoming[j], upcoming[i]
	}

	// Save all positions to database
	for i, s := range m.songs {
		m.db.Exec(`UPDATE queue SET queue_order = ? WHERE id = ?`, i, s.ID)
	}

	onChange := m.onChange
	m.mu.Unlock()

	// Call onChange AFTER releasing lock to avoid deadlock
	if onChange != nil {
		onChange()
	}
}

// Requeue finds a song by ID (typically from history) and re-adds it to the queue with a new user
// Creates a new database entry with a unique ID
func (m *Manager) Requeue(songID string, newAddedBy string) error {
	m.mu.Lock()

	// Find the song
	var songCopy models.Song
	found := false
	for _, song := range m.songs {
		if song.ID == songID {
			// Copy the song
			songCopy = song
			found = true
			break
		}
	}

	if !found {
		m.mu.Unlock()
		return nil // Song not found, silently ignore
	}

	// Check if queue was exhausted before adding
	wasExhausted := m.position >= len(m.songs)

	// Create a new song entry with a unique ID
	newSong := songCopy
	newSong.AddedBy = newAddedBy
	newSong.AddedAt = time.Now()

	// Generate a new unique ID by appending timestamp
	// This ensures a new database entry is created
	newSong.ID = songID + "_" + newSong.AddedAt.Format("20060102150405.000")

	// Get max queue_order to avoid collisions with existing entries
	var maxOrder int
	row := m.db.QueryRow(`SELECT COALESCE(MAX(queue_order), -1) FROM queue`)
	row.Scan(&maxOrder)

	// Add to end of queue
	m.songs = append(m.songs, newSong)
	err := m.saveSong(newSong, maxOrder+1)

	// If the queue was exhausted, set position to the new song so it's playable
	if wasExhausted {
		m.position = len(m.songs) - 1
		m.savePosition()
	}

	onChange := m.onChange
	m.mu.Unlock()

	if onChange != nil {
		onChange()
	}
	return err
}

// BumpUserToEnd moves all songs by a user (after current position) to the end of the queue
// Used when a user marks themselves as AFK
func (m *Manager) BumpUserToEnd(martynKey string) {
	m.mu.Lock()

	// Find all songs by this user that are after current position
	var userSongs []models.Song
	var otherSongs []models.Song

	for i, song := range m.songs {
		if i <= m.position {
			// Keep songs at or before current position in place
			otherSongs = append(otherSongs, song)
		} else if song.AddedBy == martynKey {
			userSongs = append(userSongs, song)
		} else {
			otherSongs = append(otherSongs, song)
		}
	}

	// If no user songs found after current, nothing to do
	if len(userSongs) == 0 {
		m.mu.Unlock()
		return
	}

	// Rebuild queue: other songs first, then user songs at end
	m.songs = append(otherSongs, userSongs...)

	// Save all positions to database
	for i, s := range m.songs {
		m.db.Exec(`UPDATE queue SET queue_order = ? WHERE id = ?`, i, s.ID)
	}

	onChange := m.onChange
	m.mu.Unlock()

	// Call onChange AFTER releasing lock to avoid deadlock
	if onChange != nil {
		onChange()
	}
}

// GetState returns the current queue state
func (m *Manager) GetState() models.QueueState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return models.QueueState{
		Songs:    m.songs,
		Position: m.position,
		Autoplay: m.autoplay,
	}
}

// GetAutoplay returns the current autoplay setting
func (m *Manager) GetAutoplay() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.autoplay
}

// SetAutoplay sets the autoplay setting
func (m *Manager) SetAutoplay(enabled bool) {
	m.mu.Lock()
	m.autoplay = enabled
	m.db.Exec(`UPDATE queue_state SET autoplay = ? WHERE id = 1`, btoi(enabled))
	onChange := m.onChange
	m.mu.Unlock()

	if onChange != nil {
		onChange()
	}
}

// SetFairRotation enables or disables fair rotation mode
// When enabled, songs are inserted to ensure singers take fair turns
func (m *Manager) SetFairRotation(enabled bool) {
	m.mu.Lock()
	m.fairRotation = enabled
	m.mu.Unlock()
}

// GetFairRotation returns whether fair rotation is enabled
func (m *Manager) GetFairRotation() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fairRotation
}

// btoi converts bool to int for SQLite
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// IsEmpty returns true if the queue is empty or exhausted
func (m *Manager) IsEmpty() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.songs) == 0 || m.position >= len(m.songs)
}

// Clear removes all songs from the queue
func (m *Manager) Clear() error {
	m.mu.Lock()
	m.songs = []models.Song{}
	m.position = 0

	_, err := m.db.Exec(`DELETE FROM queue`)
	m.savePosition()
	onChange := m.onChange
	m.mu.Unlock()

	// Call onChange AFTER releasing lock to avoid deadlock
	if onChange != nil {
		onChange()
	}
	return err
}

// RemoveByUser removes all songs added by a specific user
// Returns true if the current song was removed (needs skip)
func (m *Manager) RemoveByUser(martynKey string) (bool, error) {
	m.mu.Lock()

	currentRemoved := false
	newSongs := make([]models.Song, 0, len(m.songs))
	newPosition := m.position
	removedCount := 0

	for i, song := range m.songs {
		if song.AddedBy == martynKey {
			// Remove from database
			m.db.Exec(`DELETE FROM queue WHERE id = ?`, song.ID)

			// Check if this is the current song
			if i == m.position {
				currentRemoved = true
			}

			// Adjust position if this song was before current
			if i < m.position {
				newPosition--
			}
			removedCount++
		} else {
			newSongs = append(newSongs, song)
		}
	}

	m.songs = newSongs
	m.position = newPosition

	// If position is now invalid, reset to 0
	if m.position < 0 {
		m.position = 0
	}
	if m.position >= len(m.songs) && len(m.songs) > 0 {
		m.position = len(m.songs) - 1
	}

	// Update queue order in database
	for i, song := range m.songs {
		m.db.Exec(`UPDATE queue SET queue_order = ? WHERE id = ?`, i, song.ID)
	}
	m.savePosition()

	onChange := m.onChange
	m.mu.Unlock()

	// Call onChange AFTER releasing lock to avoid deadlock
	if removedCount > 0 && onChange != nil {
		onChange()
	}

	return currentRemoved, nil
}

// UpdateSongPaths updates the vocal/instrumental paths for a song
func (m *Manager) UpdateSongPaths(songID, vocalPath, instrPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.songs {
		if m.songs[i].ID == songID {
			m.songs[i].VocalPath = vocalPath
			m.songs[i].InstrPath = instrPath

			_, err := m.db.Exec(
				`UPDATE queue SET vocal_path = ?, instr_path = ? WHERE id = ?`,
				vocalPath, instrPath, songID,
			)
			return err
		}
	}
	return nil
}

// OnChange sets the callback for queue changes
func (m *Manager) OnChange(fn func()) {
	m.onChange = fn
}

// saveSong persists a song to SQLite
func (m *Manager) saveSong(song models.Song, order int) error {
	_, err := m.db.Exec(`
		INSERT OR REPLACE INTO queue
		(id, title, artist, duration, thumbnail_url, video_url,
		 vocal_path, instr_path, vocal_assist, added_by, added_at, queue_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		song.ID,
		song.Title,
		song.Artist,
		song.Duration,
		song.ThumbnailURL,
		song.VideoURL,
		song.VocalPath,
		song.InstrPath,
		song.VocalAssist,
		song.AddedBy,
		song.AddedAt.Format(time.RFC3339),
		order,
	)
	return err
}

// reorderQueue saves all songs with their current position in the queue
// Must be called with lock held
func (m *Manager) reorderQueue() error {
	for i, song := range m.songs {
		if err := m.saveSong(song, i); err != nil {
			return err
		}
	}
	return nil
}

// savePosition persists the queue position
func (m *Manager) savePosition() {
	m.db.Exec(`UPDATE queue_state SET position = ? WHERE id = 1`, m.position)
}

// Close closes the database connection
func (m *Manager) Close() error {
	return m.db.Close()
}

// ToJSON returns the queue as JSON for debugging
func (m *Manager) ToJSON() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, _ := json.MarshalIndent(m.GetState(), "", "  ")
	return string(data)
}
