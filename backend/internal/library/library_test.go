package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// Manager Initialization Tests
// =============================================================================

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	if m.db == nil {
		t.Error("Expected db to be initialized")
	}
}

// =============================================================================
// Location Management Tests
// =============================================================================

func TestAddLocation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	// Create a test directory to add as a location
	songsDir := filepath.Join(tmpDir, "songs")
	if err := os.Mkdir(songsDir, 0755); err != nil {
		t.Fatalf("Failed to create songs dir: %v", err)
	}

	loc, err := m.AddLocation(songsDir, "Test Songs")
	if err != nil {
		t.Fatalf("Failed to add location: %v", err)
	}

	if loc.Name != "Test Songs" {
		t.Errorf("Expected name 'Test Songs', got '%s'", loc.Name)
	}
	if loc.ID == 0 {
		t.Error("Expected non-zero ID")
	}
}

func TestAddLocationWithSpaces(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	// Create a directory with spaces in the name
	songsDir := filepath.Join(tmpDir, "Songs More")
	if err := os.Mkdir(songsDir, 0755); err != nil {
		t.Fatalf("Failed to create songs dir: %v", err)
	}

	loc, err := m.AddLocation(songsDir, "Songs More")
	if err != nil {
		t.Fatalf("Failed to add location with spaces: %v", err)
	}

	if loc.Name != "Songs More" {
		t.Errorf("Expected name 'Songs More', got '%s'", loc.Name)
	}
}

func TestAddNonExistentLocation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	_, err = m.AddLocation("/nonexistent/path", "Test")
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

func TestGetLocations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	// Add two locations
	dir1 := filepath.Join(tmpDir, "songs1")
	dir2 := filepath.Join(tmpDir, "songs2")
	os.Mkdir(dir1, 0755)
	os.Mkdir(dir2, 0755)

	m.AddLocation(dir1, "Songs 1")
	m.AddLocation(dir2, "Songs 2")

	locations, err := m.GetLocations()
	if err != nil {
		t.Fatalf("Failed to get locations: %v", err)
	}

	if len(locations) != 2 {
		t.Errorf("Expected 2 locations, got %d", len(locations))
	}
}

func TestRemoveLocation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	loc, _ := m.AddLocation(songsDir, "Test Songs")

	err = m.RemoveLocation(loc.ID)
	if err != nil {
		t.Fatalf("Failed to remove location: %v", err)
	}

	locations, _ := m.GetLocations()
	if len(locations) != 0 {
		t.Errorf("Expected 0 locations after removal, got %d", len(locations))
	}
}

// =============================================================================
// Scanning Tests
// =============================================================================

func TestScanEmptyLocation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	loc, _ := m.AddLocation(songsDir, "Empty Songs")

	count, err := m.ScanLocation(loc.ID)
	if err != nil {
		t.Fatalf("Failed to scan empty location: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 songs in empty location, got %d", count)
	}
}

func TestScanCDGAudioPair(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	// Create CDG + MP3 pair
	cdgPath := filepath.Join(songsDir, "Artist - Song Title.cdg")
	mp3Path := filepath.Join(songsDir, "Artist - Song Title.mp3")
	os.WriteFile(cdgPath, []byte("fake cdg"), 0644)
	os.WriteFile(mp3Path, []byte("fake mp3"), 0644)

	loc, _ := m.AddLocation(songsDir, "Test Songs")

	count, err := m.ScanLocation(loc.ID)
	if err != nil {
		t.Fatalf("Failed to scan location: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 CDG+Audio pair, got %d", count)
	}

	// Verify the song was added correctly
	songs, _ := m.SearchSongs("Song Title", 10)
	if len(songs) != 1 {
		t.Fatalf("Expected 1 song in search results, got %d", len(songs))
	}

	if songs[0].CDGPath != cdgPath {
		t.Errorf("Expected CDG path '%s', got '%s'", cdgPath, songs[0].CDGPath)
	}
	if songs[0].AudioPath != mp3Path {
		t.Errorf("Expected audio path '%s', got '%s'", mp3Path, songs[0].AudioPath)
	}
}

