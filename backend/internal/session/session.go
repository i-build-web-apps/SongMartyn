package session

import (
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"songmartyn/internal/names"
	"songmartyn/pkg/models"
)

// Manager handles session persistence (The Martyn Handshake)
type Manager struct {
	db       *sql.DB
	sessions map[string]*models.Session // In-memory cache
	mu       sync.RWMutex
}

// NewManager creates a new session manager with SQLite persistence
func NewManager(dbPath string) (*Manager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create sessions table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			martyn_key TEXT PRIMARY KEY,
			display_name TEXT,
			vocal_assist TEXT DEFAULT 'OFF',
			search_history TEXT DEFAULT '[]',
			current_song_id TEXT,
			connected_at DATETIME,
			last_seen_at DATETIME,
			ip_address TEXT DEFAULT '',
			device_name TEXT DEFAULT '',
			user_agent TEXT DEFAULT '',
			is_admin INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		return nil, err
	}

	// Add new columns if they don't exist (for existing databases)
	db.Exec(`ALTER TABLE sessions ADD COLUMN ip_address TEXT DEFAULT ''`)
	db.Exec(`ALTER TABLE sessions ADD COLUMN device_name TEXT DEFAULT ''`)
	db.Exec(`ALTER TABLE sessions ADD COLUMN user_agent TEXT DEFAULT ''`)
	db.Exec(`ALTER TABLE sessions ADD COLUMN is_admin INTEGER DEFAULT 0`)

	m := &Manager{
		db:       db,
		sessions: make(map[string]*models.Session),
	}

	// Load existing sessions into memory
	if err := m.loadSessions(); err != nil {
		return nil, err
	}

	return m, nil
}

// loadSessions loads all sessions from SQLite into memory
func (m *Manager) loadSessions() error {
	rows, err := m.db.Query(`
		SELECT martyn_key, display_name, vocal_assist, search_history,
		       current_song_id, connected_at, last_seen_at,
		       COALESCE(ip_address, ''), COALESCE(device_name, ''),
		       COALESCE(user_agent, ''), COALESCE(is_admin, 0)
		FROM sessions
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var session models.Session
		var searchHistoryJSON string
		var currentSongID sql.NullString
		var connectedAt, lastSeenAt string
		var isAdmin int

		err := rows.Scan(
			&session.MartynKey,
			&session.DisplayName,
			&session.VocalAssist,
			&searchHistoryJSON,
			&currentSongID,
			&connectedAt,
			&lastSeenAt,
			&session.IPAddress,
			&session.DeviceName,
			&session.UserAgent,
			&isAdmin,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(searchHistoryJSON), &session.SearchHistory)
		if currentSongID.Valid {
			session.CurrentSongID = currentSongID.String
		}
		session.ConnectedAt, _ = time.Parse(time.RFC3339, connectedAt)
		session.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeenAt)
		session.IsAdmin = isAdmin == 1

		m.sessions[session.MartynKey] = &session
	}

	return nil
}

// GetOrCreate retrieves an existing session by MartynKey or creates a new one
func (m *Manager) GetOrCreate(martynKey, displayName string) *models.Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to find existing session
	if martynKey != "" {
		if session, ok := m.sessions[martynKey]; ok {
			session.LastSeenAt = time.Now()
			m.saveSession(session)
			return session
		}
	}

	// Create new session
	newKey := uuid.New().String()
	if displayName == "" {
		// Generate a unique funny singer name
		existingNames := make(map[string]bool)
		for _, s := range m.sessions {
			existingNames[s.DisplayName] = true
		}
		displayName = names.GenerateUniqueSingerName(existingNames)
	}

	session := &models.Session{
		MartynKey:     newKey,
		DisplayName:   displayName,
		VocalAssist:   models.VocalOff,
		SearchHistory: []string{},
		ConnectedAt:   time.Now(),
		LastSeenAt:    time.Now(),
	}

	m.sessions[newKey] = session
	m.saveSession(session)

	return session
}

// Get retrieves a session by MartynKey
func (m *Manager) Get(martynKey string) *models.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[martynKey]
}

// Update updates a session's properties
func (m *Manager) Update(session *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session.LastSeenAt = time.Now()
	m.sessions[session.MartynKey] = session
	return m.saveSession(session)
}

// UpdateVocalAssist updates a session's vocal assist preference
func (m *Manager) UpdateVocalAssist(martynKey string, level models.VocalAssistLevel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[martynKey]
	if !ok {
		return nil
	}

	session.VocalAssist = level
	session.LastSeenAt = time.Now()
	return m.saveSession(session)
}

// AddSearchHistory adds a search query to the session's history
func (m *Manager) AddSearchHistory(martynKey, query string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[martynKey]
	if !ok {
		return nil
	}

	// Prepend to history (most recent first), limit to 20 entries
	session.SearchHistory = append([]string{query}, session.SearchHistory...)
	if len(session.SearchHistory) > 20 {
		session.SearchHistory = session.SearchHistory[:20]
	}

	session.LastSeenAt = time.Now()
	return m.saveSession(session)
}

// GetActiveSessions returns all sessions seen in the last hour
func (m *Manager) GetActiveSessions() []models.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cutoff := time.Now().Add(-1 * time.Hour)
	var active []models.Session

	for _, session := range m.sessions {
		if session.LastSeenAt.After(cutoff) {
			active = append(active, *session)
		}
	}

	return active
}

// saveSession persists a session to SQLite
func (m *Manager) saveSession(session *models.Session) error {
	searchHistoryJSON, _ := json.Marshal(session.SearchHistory)

	isAdmin := 0
	if session.IsAdmin {
		isAdmin = 1
	}

	_, err := m.db.Exec(`
		INSERT OR REPLACE INTO sessions
		(martyn_key, display_name, vocal_assist, search_history,
		 current_song_id, connected_at, last_seen_at,
		 ip_address, device_name, user_agent, is_admin)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		session.MartynKey,
		session.DisplayName,
		session.VocalAssist,
		string(searchHistoryJSON),
		session.CurrentSongID,
		session.ConnectedAt.Format(time.RFC3339),
		session.LastSeenAt.Format(time.RFC3339),
		session.IPAddress,
		session.DeviceName,
		session.UserAgent,
		isAdmin,
	)
	return err
}

