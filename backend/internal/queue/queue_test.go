package queue

import (
	"os"
	"testing"
	"time"

	"songmartyn/pkg/models"
)

func TestNewManager(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	// Queue should be empty initially
	if !manager.IsEmpty() {
		t.Error("New queue should be empty")
	}
}

func createTestSong(id, title, artist, addedBy string) models.Song {
	return models.Song{
		ID:          id,
		Title:       title,
		Artist:      artist,
		Duration:    180,
		VocalAssist: models.VocalOff,
		AddedBy:     addedBy,
		AddedAt:     time.Now(),
	}
}

func TestAddSong(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	song := createTestSong("song1", "Test Song", "Test Artist", "user1")
	err = manager.Add(song)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if manager.IsEmpty() {
		t.Error("Queue should not be empty after adding song")
	}

	state := manager.GetState()
	if len(state.Songs) != 1 {
		t.Errorf("Expected 1 song, got %d", len(state.Songs))
	}

	if state.Songs[0].Title != "Test Song" {
		t.Errorf("Expected title 'Test Song', got '%s'", state.Songs[0].Title)
	}
}

func TestAddMultipleSongs(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	songs := []models.Song{
		createTestSong("song1", "Song One", "Artist A", "user1"),
		createTestSong("song2", "Song Two", "Artist B", "user2"),
		createTestSong("song3", "Song Three", "Artist C", "user3"),
	}

	for _, song := range songs {
		if err := manager.Add(song); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	state := manager.GetState()
	if len(state.Songs) != 3 {
		t.Errorf("Expected 3 songs, got %d", len(state.Songs))
	}

	// Check order
	if state.Songs[0].Title != "Song One" {
		t.Errorf("First song should be 'Song One', got '%s'", state.Songs[0].Title)
	}
	if state.Songs[2].Title != "Song Three" {
		t.Errorf("Third song should be 'Song Three', got '%s'", state.Songs[2].Title)
	}
}

func TestRemoveSong(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	manager.Add(createTestSong("song1", "Song One", "Artist", "user1"))
	manager.Add(createTestSong("song2", "Song Two", "Artist", "user1"))

	currentRemoved, err := manager.Remove("song1")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if !currentRemoved {
		t.Error("Should indicate current song was removed")
	}

	state := manager.GetState()
	if len(state.Songs) != 1 {
		t.Errorf("Expected 1 song after removal, got %d", len(state.Songs))
	}

	if state.Songs[0].ID != "song2" {
		t.Errorf("Remaining song should be 'song2', got '%s'", state.Songs[0].ID)
	}
}

func TestCurrent(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	// Empty queue
	if manager.Current() != nil {
		t.Error("Current should be nil for empty queue")
	}

	manager.Add(createTestSong("song1", "Song One", "Artist", "user1"))
	manager.Add(createTestSong("song2", "Song Two", "Artist", "user1"))

	current := manager.Current()
	if current == nil {
		t.Fatal("Current should not be nil")
	}

	if current.Title != "Song One" {
		t.Errorf("Current should be 'Song One', got '%s'", current.Title)
	}
}

func TestNext(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	manager.Add(createTestSong("song1", "Song One", "Artist", "user1"))
	manager.Add(createTestSong("song2", "Song Two", "Artist", "user1"))
	manager.Add(createTestSong("song3", "Song Three", "Artist", "user1"))

	// Advance to next
	next := manager.Next()
	if next == nil {
		t.Fatal("Next should not be nil")
	}

	if next.Title != "Song Two" {
		t.Errorf("Next should be 'Song Two', got '%s'", next.Title)
	}

	// Current should now be Song Two
	current := manager.Current()
	if current.Title != "Song Two" {
		t.Errorf("Current should be 'Song Two', got '%s'", current.Title)
	}

	// Position should be 1
	state := manager.GetState()
	if state.Position != 1 {
		t.Errorf("Position should be 1, got %d", state.Position)
	}
}

func TestSkip(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	manager.Add(createTestSong("song1", "Song One", "Artist", "user1"))
	manager.Add(createTestSong("song2", "Song Two", "Artist", "user1"))

	// Skip first song
	next := manager.Skip()
	if next == nil {
		t.Fatal("Skip should return next song")
	}

	if next.Title != "Song Two" {
		t.Errorf("Next should be 'Song Two', got '%s'", next.Title)
	}

	// Skip second (last) song - should return nil
	next = manager.Skip()
	if next != nil {
		t.Error("Skip on last song should return nil")
	}

	// Queue should be exhausted (position past last song)
	if !manager.IsEmpty() {
		t.Error("Queue should be empty (exhausted) after skipping last song")
	}
}

func TestMove(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	manager.Add(createTestSong("song1", "Song One", "Artist", "user1"))
	manager.Add(createTestSong("song2", "Song Two", "Artist", "user1"))
	manager.Add(createTestSong("song3", "Song Three", "Artist", "user1"))

	// Move song3 to position 1 (second slot)
	err = manager.Move(2, 1)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	state := manager.GetState()
	if state.Songs[0].Title != "Song One" {
		t.Errorf("First song should be 'Song One', got '%s'", state.Songs[0].Title)
	}
	if state.Songs[1].Title != "Song Three" {
		t.Errorf("Second song should be 'Song Three', got '%s'", state.Songs[1].Title)
	}
	if state.Songs[2].Title != "Song Two" {
		t.Errorf("Third song should be 'Song Two', got '%s'", state.Songs[2].Title)
	}
}

func TestShuffle(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	// Add many songs
	for i := 0; i < 10; i++ {
		manager.Add(createTestSong(
			string(rune('a'+i)),
			"Song "+string(rune('A'+i)),
			"Artist",
			"user1",
		))
	}

	// Current song should stay at position 0
	firstSong := manager.Current().Title

	manager.Shuffle()

	// First song should remain the same (current song)
	state := manager.GetState()
	if state.Songs[0].Title != firstSong {
		t.Errorf("First song should remain '%s' after shuffle, got '%s'", firstSong, state.Songs[0].Title)
	}

	// Total should still be 10
	if len(state.Songs) != 10 {
		t.Errorf("Should still have 10 songs after shuffle, got %d", len(state.Songs))
	}
}

func TestClear(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	manager.Add(createTestSong("song1", "Song One", "Artist", "user1"))
	manager.Add(createTestSong("song2", "Song Two", "Artist", "user1"))

	err = manager.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if !manager.IsEmpty() {
		t.Error("Queue should be empty after clear")
	}

	state := manager.GetState()
	if len(state.Songs) != 0 {
		t.Errorf("Expected 0 songs, got %d", len(state.Songs))
	}
}

func TestAutoplay(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	// Default should be off
	if manager.GetAutoplay() {
		t.Error("Autoplay should be off by default")
	}

	// Enable autoplay
	manager.SetAutoplay(true)
	if !manager.GetAutoplay() {
		t.Error("Autoplay should be enabled")
	}

	// Disable autoplay
	manager.SetAutoplay(false)
	if manager.GetAutoplay() {
		t.Error("Autoplay should be disabled")
	}
}

func TestRemoveByUser(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	manager.Add(createTestSong("song1", "Song One", "Artist", "user1"))
	manager.Add(createTestSong("song2", "Song Two", "Artist", "user2"))
	manager.Add(createTestSong("song3", "Song Three", "Artist", "user1"))
	manager.Add(createTestSong("song4", "Song Four", "Artist", "user2"))

	// Remove all songs by user1
	currentRemoved, err := manager.RemoveByUser("user1")
	if err != nil {
		t.Fatalf("RemoveByUser failed: %v", err)
	}

	if !currentRemoved {
		t.Error("Current song (by user1) should be reported as removed")
	}

	state := manager.GetState()
	if len(state.Songs) != 2 {
		t.Errorf("Expected 2 songs after removal, got %d", len(state.Songs))
	}

	// All remaining should be from user2
	for _, song := range state.Songs {
		if song.AddedBy != "user2" {
			t.Errorf("All remaining songs should be by user2, got '%s'", song.AddedBy)
		}
	}
}

func TestBumpUserToEnd(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	manager.Add(createTestSong("song1", "Song One", "Artist", "user1"))
	manager.Add(createTestSong("song2", "Song Two", "Artist", "user1"))
	manager.Add(createTestSong("song3", "Song Three", "Artist", "user2"))
	manager.Add(createTestSong("song4", "Song Four", "Artist", "user1"))

	// Bump user1's songs to end (but not current song at position 0)
	manager.BumpUserToEnd("user1")

	state := manager.GetState()

	// Song One should still be at position 0 (current song)
	if state.Songs[0].Title != "Song One" {
		t.Errorf("Current song should remain 'Song One', got '%s'", state.Songs[0].Title)
	}

	// Song Three (user2) should be second (position 1)
	if state.Songs[1].Title != "Song Three" {
		t.Errorf("Second song should be 'Song Three', got '%s'", state.Songs[1].Title)
	}

	// User1's other songs should be at end
	if state.Songs[2].AddedBy != "user1" || state.Songs[3].AddedBy != "user1" {
		t.Error("User1's songs should be bumped to end")
	}
}

func TestRequeue(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	manager.Add(createTestSong("song1", "Song One", "Artist A", "user1"))

	// Requeue the song for a different user
	err = manager.Requeue("song1", "user2")
	if err != nil {
		t.Fatalf("Requeue failed: %v", err)
	}

	state := manager.GetState()
	if len(state.Songs) != 2 {
		t.Errorf("Expected 2 songs after requeue, got %d", len(state.Songs))
	}

	// Original should still exist
	if state.Songs[0].AddedBy != "user1" {
		t.Error("Original song should still be by user1")
	}

	// New copy should be by user2 with different ID
	if state.Songs[1].AddedBy != "user2" {
		t.Error("Requeued song should be by user2")
	}

	// IDs should be different
	if state.Songs[0].ID == state.Songs[1].ID {
		t.Error("Requeued song should have different ID")
	}
}

func TestRequeueWhenExhausted(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	// Add a song and skip past it (exhaust the queue)
	manager.Add(createTestSong("song1", "Song One", "Artist A", "user1"))
	manager.Skip() // Move position past the only song

	state := manager.GetState()
	if state.Position != 1 {
		t.Fatalf("Expected position 1 after skip, got %d", state.Position)
	}

	// Queue should be exhausted (position >= len(songs))
	if state.Position < len(state.Songs) {
		t.Fatal("Queue should be exhausted")
	}

	// Requeue the song - it should become the next playable song
	err = manager.Requeue("song1", "user2")
	if err != nil {
		t.Fatalf("Requeue failed: %v", err)
	}

	state = manager.GetState()

	// After requeue, the new song should be at the current position (upcoming, not history)
	if len(state.Songs) != 2 {
		t.Errorf("Expected 2 songs after requeue, got %d", len(state.Songs))
	}

	// Position should now point to the new song
	if state.Position != 1 {
		t.Errorf("Expected position 1 after requeue, got %d", state.Position)
	}

	// The song at position should be the requeued one (by user2)
	if state.Position < len(state.Songs) && state.Songs[state.Position].AddedBy != "user2" {
		t.Error("Requeued song should be at current position")
	}

	// Original should be in history (position 0, which is < current position 1)
	if state.Songs[0].AddedBy != "user1" {
		t.Error("Original song should still be in history")
	}
}

func TestRequeueOrderCollision(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	// Add multiple songs
	manager.Add(createTestSong("song1", "Song One", "Artist", "user1"))
	manager.Add(createTestSong("song2", "Song Two", "Artist", "user1"))
	manager.Add(createTestSong("song3", "Song Three", "Artist", "user1"))

	// Remove the middle song to create a gap in queue_order
	manager.Remove("song2")

	// Requeue song1 - should get a unique queue_order, not collide with song3
	err = manager.Requeue("song1", "user2")
	if err != nil {
		t.Fatalf("Requeue failed: %v", err)
	}

	state := manager.GetState()
	if len(state.Songs) != 3 {
		t.Errorf("Expected 3 songs after requeue, got %d", len(state.Songs))
	}

	// All songs should have unique IDs
	ids := make(map[string]bool)
	for _, song := range state.Songs {
		if ids[song.ID] {
			t.Errorf("Duplicate song ID found: %s", song.ID)
		}
		ids[song.ID] = true
	}
}

func TestOnChange(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	changeCount := 0
	manager.OnChange(func() {
		changeCount++
	})

	manager.Add(createTestSong("song1", "Song One", "Artist", "user1"))
	if changeCount != 1 {
		t.Errorf("Expected 1 change callback, got %d", changeCount)
	}

	manager.Add(createTestSong("song2", "Song Two", "Artist", "user1"))
	if changeCount != 2 {
		t.Errorf("Expected 2 change callbacks, got %d", changeCount)
	}

	manager.Next()
	if changeCount != 3 {
		t.Errorf("Expected 3 change callbacks, got %d", changeCount)
	}
}

func TestPersistence(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create manager and add songs
	manager1, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	manager1.Add(createTestSong("song1", "Persisted Song One", "Artist", "user1"))
	manager1.Add(createTestSong("song2", "Persisted Song Two", "Artist", "user1"))
	manager1.SetAutoplay(true)
	manager1.Next() // Move to position 1
	manager1.Close()

	// Reopen manager
	manager2, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager2.Close()

	state := manager2.GetState()

	if len(state.Songs) != 2 {
		t.Errorf("Expected 2 songs after reload, got %d", len(state.Songs))
	}

	if state.Songs[0].Title != "Persisted Song One" {
		t.Errorf("First song should be 'Persisted Song One', got '%s'", state.Songs[0].Title)
	}

	if state.Position != 1 {
		t.Errorf("Position should be persisted as 1, got %d", state.Position)
	}

	if !state.Autoplay {
		t.Error("Autoplay should be persisted as true")
	}
}

func TestUpdateSongPaths(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	manager.Add(createTestSong("song1", "Song One", "Artist", "user1"))

	err = manager.UpdateSongPaths("song1", "/path/to/vocals.wav", "/path/to/instr.wav")
	if err != nil {
		t.Fatalf("UpdateSongPaths failed: %v", err)
	}

	current := manager.Current()
	if current.VocalPath != "/path/to/vocals.wav" {
		t.Errorf("VocalPath should be updated, got '%s'", current.VocalPath)
	}
	if current.InstrPath != "/path/to/instr.wav" {
		t.Errorf("InstrPath should be updated, got '%s'", current.InstrPath)
	}
}
