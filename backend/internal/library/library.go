package library

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"songmartyn/pkg/models"

	_ "github.com/mattn/go-sqlite3"
)

// Supported audio/video extensions
var supportedExtensions = map[string]bool{
	".mp3":  true,
	".mp4":  true,
	".m4a":  true,
	".wav":  true,
	".flac": true,
	".ogg":  true,
	".webm": true,
	".mkv":  true,
	".avi":  true,
}

// Manager handles the song library
type Manager struct {
	db *sql.DB
}

// NewManager creates a new library manager
func NewManager(dbPath string) (*Manager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	m := &Manager{db: db}
	if err := m.initDB(); err != nil {
		return nil, err
	}

	return m, nil
}

// initDB creates the necessary tables
func (m *Manager) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS library_locations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		song_count INTEGER DEFAULT 0,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_scan DATETIME
	);

	CREATE TABLE IF NOT EXISTS library_songs (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		artist TEXT DEFAULT '',
		album TEXT DEFAULT '',
		duration INTEGER DEFAULT 0,
		file_path TEXT UNIQUE NOT NULL,
		thumbnail_url TEXT DEFAULT '',
		vocal_path TEXT DEFAULT '',
		instr_path TEXT DEFAULT '',
		library_id INTEGER NOT NULL,
		times_sung INTEGER DEFAULT 0,
		last_sung_at DATETIME,
		last_sung_by TEXT,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (library_id) REFERENCES library_locations(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS song_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		song_id TEXT NOT NULL,
		martyn_key TEXT NOT NULL,
		sung_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		song_title TEXT NOT NULL,
		song_artist TEXT DEFAULT '',
		FOREIGN KEY (song_id) REFERENCES library_songs(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_songs_title ON library_songs(title);
	CREATE INDEX IF NOT EXISTS idx_songs_artist ON library_songs(artist);
	CREATE INDEX IF NOT EXISTS idx_songs_library ON library_songs(library_id);
	CREATE INDEX IF NOT EXISTS idx_history_martyn ON song_history(martyn_key);
	CREATE INDEX IF NOT EXISTS idx_history_song ON song_history(song_id);

	CREATE TABLE IF NOT EXISTS search_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		query TEXT NOT NULL,
		source TEXT NOT NULL,
		results_count INTEGER DEFAULT 0,
		martyn_key TEXT DEFAULT '',
		ip_address TEXT DEFAULT '',
		searched_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_search_logs_query ON search_logs(query);
	CREATE INDEX IF NOT EXISTS idx_search_logs_source ON search_logs(source);
	CREATE INDEX IF NOT EXISTS idx_search_logs_searched_at ON search_logs(searched_at);
	`

	_, err := m.db.Exec(schema)
	return err
}

// AddLocation adds a new library location
func (m *Manager) AddLocation(path, name string) (*models.LibraryLocation, error) {
	// Validate path exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	// Normalize path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	result, err := m.db.Exec(
		"INSERT INTO library_locations (path, name) VALUES (?, ?)",
		absPath, name,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &models.LibraryLocation{
		ID:      id,
		Path:    absPath,
		Name:    name,
		AddedAt: time.Now(),
	}, nil
}

// RemoveLocation removes a library location and its songs
func (m *Manager) RemoveLocation(id int64) error {
	_, err := m.db.Exec("DELETE FROM library_locations WHERE id = ?", id)
	return err
}

// GetLocations returns all library locations
func (m *Manager) GetLocations() ([]models.LibraryLocation, error) {
	rows, err := m.db.Query(`
		SELECT id, path, name, song_count, added_at, COALESCE(last_scan, added_at)
		FROM library_locations ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []models.LibraryLocation
	for rows.Next() {
		var loc models.LibraryLocation
		var addedAtStr, lastScanStr string
		if err := rows.Scan(&loc.ID, &loc.Path, &loc.Name, &loc.SongCount, &addedAtStr, &lastScanStr); err != nil {
			return nil, err
		}
		// Parse datetime strings from SQLite
		loc.AddedAt, _ = time.Parse("2006-01-02 15:04:05", addedAtStr)
		loc.LastScan, _ = time.Parse("2006-01-02 15:04:05", lastScanStr)
		locations = append(locations, loc)
	}
	return locations, nil
}

// ScanLocation scans a library location for media files
func (m *Manager) ScanLocation(id int64) (int, error) {
	// Get location path
	var path string
	err := m.db.QueryRow("SELECT path FROM library_locations WHERE id = ?", id).Scan(&path)
	if err != nil {
		return 0, err
	}

	count := 0
	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(filePath))
		if !supportedExtensions[ext] {
			return nil
		}

		// Generate ID from file path
		hash := md5.Sum([]byte(filePath))
		songID := hex.EncodeToString(hash[:])

		// Parse title and artist from filename
		title, artist := parseFilename(filePath)

		// Insert or update song
		_, err = m.db.Exec(`
			INSERT INTO library_songs (id, title, artist, file_path, library_id)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				title = excluded.title,
				artist = excluded.artist,
				file_path = excluded.file_path
		`, songID, title, artist, filePath, id)
		if err != nil {
			log.Printf("Error adding song %s: %v", filePath, err)
			return nil
		}

		count++
		return nil
	})

	if err != nil {
		return count, err
	}

	// Update location stats
	m.db.Exec(`
		UPDATE library_locations
		SET song_count = (SELECT COUNT(*) FROM library_songs WHERE library_id = ?),
		    last_scan = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id, id)

	return count, nil
}

// parseFilename extracts title and artist from filename
// Supports formats like "Artist - Title.mp3" or just "Title.mp3"
func parseFilename(path string) (title, artist string) {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Try "Artist - Title" format
	parts := strings.SplitN(name, " - ", 2)
	if len(parts) == 2 {
		artist = strings.TrimSpace(parts[0])
		title = strings.TrimSpace(parts[1])
	} else {
		title = strings.TrimSpace(name)
	}

	return title, artist
}

// SearchSongs searches the library for songs
func (m *Manager) SearchSongs(query string, limit int) ([]models.LibrarySong, error) {
	if limit <= 0 {
		limit = 50
	}

	searchTerm := "%" + query + "%"
	rows, err := m.db.Query(`
		SELECT id, title, artist, album, duration, file_path, thumbnail_url,
		       vocal_path, instr_path, library_id, times_sung, last_sung_at, last_sung_by, added_at
		FROM library_songs
		WHERE title LIKE ? OR artist LIKE ? OR album LIKE ?
		ORDER BY times_sung DESC, title ASC
		LIMIT ?
	`, searchTerm, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []models.LibrarySong
	for rows.Next() {
		var song models.LibrarySong
		var lastSungAt sql.NullTime
		var lastSungBy sql.NullString
		if err := rows.Scan(
			&song.ID, &song.Title, &song.Artist, &song.Album, &song.Duration,
			&song.FilePath, &song.ThumbnailURL, &song.VocalPath, &song.InstrPath,
			&song.LibraryID, &song.TimesSung, &lastSungAt, &lastSungBy, &song.AddedAt,
		); err != nil {
			return nil, err
		}
		if lastSungAt.Valid {
			song.LastSungAt = &lastSungAt.Time
		}
		if lastSungBy.Valid {
			song.LastSungBy = lastSungBy.String
		}
		songs = append(songs, song)
	}
	return songs, nil
}

// GetSong returns a song by ID
func (m *Manager) GetSong(id string) (*models.LibrarySong, error) {
	var song models.LibrarySong
	var lastSungAt sql.NullTime
	var lastSungBy sql.NullString
	err := m.db.QueryRow(`
		SELECT id, title, artist, album, duration, file_path, thumbnail_url,
		       vocal_path, instr_path, library_id, times_sung, last_sung_at, last_sung_by, added_at
		FROM library_songs WHERE id = ?
	`, id).Scan(
		&song.ID, &song.Title, &song.Artist, &song.Album, &song.Duration,
		&song.FilePath, &song.ThumbnailURL, &song.VocalPath, &song.InstrPath,
		&song.LibraryID, &song.TimesSung, &lastSungAt, &lastSungBy, &song.AddedAt,
	)
	if err != nil {
		return nil, err
	}
	if lastSungAt.Valid {
		song.LastSungAt = &lastSungAt.Time
	}
	if lastSungBy.Valid {
		song.LastSungBy = lastSungBy.String
	}
	return &song, nil
}

// RecordSongPlayed records that a user sang a song
func (m *Manager) RecordSongPlayed(songID, martynKey string) error {
	// Get song details for history
	song, err := m.GetSong(songID)
	if err != nil {
		return err
	}

	// Add to history
	_, err = m.db.Exec(`
		INSERT INTO song_history (song_id, martyn_key, song_title, song_artist)
		VALUES (?, ?, ?, ?)
	`, songID, martynKey, song.Title, song.Artist)
	if err != nil {
		return err
	}

	// Update song stats
	_, err = m.db.Exec(`
		UPDATE library_songs
		SET times_sung = times_sung + 1,
		    last_sung_at = CURRENT_TIMESTAMP,
		    last_sung_by = ?
		WHERE id = ?
	`, martynKey, songID)

	return err
}

// GetUserHistory returns a user's song history
func (m *Manager) GetUserHistory(martynKey string, limit int) ([]models.SongHistory, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := m.db.Query(`
		SELECT id, song_id, martyn_key, sung_at, song_title, song_artist
		FROM song_history
		WHERE martyn_key = ?
		ORDER BY sung_at DESC
		LIMIT ?
	`, martynKey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.SongHistory
	for rows.Next() {
		var h models.SongHistory
		if err := rows.Scan(&h.ID, &h.SongID, &h.MartynKey, &h.SungAt, &h.SongTitle, &h.SongArtist); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

// GetPopularSongs returns the most sung songs
func (m *Manager) GetPopularSongs(limit int) ([]models.LibrarySong, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := m.db.Query(`
		SELECT id, title, artist, album, duration, file_path, thumbnail_url,
		       vocal_path, instr_path, library_id, times_sung, last_sung_at, last_sung_by, added_at
		FROM library_songs
		WHERE times_sung > 0
		ORDER BY times_sung DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []models.LibrarySong
	for rows.Next() {
		var song models.LibrarySong
		var lastSungAt sql.NullTime
		var lastSungBy sql.NullString
		if err := rows.Scan(
			&song.ID, &song.Title, &song.Artist, &song.Album, &song.Duration,
			&song.FilePath, &song.ThumbnailURL, &song.VocalPath, &song.InstrPath,
			&song.LibraryID, &song.TimesSung, &lastSungAt, &lastSungBy, &song.AddedAt,
		); err != nil {
			return nil, err
		}
		if lastSungAt.Valid {
			song.LastSungAt = &lastSungAt.Time
		}
		if lastSungBy.Valid {
			song.LastSungBy = lastSungBy.String
		}
		songs = append(songs, song)
	}
	return songs, nil
}

