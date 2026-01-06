package browser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSessionManager tests session management functionality.
func TestSessionManager(t *testing.T) {
	t.Run("NewSessionManager", func(t *testing.T) {
		sm := NewSessionManager("/tmp/test-workspace", Config{
			Host:    "localhost",
			Port:    0,
			Timeout: 30 * time.Second,
		})

		if sm == nil {
			t.Fatal("NewSessionManager() returned nil")
		}
		if sm.workspaceDir != "/tmp/test-workspace" {
			t.Errorf("workspaceDir = %s, want /tmp/test-workspace", sm.workspaceDir)
		}
		if sm.config.Host != "localhost" {
			t.Errorf("config.Host = %s, want localhost", sm.config.Host)
		}
	})

	t.Run("validateSession", func(t *testing.T) {
		sm := NewSessionManager("/tmp/test-workspace", Config{
			Host:    "localhost",
			Port:    0,
			Timeout: 30 * time.Second,
		})

		// Create a temp directory for user data
		tmpDir := t.TempDir()

		tests := []struct {
			name    string
			session *Session
			wantErr bool
		}{
			{
				name: "valid session",
				session: &Session{
					PID:         1234,
					Port:        9222,
					Host:        "localhost",
					UserDataDir: tmpDir,
					StartedAt:   time.Now(),
				},
				wantErr: false,
			},
			{
				name: "invalid PID",
				session: &Session{
					PID:       0,
					Port:      9222,
					Host:      "localhost",
					StartedAt: time.Now(),
				},
				wantErr: true,
			},
			{
				name: "invalid port",
				session: &Session{
					PID:       1234,
					Port:      100,
					Host:      "localhost",
					StartedAt: time.Now(),
				},
				wantErr: true,
			},
			{
				name: "empty host",
				session: &Session{
					PID:       1234,
					Port:      9222,
					Host:      "",
					StartedAt: time.Now(),
				},
				wantErr: true,
			},
			{
				name: "stale session",
				session: &Session{
					PID:       1234,
					Port:      9222,
					Host:      "localhost",
					StartedAt: time.Now().Add(-25 * time.Hour),
				},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := sm.validateSession(tt.session)
				if (err != nil) != tt.wantErr {
					t.Errorf("validateSession() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("saveAndLoadSession", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm := NewSessionManager(tmpDir, Config{
			Host:    "localhost",
			Port:    0,
			Timeout: 30 * time.Second,
		})

		session := &Session{
			PID:         12345,
			Port:        9223,
			Host:        "localhost",
			UserDataDir: "/tmp/test-user-data",
			StartedAt:   time.Now(),
		}

		// Save session
		if err := sm.saveSession(session); err != nil {
			t.Fatalf("saveSession() failed: %v", err)
		}

		// Verify file exists
		sessionPath := filepath.Join(tmpDir, ".mehrhof/browser.json")
		if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
			t.Error("session file was not created")
		}

		// Load session
		sm2 := NewSessionManager(tmpDir, Config{})
		loaded, err := sm2.loadSession()
		if err != nil {
			t.Fatalf("loadSession() failed: %v", err)
		}

		if loaded.PID != session.PID {
			t.Errorf("loaded.PID = %d, want %d", loaded.PID, session.PID)
		}
		if loaded.Port != session.Port {
			t.Errorf("loaded.Port = %d, want %d", loaded.Port, session.Port)
		}
		if loaded.Host != session.Host {
			t.Errorf("loaded.Host = %s, want %s", loaded.Host, session.Host)
		}
		if loaded.UserDataDir != session.UserDataDir {
			t.Errorf("loaded.UserDataDir = %s, want %s", loaded.UserDataDir, session.UserDataDir)
		}
	})

	t.Run("loadSession_corruptedJSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm := NewSessionManager(tmpDir, Config{})

		// Write corrupted JSON
		sessionPath := filepath.Join(tmpDir, ".mehrhof/browser.json")
		if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(sessionPath, []byte("{invalid json"), 0o644); err != nil {
			t.Fatalf("failed to write corrupted file: %v", err)
		}

		// Load should fail and backup the file
		_, err := sm.loadSession()
		if err == nil {
			t.Error("loadSession() should return error for corrupted JSON")
		}

		// Verify backup was created (name includes timestamp)
		backupExists := false
		files, _ := os.ReadDir(filepath.Dir(sessionPath))
		for _, f := range files {
			// Backup files are named: browser.json.corrupted.{timestamp}
			if strings.Contains(f.Name(), ".corrupted.") {
				backupExists = true

				break
			}
		}
		if !backupExists {
			t.Error("corrupted file was not backed up")
		}

		// Verify original file was removed
		if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
			t.Error("corrupted session file was not removed")
		}
	})

	t.Run("isProcessAlive", func(t *testing.T) {
		sm := NewSessionManager("/tmp/test-workspace", Config{})

		// Test with current process (should be alive)
		currentPID := os.Getpid()
		if !sm.isProcessAlive(currentPID) {
			t.Error("isProcessAlive(currentPID) returned false, want true")
		}

		// Test with invalid PID (should not be alive)
		if sm.isProcessAlive(-1) {
			t.Error("isProcessAlive(-1) returned true, want false")
		}

		// Test with unlikely PID
		if sm.isProcessAlive(999999) {
			t.Error("isProcessAlive(999999) returned true, want false")
		}
	})

	t.Run("GetSession", func(t *testing.T) {
		sm := NewSessionManager("/tmp/test-workspace", Config{})

		// Initially should return nil
		if sm.GetSession() != nil {
			t.Error("GetSession() should return nil initially")
		}

		// After setting session
		session := &Session{
			PID:       1234,
			Port:      9222,
			Host:      "localhost",
			StartedAt: time.Now(),
		}
		sm.session = session

		if got := sm.GetSession(); got != session {
			t.Error("GetSession() did not return the set session")
		}
	})
}

// TestSessionJSONHandling tests JSON marshaling/unmarshaling of sessions.
func TestSessionJSONHandling(t *testing.T) {
	session := &Session{
		PID:         12345,
		Port:        9223,
		Host:        "localhost",
		UserDataDir: "/tmp/test-user-data",
		StartedAt:   time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC),
	}

	// Marshal
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	// Unmarshal
	var unmarshaled Session
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	// Verify fields match
	if unmarshaled.PID != session.PID {
		t.Errorf("PID = %d, want %d", unmarshaled.PID, session.PID)
	}
	if unmarshaled.Port != session.Port {
		t.Errorf("Port = %d, want %d", unmarshaled.Port, session.Port)
	}
	if unmarshaled.Host != session.Host {
		t.Errorf("Host = %s, want %s", unmarshaled.Host, session.Host)
	}
	if unmarshaled.UserDataDir != session.UserDataDir {
		t.Errorf("UserDataDir = %s, want %s", unmarshaled.UserDataDir, session.UserDataDir)
	}
}
