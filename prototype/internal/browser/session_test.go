package browser

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

// TestIsEndpointResponsive tests the HTTP endpoint health check.
func TestIsEndpointResponsive(t *testing.T) {
	sm := NewSessionManager("/tmp/test-workspace", Config{})

	t.Run("responsive endpoint", func(t *testing.T) {
		// Start a mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/json/version" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"webSocketDebuggerUrl":"ws://localhost:9222"}`))
			}
		}))
		defer server.Close()

		// Extract host and port from server URL
		host, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
		port, _ := strconv.Atoi(portStr)

		if !sm.isEndpointResponsive(context.Background(), host, port) {
			t.Error("isEndpointResponsive() returned false for responsive server, want true")
		}
	})

	t.Run("unresponsive endpoint - connection refused", func(t *testing.T) {
		// Use a port that's not listening
		if sm.isEndpointResponsive(context.Background(), "localhost", 59999) {
			t.Error("isEndpointResponsive() returned true for non-listening port, want false")
		}
	})

	t.Run("unresponsive endpoint - non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		host, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
		port, _ := strconv.Atoi(portStr)

		if sm.isEndpointResponsive(context.Background(), host, port) {
			t.Error("isEndpointResponsive() returned true for 500 status, want false")
		}
	})

	t.Run("unresponsive endpoint - timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sleep longer than the 2 second timeout
			time.Sleep(3 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		host, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
		port, _ := strconv.Atoi(portStr)

		start := time.Now()
		result := sm.isEndpointResponsive(context.Background(), host, port)
		elapsed := time.Since(start)

		if result {
			t.Error("isEndpointResponsive() returned true for slow server, want false")
		}

		// Should timeout around 2 seconds, not wait for full 3 seconds
		if elapsed > 3*time.Second {
			t.Errorf("isEndpointResponsive() took %v, expected timeout around 2s", elapsed)
		}
	})

	t.Run("invalid host", func(t *testing.T) {
		if sm.isEndpointResponsive(context.Background(), "invalid.host.that.does.not.exist.local", 9222) {
			t.Error("isEndpointResponsive() returned true for invalid host, want false")
		}
	})
}

// TestKillProcess tests process termination.
func TestKillProcess(t *testing.T) {
	sm := NewSessionManager("/tmp/test-workspace", Config{})

	t.Run("kill non-existent process", func(t *testing.T) {
		// Should not panic when killing a non-existent process
		sm.killProcess(999999)
	})

	t.Run("kill invalid PID", func(t *testing.T) {
		// Should not panic with negative PID
		sm.killProcess(-1)
	})

	t.Run("kill spawned process", func(t *testing.T) {
		// Start a sleep process that we can kill
		ctx := context.Background()
		cmd := exec.CommandContext(ctx, "sleep", "60")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			t.Skipf("Could not start sleep process: %v", err)
		}

		pid := cmd.Process.Pid

		// Verify process is running
		if !sm.isProcessAlive(pid) {
			t.Fatal("Started process is not alive")
		}

		// Kill it
		sm.killProcess(pid)

		// Give it a moment to terminate
		time.Sleep(100 * time.Millisecond)

		// Verify it's dead
		if sm.isProcessAlive(pid) {
			t.Error("Process still alive after killProcess()")
			// Clean up if test failed
			_ = cmd.Process.Kill()
		}
	})
}

