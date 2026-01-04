package queue

import (
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"songmartyn/pkg/models"
)

// Manager handles the song queue with persistence
type Manager struct {
	db       *sql.DB
	songs    []models.Song
	position int
	mu       sync.RWMutex

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
			position INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		return nil, err
	}

	// Initialize state if not exists
	db.Exec(`INSERT OR IGNORE INTO queue_state (id, position) VALUES (1, 0)`)

	m := &Manager{
		db:    db,
		songs: []models.Song{},
	}

	// Load existing queue
	if err := m.loadQueue(); err != nil {
		return nil, err
	}

	return m, nil
}

// loadQueue loads the queue from SQLite
func (m *Manager) loadQueue() error {
	// Load position
	row := m.db.QueryRow(`SELECT position FROM queue_state WHERE id = 1`)
	row.Scan(&m.position)

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
	defer m.mu.Unlock()

	song.AddedAt = time.Now()
	m.songs = append(m.songs, song)

	err := m.saveSong(song, len(m.songs)-1)
	if m.onChange != nil {
		m.onChange()
	}
	return err
}

// Remove removes a song from the queue by ID
func (m *Manager) Remove(songID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, song := range m.songs {
		if song.ID == songID {
			m.songs = append(m.songs[:i], m.songs[i+1:]...)

			// Adjust position if needed
			if i < m.position {
				m.position--
			}

			_, err := m.db.Exec(`DELETE FROM queue WHERE id = ?`, songID)
			m.savePosition()

			if m.onChange != nil {
				m.onChange()
			}
			return err
		}
	}
	return nil
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
	defer m.mu.Unlock()

	if m.position < len(m.songs)-1 {
		m.position++
		m.savePosition()
		if m.onChange != nil {
			m.onChange()
		}
		return &m.songs[m.position]
	}
	return nil
}

// Previous goes back to the previous song
func (m *Manager) Previous() *models.Song {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.position > 0 {
		m.position--
		m.savePosition()
		if m.onChange != nil {
			m.onChange()
		}
		return &m.songs[m.position]
	}
	return nil
}

// Move moves a song from one position to another
func (m *Manager) Move(fromIndex, toIndex int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if fromIndex < 0 || fromIndex >= len(m.songs) ||
		toIndex < 0 || toIndex >= len(m.songs) {
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

	if m.onChange != nil {
		m.onChange()
	}
	return nil
}

// GetState returns the current queue state
func (m *Manager) GetState() models.QueueState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return models.QueueState{
		Songs:    m.songs,
		Position: m.position,
	}
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
	defer m.mu.Unlock()

	m.songs = []models.Song{}
	m.position = 0

	_, err := m.db.Exec(`DELETE FROM queue`)
	m.savePosition()

	if m.onChange != nil {
		m.onChange()
	}
	return err
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