// Close closes the database connection
func (m *Manager) Close() error {
	return m.db.Close()
}

// SetAdmin sets the admin status for a session
func (m *Manager) SetAdmin(martynKey string, isAdmin bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[martynKey]
	if !ok {
		return nil
	}

	session.IsAdmin = isAdmin
	session.LastSeenAt = time.Now()
	return m.saveSession(session)
}

// UpdateDisplayName updates a session's display name
func (m *Manager) UpdateDisplayName(martynKey, displayName string) error {
	return m.UpdateProfile(martynKey, displayName, "")
}

// UpdateProfile updates a session's display name and avatar
func (m *Manager) UpdateProfile(martynKey, displayName, avatarID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[martynKey]
	if !ok {
		return nil
	}

	session.DisplayName = displayName
	if avatarID != "" {
		session.AvatarID = avatarID
	}
	session.LastSeenAt = time.Now()
	return m.saveSession(session)
}

// UpdateDeviceInfo updates a session's device information
func (m *Manager) UpdateDeviceInfo(martynKey, ipAddress, userAgent, deviceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[martynKey]
	if !ok {
		return nil
	}

	session.IPAddress = ipAddress
	session.UserAgent = userAgent
	session.DeviceName = deviceName
	session.LastSeenAt = time.Now()
	return m.saveSession(session)
}

// SetOnline marks a session as online or offline
func (m *Manager) SetOnline(martynKey string, online bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[martynKey]; ok {
		session.IsOnline = online
		if online {
			session.LastSeenAt = time.Now()
		}
	}
}

// GetAllSessions returns all sessions (for admin listing)
func (m *Manager) GetAllSessions() []models.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]models.Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, *session)
	}
	return sessions
}
