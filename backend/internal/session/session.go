package session

import (
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"songmartyn/internal/avatar"
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
	db.Exec(`ALTER TABLE sessions ADD COLUMN avatar_config TEXT DEFAULT ''`)

	// Create blocked_users table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS blocked_users (
			martyn_key TEXT PRIMARY KEY,
			blocked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			blocked_until DATETIME,
			reason TEXT DEFAULT ''
		)
	`)
	if err != nil {
		return nil, err
	}

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
		       COALESCE(user_agent, ''), COALESCE(is_admin, 0),
		       COALESCE(avatar_config, '')
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
		var avatarConfigJSON string

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
			&avatarConfigJSON,
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

		// Load avatar config from JSON
		if avatarConfigJSON != "" {
			var avatarConfig models.AvatarConfig
			if err := json.Unmarshal([]byte(avatarConfigJSON), &avatarConfig); err == nil {
				session.AvatarConfig = &avatarConfig
			}
		}

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
			// Generate avatar if missing (for sessions created before avatar support)
			if session.AvatarConfig == nil {
				randomAvatar := avatar.NewRandomWithColors()
				session.AvatarConfig = &models.AvatarConfig{
					Env:   randomAvatar.Env,
					Clo:   randomAvatar.Clo,
					Head:  randomAvatar.Head,
					Mouth: randomAvatar.Mouth,
					Eyes:  randomAvatar.Eyes,
					Top:   randomAvatar.Top,
				}
				if randomAvatar.Colors != nil {
					session.AvatarConfig.Colors = &models.AvatarColors{
						Env:   randomAvatar.Colors.Env,
						Clo:   randomAvatar.Colors.Clo,
						Head:  randomAvatar.Colors.Head,
						Mouth: randomAvatar.Colors.Mouth,
						Eyes:  randomAvatar.Colors.Eyes,
						Top:   randomAvatar.Colors.Top,
					}
				}
			}
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

	// Generate random avatar with colors for new user
	randomAvatar := avatar.NewRandomWithColors()
	avatarConfig := &models.AvatarConfig{
		Env:   randomAvatar.Env,
		Clo:   randomAvatar.Clo,
		Head:  randomAvatar.Head,
		Mouth: randomAvatar.Mouth,
		Eyes:  randomAvatar.Eyes,
		Top:   randomAvatar.Top,
	}
	if randomAvatar.Colors != nil {
		avatarConfig.Colors = &models.AvatarColors{
			Env:   randomAvatar.Colors.Env,
			Clo:   randomAvatar.Colors.Clo,
			Head:  randomAvatar.Colors.Head,
			Mouth: randomAvatar.Colors.Mouth,
			Eyes:  randomAvatar.Colors.Eyes,
			Top:   randomAvatar.Colors.Top,
		}
	}

	session := &models.Session{
		MartynKey:     newKey,
		DisplayName:   displayName,
		AvatarConfig:  avatarConfig,
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

	// Serialize avatar config to JSON
	avatarConfigJSON := ""
	if session.AvatarConfig != nil {
		if data, err := json.Marshal(session.AvatarConfig); err == nil {
			avatarConfigJSON = string(data)
		}
	}

	_, err := m.db.Exec(`
		INSERT OR REPLACE INTO sessions
		(martyn_key, display_name, vocal_assist, search_history,
		 current_song_id, connected_at, last_seen_at,
		 ip_address, device_name, user_agent, is_admin, avatar_config)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		avatarConfigJSON,
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

// SetAFK sets the AFK status for a session
func (m *Manager) SetAFK(martynKey string, isAFK bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[martynKey]
	if !ok {
		return nil
	}

	session.IsAFK = isAFK
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

// UpdateAvatarConfig updates a session's avatar configuration
func (m *Manager) UpdateAvatarConfig(martynKey string, config *models.AvatarConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[martynKey]
	if !ok {
		return nil
	}

	session.AvatarConfig = config
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

// BlockUser blocks a user for a specified duration (0 = permanent)
func (m *Manager) BlockUser(martynKey string, duration time.Duration, reason string) error {
	var blockedUntil *time.Time
	if duration > 0 {
		t := time.Now().Add(duration)
		blockedUntil = &t
	}

	_, err := m.db.Exec(`
		INSERT OR REPLACE INTO blocked_users (martyn_key, blocked_at, blocked_until, reason)
		VALUES (?, CURRENT_TIMESTAMP, ?, ?)
	`, martynKey, blockedUntil, reason)
	return err
}

// UnblockUser removes a user from the block list
func (m *Manager) UnblockUser(martynKey string) error {
	_, err := m.db.Exec(`DELETE FROM blocked_users WHERE martyn_key = ?`, martynKey)
	return err
}

// IsBlocked checks if a user is currently blocked
func (m *Manager) IsBlocked(martynKey string) (bool, string) {
	var reason string
	var blockedUntil sql.NullTime

	err := m.db.QueryRow(`
		SELECT reason, blocked_until FROM blocked_users WHERE martyn_key = ?
	`, martynKey).Scan(&reason, &blockedUntil)

	if err == sql.ErrNoRows {
		return false, ""
	}
	if err != nil {
		return false, ""
	}

	// If blocked_until is set and has passed, auto-unblock
	if blockedUntil.Valid && time.Now().After(blockedUntil.Time) {
		m.UnblockUser(martynKey)
		return false, ""
	}

	return true, reason
}

// BlockedUser represents a blocked user entry
type BlockedUser struct {
	MartynKey    string
	DisplayName  string
	BlockedAt    time.Time
	BlockedUntil *time.Time
	Reason       string
}

// GetBlockedUsers returns all currently blocked users
func (m *Manager) GetBlockedUsers() []BlockedUser {
	rows, err := m.db.Query(`
		SELECT b.martyn_key, b.blocked_at, b.blocked_until, b.reason,
		       COALESCE(s.display_name, 'Unknown') as display_name
		FROM blocked_users b
		LEFT JOIN sessions s ON b.martyn_key = s.martyn_key
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var users []BlockedUser
	for rows.Next() {
		var user BlockedUser
		var blockedAt string
		var blockedUntil sql.NullString

		err := rows.Scan(&user.MartynKey, &blockedAt, &blockedUntil, &user.Reason, &user.DisplayName)
		if err != nil {
			continue
		}

		user.BlockedAt, _ = time.Parse(time.RFC3339, blockedAt)
		if blockedUntil.Valid {
			if t, err := time.Parse(time.RFC3339, blockedUntil.String); err == nil {
				// Skip expired blocks
				if t.Before(time.Now()) {
					m.UnblockUser(user.MartynKey)
					continue
				}
				user.BlockedUntil = &t
			}
		}

		users = append(users, user)
	}

	return users
}

// FlushSessions clears all sessions from the database and memory
func (m *Manager) FlushSessions() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear memory
	m.sessions = make(map[string]*models.Session)

	// Clear database
	_, err := m.db.Exec(`DELETE FROM sessions`)
	return err
}

// FlushBlockedUsers clears all blocked users
func (m *Manager) FlushBlockedUsers() error {
	_, err := m.db.Exec(`DELETE FROM blocked_users`)
	return err
}

// GetSessionCount returns the number of sessions in the database
func (m *Manager) GetSessionCount() int {
	var count int
	m.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&count)
	return count
}

// GetBlockedUserCount returns the number of blocked users
func (m *Manager) GetBlockedUserCount() int {
	var count int
	m.db.QueryRow(`SELECT COUNT(*) FROM blocked_users`).Scan(&count)
	return count
}
