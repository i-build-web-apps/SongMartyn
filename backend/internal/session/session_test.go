package session

import (
	"os"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	// Create a temp database
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create manager
	manager, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}
}

func TestCreateSession(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create a new session (empty key = new session)
	session := manager.GetOrCreate("", "TestUser")

	if session == nil {
		t.Fatal("Session should not be nil")
	}

	if session.MartynKey == "" {
		t.Error("MartynKey should be generated")
	}

	if session.DisplayName != "TestUser" {
		t.Errorf("Expected DisplayName 'TestUser', got '%s'", session.DisplayName)
	}

	// Default values
	if session.IsAdmin {
		t.Error("New session should not be admin by default")
	}

	if session.NameLocked {
		t.Error("New session should not have name locked by default")
	}

	if session.AvatarConfig == nil {
		t.Error("New session should have an avatar config")
	}
}

func TestGetExistingSession(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create first session
	session1 := manager.GetOrCreate("", "FirstUser")
	originalKey := session1.MartynKey
	session1.DisplayName = "TestUser"
	manager.Update(session1)

	// Get same session again by key
	session2 := manager.GetOrCreate(originalKey, "IgnoredName")

	// Should be the same session
	if session2.MartynKey != originalKey {
		t.Error("Should return the same session")
	}

	if session2.DisplayName != "TestUser" {
		t.Errorf("Expected DisplayName 'TestUser', got '%s'", session2.DisplayName)
	}
}

func TestUpdateProfile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create session
	session := manager.GetOrCreate("", "OldName")

	// Update profile
	err = manager.UpdateProfile(session.MartynKey, "NewName", "avatar123")
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	// Verify update
	if session.DisplayName != "NewName" {
		t.Errorf("Expected DisplayName 'NewName', got '%s'", session.DisplayName)
	}

	if session.AvatarID != "avatar123" {
		t.Errorf("Expected AvatarID 'avatar123', got '%s'", session.AvatarID)
	}
}

func TestUpdateProfileNameLocked(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create session
	session := manager.GetOrCreate("", "OriginalName")
	session.NameLocked = true
	manager.Update(session)

	// Try to update profile with name locked
	err = manager.UpdateProfile(session.MartynKey, "NewName", "")
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	// Name should NOT change because it's locked
	if session.DisplayName != "OriginalName" {
		t.Errorf("Name should remain locked at 'OriginalName', got '%s'", session.DisplayName)
	}
}

func TestSetNameLocked(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create session
	session := manager.GetOrCreate("", "TestUser")

	if session.NameLocked {
		t.Error("NameLocked should be false initially")
	}

	// Lock the name
	err = manager.SetNameLocked(session.MartynKey, true)
	if err != nil {
		t.Fatalf("SetNameLocked failed: %v", err)
	}

	if !session.NameLocked {
		t.Error("NameLocked should be true after setting")
	}

	// Unlock the name
	err = manager.SetNameLocked(session.MartynKey, false)
	if err != nil {
		t.Fatalf("SetNameLocked failed: %v", err)
	}

	if session.NameLocked {
		t.Error("NameLocked should be false after unsetting")
	}
}

func TestIsNameLocked(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create session
	session := manager.GetOrCreate("", "TestUser")
	session.NameLocked = true
	manager.Update(session)

	if !manager.IsNameLocked(session.MartynKey) {
		t.Error("IsNameLocked should return true")
	}

	// Non-existent session
	if manager.IsNameLocked("nonexistent") {
		t.Error("IsNameLocked should return false for non-existent session")
	}
}

func TestAdminSetDisplayName(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create session
	session := manager.GetOrCreate("", "OldName")
	session.NameLocked = true // Even if locked, admin can change
	manager.Update(session)

	// Admin sets new name
	err = manager.AdminSetDisplayName(session.MartynKey, "AdminSetName")
	if err != nil {
		t.Fatalf("AdminSetDisplayName failed: %v", err)
	}

	if session.DisplayName != "AdminSetName" {
		t.Errorf("Expected DisplayName 'AdminSetName', got '%s'", session.DisplayName)
	}
}

func TestAdminSetDisplayNameNonExistent(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Try to set name on non-existent session - should return nil (no error)
	err = manager.AdminSetDisplayName("nonexistent", "Name")
	if err != nil {
		t.Error("Should not error for non-existent session")
	}
}