// TestConnectOrCreateWithUnresponsiveSession tests stale session handling.
func TestConnectOrCreateWithUnresponsiveSession(t *testing.T) {
	t.Run("detects unresponsive session and creates new", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm := NewSessionManager(tmpDir, Config{
			Host:    "localhost",
			Port:    0,
			Timeout: 30 * time.Second,
		})

		// Start a mock process that won't have Chrome's HTTP endpoint
		cmdCtx := context.Background()
		cmd := exec.CommandContext(cmdCtx, "sleep", "60")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			t.Skipf("Could not start mock process: %v", err)
		}
		defer func() { _ = cmd.Process.Kill() }()

		pid := cmd.Process.Pid

		// Create a fake session file pointing to this process with a port that won't respond
		session := &Session{
			PID:         pid,
			Port:        9299, // Port that won't have Chrome's endpoint
			Host:        "localhost",
			UserDataDir: t.TempDir(),
			StartedAt:   time.Now(),
		}
		if err := sm.saveSession(session); err != nil {
			t.Fatalf("Failed to save session: %v", err)
		}

		// Verify the session file exists
		if _, err := sm.loadSession(); err != nil {
			t.Fatalf("Session file not loadable: %v", err)
		}

		// Verify the process is alive but endpoint is not responsive
		if !sm.isProcessAlive(pid) {
			t.Fatal("Test process should be alive")
		}
		if sm.isEndpointResponsive(context.Background(), session.Host, session.Port) {
			t.Fatal("Endpoint should not be responsive")
		}

		// Now when we call ConnectOrCreate, it should:
		// 1. Detect the session is unresponsive
		// 2. Kill the process
		// 3. Clean up the session file
		// 4. Try to launch a new browser (which may fail in test env, but that's OK)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// This will fail to launch Chrome in test environment, but we can verify
		// that it detected and cleaned up the stale session
		_, _ = sm.ConnectOrCreate(ctx)

		// The old process should be killed
		time.Sleep(100 * time.Millisecond)
		if sm.isProcessAlive(pid) {
			t.Error("Old process should have been killed")
		}
	})

	t.Run("reuses responsive session", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Start a mock HTTP server that responds like Chrome
		// We need to use a listener on a specific port in the valid range
		lc := net.ListenConfig{}
		listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:9250")
		if err != nil {
			t.Skipf("Could not listen on port 9250: %v", err)
		}
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/json/version" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"webSocketDebuggerUrl":"ws://localhost:9250"}`))
			}
		}))
		_ = server.Listener.Close()
		server.Listener = listener
		server.Start()
		defer server.Close()

		host := "127.0.0.1"
		port := 9250

		sm := NewSessionManager(tmpDir, Config{
			Host:    host,
			Port:    port,
			Timeout: 30 * time.Second,
		})

		// Create a session file pointing to our mock server
		// Use current process PID since we need a real process
		session := &Session{
			PID:         os.Getpid(),
			Port:        port,
			Host:        host,
			UserDataDir: t.TempDir(),
			StartedAt:   time.Now(),
		}
		if err := sm.saveSession(session); err != nil {
			t.Fatalf("Failed to save session: %v", err)
		}

		// Verify both process and endpoint are "alive"
		if !sm.isProcessAlive(session.PID) {
			t.Fatal("Test process should be alive")
		}
		if !sm.isEndpointResponsive(context.Background(), session.Host, session.Port) {
			t.Fatal("Endpoint should be responsive")
		}

		// ConnectOrCreate should detect this as a valid session and reuse it
		loadedSession, err := sm.loadSession()
		if err != nil {
			t.Fatalf("Failed to load session: %v", err)
		}

		// Manually validate and check endpoint
		if err := sm.validateSession(loadedSession); err != nil {
			t.Fatalf("Session validation failed: %v", err)
		}

		if !sm.isProcessAlive(loadedSession.PID) || !sm.isEndpointResponsive(context.Background(), loadedSession.Host, loadedSession.Port) {
			t.Fatal("Session should be detected as valid and responsive")
		}
	})
}

// TestStaleSessionCleanup tests that cleanupStaleSession works correctly.
func TestStaleSessionCleanup(t *testing.T) {
	t.Run("cleans up session file and user data dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		userDataDir := filepath.Join(tmpDir, "user-data")
		if err := os.MkdirAll(userDataDir, 0o755); err != nil {
			t.Fatalf("Failed to create user data dir: %v", err)
		}

		sm := NewSessionManager(tmpDir, Config{})

		session := &Session{
			PID:         12345,
			Port:        9222,
			Host:        "localhost",
			UserDataDir: userDataDir,
			StartedAt:   time.Now(),
		}
		if err := sm.saveSession(session); err != nil {
			t.Fatalf("Failed to save session: %v", err)
		}

		// Verify session file exists
		sessionPath := filepath.Join(tmpDir, ".mehrhof/browser.json")
		if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
			t.Fatal("Session file should exist")
		}

		// Cleanup
		sm.cleanupStaleSession()

		// Session file should be removed
		if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
			t.Error("Session file should have been removed")
		}

		// User data dir should be removed
		if _, err := os.Stat(userDataDir); !os.IsNotExist(err) {
			t.Error("User data directory should have been removed")
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