func TestScanCDGAudioPairInFolderWithSpaces(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	// Create a folder with spaces in the name (like "Songs More")
	songsDir := filepath.Join(tmpDir, "Songs More")
	os.Mkdir(songsDir, 0755)

	// Create CDG + MP3 pair with spaces in filename
	cdgPath := filepath.Join(songsDir, "Police - Every Little Thing She Does Is Magic.cdg")
	mp3Path := filepath.Join(songsDir, "Police - Every Little Thing She Does Is Magic.mp3")
	os.WriteFile(cdgPath, []byte("fake cdg"), 0644)
	os.WriteFile(mp3Path, []byte("fake mp3"), 0644)

	loc, err := m.AddLocation(songsDir, "Songs More")
	if err != nil {
		t.Fatalf("Failed to add location with spaces: %v", err)
	}

	count, err := m.ScanLocation(loc.ID)
	if err != nil {
		t.Fatalf("Failed to scan location with spaces: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 song in folder with spaces, got %d", count)
	}

	// Verify song can be found
	songs, _ := m.SearchSongs("Police", 10)
	if len(songs) != 1 {
		t.Errorf("Expected 1 song in search, got %d", len(songs))
	}
}

func TestScanMultipleCDGPairs(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	// Create multiple CDG + MP3 pairs
	pairs := []string{
		"Artist1 - Song1",
		"Artist2 - Song2",
		"Artist3 - Song3",
	}

	for _, name := range pairs {
		os.WriteFile(filepath.Join(songsDir, name+".cdg"), []byte("fake cdg"), 0644)
		os.WriteFile(filepath.Join(songsDir, name+".mp3"), []byte("fake mp3"), 0644)
	}

	loc, _ := m.AddLocation(songsDir, "Test Songs")

	count, err := m.ScanLocation(loc.ID)
	if err != nil {
		t.Fatalf("Failed to scan location: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 CDG+Audio pairs, got %d", count)
	}
}

func TestScanVideoFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	// Create an MP4 video file
	mp4Path := filepath.Join(songsDir, "Artist - Video Song.mp4")
	os.WriteFile(mp4Path, []byte("fake mp4"), 0644)

	loc, _ := m.AddLocation(songsDir, "Test Songs")

	count, err := m.ScanLocation(loc.ID)
	if err != nil {
		t.Fatalf("Failed to scan location: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 video file, got %d", count)
	}
}

func TestScanMixedContent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	// Create CDG + MP3 pair
	os.WriteFile(filepath.Join(songsDir, "CDG Song - Title.cdg"), []byte("fake cdg"), 0644)
	os.WriteFile(filepath.Join(songsDir, "CDG Song - Title.mp3"), []byte("fake mp3"), 0644)

	// Create standalone MP4
	os.WriteFile(filepath.Join(songsDir, "Video Song - Title.mp4"), []byte("fake mp4"), 0644)

	// Create standalone MP3 (no CDG pair)
	os.WriteFile(filepath.Join(songsDir, "Audio Only - Title.mp3"), []byte("fake mp3"), 0644)

	loc, _ := m.AddLocation(songsDir, "Test Songs")

	count, err := m.ScanLocation(loc.ID)
	if err != nil {
		t.Fatalf("Failed to scan location: %v", err)
	}

	// Should find: 1 CDG pair + 1 MP4 + 1 standalone MP3 = 3
	if count != 3 {
		t.Errorf("Expected 3 songs (1 CDG pair + 1 MP4 + 1 MP3), got %d", count)
	}
}

func TestScanSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	subDir := filepath.Join(songsDir, "subfolder")
	os.MkdirAll(subDir, 0755)

	// File in root
	os.WriteFile(filepath.Join(songsDir, "Root - Song.mp4"), []byte("fake mp4"), 0644)

	// File in subfolder
	os.WriteFile(filepath.Join(subDir, "Sub - Song.mp4"), []byte("fake mp4"), 0644)

	loc, _ := m.AddLocation(songsDir, "Test Songs")

	count, err := m.ScanLocation(loc.ID)
	if err != nil {
		t.Fatalf("Failed to scan location: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 songs (1 root + 1 subfolder), got %d", count)
	}
}

func TestScanIgnoresUnsupportedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	// Create unsupported files
	os.WriteFile(filepath.Join(songsDir, "readme.txt"), []byte("text"), 0644)
	os.WriteFile(filepath.Join(songsDir, "image.jpg"), []byte("image"), 0644)
	os.WriteFile(filepath.Join(songsDir, "document.pdf"), []byte("pdf"), 0644)

	// Create one supported file
	os.WriteFile(filepath.Join(songsDir, "Song.mp4"), []byte("fake mp4"), 0644)

	loc, _ := m.AddLocation(songsDir, "Test Songs")

	count, err := m.ScanLocation(loc.ID)
	if err != nil {
		t.Fatalf("Failed to scan location: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 song (ignoring unsupported files), got %d", count)
	}
}

func TestScanCDGWithoutAudio(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	// Create orphan CDG file (no matching audio)
	os.WriteFile(filepath.Join(songsDir, "Orphan - Song.cdg"), []byte("fake cdg"), 0644)

	loc, _ := m.AddLocation(songsDir, "Test Songs")

	count, err := m.ScanLocation(loc.ID)
	if err != nil {
		t.Fatalf("Failed to scan location: %v", err)
	}

	// Orphan CDG files should not be added
	if count != 0 {
		t.Errorf("Expected 0 songs (orphan CDG ignored), got %d", count)
	}
}

// =============================================================================
// Filename Parsing Tests
// =============================================================================

func TestParseFilenameArtistTitle(t *testing.T) {
	title, artist := parseFilename("/path/to/Artist Name - Song Title.mp3")

	if artist != "Artist Name" {
		t.Errorf("Expected artist 'Artist Name', got '%s'", artist)
	}
	if title != "Song Title" {
		t.Errorf("Expected title 'Song Title', got '%s'", title)
	}
}

func TestParseFilenameTitleOnly(t *testing.T) {
	title, artist := parseFilename("/path/to/Just A Song Title.mp3")

	if artist != "" {
		t.Errorf("Expected empty artist, got '%s'", artist)
	}
	if title != "Just A Song Title" {
		t.Errorf("Expected title 'Just A Song Title', got '%s'", title)
	}
}

func TestParseFilenameWithMultipleDashes(t *testing.T) {
	title, artist := parseFilename("/path/to/Artist - Song - Part 2.mp3")

	if artist != "Artist" {
		t.Errorf("Expected artist 'Artist', got '%s'", artist)
	}
	if title != "Song - Part 2" {
		t.Errorf("Expected title 'Song - Part 2', got '%s'", title)
	}
}

func TestParseFilenameWithParentheses(t *testing.T) {
	title, artist := parseFilename("/path/to/Stevie Wonder - Superstition (Igwa mix).cdg")

	if artist != "Stevie Wonder" {
		t.Errorf("Expected artist 'Stevie Wonder', got '%s'", artist)
	}
	if title != "Superstition (Igwa mix)" {
		t.Errorf("Expected title 'Superstition (Igwa mix)', got '%s'", title)
	}
}

// =============================================================================
// Search Tests
// =============================================================================

func TestSearchSongs(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	// Create test songs
	os.WriteFile(filepath.Join(songsDir, "Beatles - Yesterday.mp4"), []byte("fake"), 0644)
	os.WriteFile(filepath.Join(songsDir, "Beatles - Hey Jude.mp4"), []byte("fake"), 0644)
	os.WriteFile(filepath.Join(songsDir, "Queen - Bohemian Rhapsody.mp4"), []byte("fake"), 0644)

	loc, _ := m.AddLocation(songsDir, "Test Songs")
	m.ScanLocation(loc.ID)

	// Search for Beatles
	songs, err := m.SearchSongs("Beatles", 10)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(songs) != 2 {
		t.Errorf("Expected 2 Beatles songs, got %d", len(songs))
	}
}

func TestSearchSongsCaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	os.WriteFile(filepath.Join(songsDir, "ABBA - Dancing Queen.mp4"), []byte("fake"), 0644)

	loc, _ := m.AddLocation(songsDir, "Test Songs")
	m.ScanLocation(loc.ID)

	// Search with lowercase
	songs, _ := m.SearchSongs("abba", 10)
	if len(songs) != 1 {
		t.Errorf("Expected case-insensitive search to find 1 song, got %d", len(songs))
	}

	// Search with mixed case
	songs, _ = m.SearchSongs("AbBa", 10)
	if len(songs) != 1 {
		t.Errorf("Expected case-insensitive search to find 1 song, got %d", len(songs))
	}
}

// =============================================================================
// Rescan Tests
// =============================================================================

func TestRescanUpdatesExistingSongs(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	// Create initial song
	os.WriteFile(filepath.Join(songsDir, "Artist - Song.mp4"), []byte("fake"), 0644)

	loc, _ := m.AddLocation(songsDir, "Test Songs")

	// First scan
	count1, _ := m.ScanLocation(loc.ID)
	if count1 != 1 {
		t.Errorf("First scan: expected 1, got %d", count1)
	}

	// Second scan (rescan) - should not duplicate
	count2, _ := m.ScanLocation(loc.ID)
	if count2 != 1 {
		t.Errorf("Rescan: expected 1, got %d", count2)
	}

	// Verify only one song in database
	songs, _ := m.SearchSongs("Artist", 10)
	if len(songs) != 1 {
		t.Errorf("Expected 1 song after rescan, got %d", len(songs))
	}
}

// =============================================================================
// Real Folder Tests (Integration)
// =============================================================================

func TestScanRealSongsMoreFolder(t *testing.T) {
	// This test uses the actual "Songs More" folder to diagnose scanning issues
	realPath := "/Users/paul/Development/SongMartyn/Songs More"

	// Check if the folder exists
	info, err := os.Stat(realPath)
	if err != nil {
		t.Skipf("Skipping test - real folder not found: %s", realPath)
		return
	}
	if !info.IsDir() {
		t.Skipf("Skipping test - path is not a directory: %s", realPath)
		return
	}

	// List actual files in the folder
	files, err := os.ReadDir(realPath)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	t.Logf("Files in %s:", realPath)
	cdgCount := 0
	mp3Count := 0
	for _, f := range files {
		t.Logf("  - %s (dir=%v)", f.Name(), f.IsDir())
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if ext == ".cdg" {
			cdgCount++
		} else if ext == ".mp3" {
			mp3Count++
		}
	}
	t.Logf("Found %d CDG files and %d MP3 files", cdgCount, mp3Count)

	// Create a test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	// Add the real location
	loc, err := m.AddLocation(realPath, "Songs More")
	if err != nil {
		t.Fatalf("Failed to add real location: %v", err)
	}
	t.Logf("Added location with ID %d, path: %s", loc.ID, loc.Path)

	// Scan the location
	count, err := m.ScanLocation(loc.ID)
	if err != nil {
		t.Fatalf("Failed to scan real location: %v", err)
	}
	t.Logf("Scan found %d songs", count)

	// We expect 2 CDG+MP3 pairs
	if count != 2 {
		t.Errorf("Expected 2 songs from Songs More folder, got %d", count)
	}

	// Verify songs are searchable
	songs, _ := m.SearchSongs("Police", 10)
	t.Logf("Search for 'Police' found %d songs", len(songs))

	songs, _ = m.SearchSongs("Stevie", 10)
	t.Logf("Search for 'Stevie' found %d songs", len(songs))
}

// =============================================================================
// Stats Tests
// =============================================================================

func TestGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	m, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	songsDir := filepath.Join(tmpDir, "songs")
	os.Mkdir(songsDir, 0755)

	os.WriteFile(filepath.Join(songsDir, "Song1.mp4"), []byte("fake"), 0644)
	os.WriteFile(filepath.Join(songsDir, "Song2.mp4"), []byte("fake"), 0644)

	loc, _ := m.AddLocation(songsDir, "Test Songs")
	m.ScanLocation(loc.ID)

	totalSongs, totalPlays, err := m.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if totalSongs != 2 {
		t.Errorf("Expected 2 total songs, got %d", totalSongs)
	}
	if totalPlays != 0 {
		t.Errorf("Expected 0 total plays, got %d", totalPlays)
	}
}