func TestSetAdmin(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create session
	session := manager.GetOrCreate("", "TestUser")

	if session.IsAdmin {
		t.Error("IsAdmin should be false initially")
	}

	// Promote to admin
	err = manager.SetAdmin(session.MartynKey, true)
	if err != nil {
		t.Fatalf("SetAdmin failed: %v", err)
	}

	if !session.IsAdmin {
		t.Error("IsAdmin should be true after promotion")
	}

	// Demote from admin
	err = manager.SetAdmin(session.MartynKey, false)
	if err != nil {
		t.Fatalf("SetAdmin failed: %v", err)
	}

	if session.IsAdmin {
		t.Error("IsAdmin should be false after demotion")
	}
}

func TestBlockUnblock(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create session
	session := manager.GetOrCreate("", "TestUser")

	// Block the user
	err = manager.BlockUser(session.MartynKey, 60*time.Minute, "Test reason")
	if err != nil {
		t.Fatalf("BlockUser failed: %v", err)
	}

	isBlocked, reason := manager.IsBlocked(session.MartynKey)
	if !isBlocked {
		t.Error("Session should be blocked")
	}
	if reason != "Test reason" {
		t.Errorf("Expected reason 'Test reason', got '%s'", reason)
	}

	// Unblock the user
	err = manager.UnblockUser(session.MartynKey)
	if err != nil {
		t.Fatalf("UnblockUser failed: %v", err)
	}

	isBlocked, _ = manager.IsBlocked(session.MartynKey)
	if isBlocked {
		t.Error("Session should not be blocked after unblock")
	}

	// Verify session still exists
	if session.MartynKey == "" {
		t.Error("Session should still exist after unblock")
	}
}

func TestSetAFK(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create session
	session := manager.GetOrCreate("", "TestUser")

	if session.IsAFK {
		t.Error("IsAFK should be false initially")
	}

	// Set AFK
	err = manager.SetAFK(session.MartynKey, true)
	if err != nil {
		t.Fatalf("SetAFK failed: %v", err)
	}

	if !session.IsAFK {
		t.Error("IsAFK should be true after setting")
	}

	// Unset AFK
	err = manager.SetAFK(session.MartynKey, false)
	if err != nil {
		t.Fatalf("SetAFK failed: %v", err)
	}

	if session.IsAFK {
		t.Error("IsAFK should be false after unsetting")
	}
}

func TestPersistence(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	var savedKey string

	// Create manager and session
	func() {
		manager1, err := NewManager(tmpFile.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer manager1.Close()

		session1 := manager1.GetOrCreate("", "PersistentUser")
		savedKey = session1.MartynKey
		session1.DisplayName = "PersistentName"
		manager1.SetAdmin(savedKey, true)
		manager1.SetNameLocked(savedKey, true)
	}()

	// Reopen manager
	manager2, err := NewManager(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer manager2.Close()

	// Session should be loaded from database
	session2 := manager2.GetOrCreate(savedKey, "")

	if session2.DisplayName != "PersistentName" {
		t.Errorf("Expected DisplayName 'PersistentName', got '%s'", session2.DisplayName)
	}

	if !session2.NameLocked {
		t.Error("NameLocked should be persisted as true")
	}

	if !session2.IsAdmin {
		t.Error("IsAdmin should be persisted as true")
	}
}

func TestGetAllSessions(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create multiple sessions
	manager.GetOrCreate("", "User1")
	manager.GetOrCreate("", "User2")
	manager.GetOrCreate("", "User3")

	sessions := manager.GetAllSessions()
	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}
}

func TestUpdateDeviceInfo(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	session := manager.GetOrCreate("", "TestUser")

	err = manager.UpdateDeviceInfo(session.MartynKey, "192.168.1.1", "TestAgent/1.0", "iPhone")
	if err != nil {
		t.Fatalf("UpdateDeviceInfo failed: %v", err)
	}

	if session.IPAddress != "192.168.1.1" {
		t.Errorf("Expected IPAddress '192.168.1.1', got '%s'", session.IPAddress)
	}

	if session.UserAgent != "TestAgent/1.0" {
		t.Errorf("Expected UserAgent 'TestAgent/1.0', got '%s'", session.UserAgent)
	}

	if session.DeviceName != "iPhone" {
		t.Errorf("Expected DeviceName 'iPhone', got '%s'", session.DeviceName)
	}
}

func TestFlushSessions(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "session_test_*.db")
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

	// Create sessions
	manager.GetOrCreate("", "User1")
	manager.GetOrCreate("", "User2")

	if manager.GetSessionCount() != 2 {
		t.Errorf("Expected 2 sessions, got %d", manager.GetSessionCount())
	}

	// Flush
	err = manager.FlushSessions()
	if err != nil {
		t.Fatalf("FlushSessions failed: %v", err)
	}

	if manager.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions after flush, got %d", manager.GetSessionCount())
	}
}