// GetStats returns library statistics
func (m *Manager) GetStats() (totalSongs, totalPlays int, err error) {
	err = m.db.QueryRow("SELECT COUNT(*) FROM library_songs").Scan(&totalSongs)
	if err != nil {
		return
	}
	err = m.db.QueryRow("SELECT COALESCE(SUM(times_sung), 0) FROM library_songs").Scan(&totalPlays)
	return
}

// LogSearch logs a search query and its results
func (m *Manager) LogSearch(query, source string, resultsCount int, martynKey, ipAddress string) error {
	_, err := m.db.Exec(`
		INSERT INTO search_logs (query, source, results_count, martyn_key, ip_address)
		VALUES (?, ?, ?, ?, ?)
	`, query, source, resultsCount, martynKey, ipAddress)
	return err
}

// SearchLog represents a logged search
type SearchLog struct {
	ID           int64  `json:"id"`
	Query        string `json:"query"`
	Source       string `json:"source"`
	ResultsCount int    `json:"results_count"`
	MartynKey    string `json:"martyn_key"`
	IPAddress    string `json:"ip_address"`
	SearchedAt   string `json:"searched_at"`
}

// GetSearchLogs returns recent search logs
func (m *Manager) GetSearchLogs(limit int, source string) ([]SearchLog, error) {
	if limit <= 0 {
		limit = 100
	}

	var rows *sql.Rows
	var err error

	if source != "" {
		rows, err = m.db.Query(`
			SELECT id, query, source, results_count, martyn_key, ip_address, searched_at
			FROM search_logs
			WHERE source = ?
			ORDER BY searched_at DESC
			LIMIT ?
		`, source, limit)
	} else {
		rows, err = m.db.Query(`
			SELECT id, query, source, results_count, martyn_key, ip_address, searched_at
			FROM search_logs
			ORDER BY searched_at DESC
			LIMIT ?
		`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []SearchLog
	for rows.Next() {
		var log SearchLog
		if err := rows.Scan(&log.ID, &log.Query, &log.Source, &log.ResultsCount, &log.MartynKey, &log.IPAddress, &log.SearchedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// GetSearchStats returns search statistics
func (m *Manager) GetSearchStats() (totalSearches, uniqueQueries, notFoundCount int, topQueries []SearchLog, err error) {
	err = m.db.QueryRow("SELECT COUNT(*) FROM search_logs").Scan(&totalSearches)
	if err != nil {
		return
	}
	err = m.db.QueryRow("SELECT COUNT(DISTINCT query) FROM search_logs").Scan(&uniqueQueries)
	if err != nil {
		return
	}
	err = m.db.QueryRow("SELECT COUNT(*) FROM search_logs WHERE results_count = 0").Scan(&notFoundCount)
	if err != nil {
		return
	}

	// Get top queries with no results (potential songs to add)
	rows, err := m.db.Query(`
		SELECT query, source, COUNT(*) as search_count
		FROM search_logs
		WHERE results_count = 0
		GROUP BY query
		ORDER BY search_count DESC
		LIMIT 20
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var log SearchLog
		var count int
		if err = rows.Scan(&log.Query, &log.Source, &count); err != nil {
			return
		}
		log.ResultsCount = count // Repurpose to show how many times searched
		topQueries = append(topQueries, log)
	}
	return
}

// ClearSearchLogs clears all search logs
func (m *Manager) ClearSearchLogs() error {
	_, err := m.db.Exec("DELETE FROM search_logs")
	return err
}

// Close closes the database connection
func (m *Manager) Close() error {
	return m.db.Close()
}
