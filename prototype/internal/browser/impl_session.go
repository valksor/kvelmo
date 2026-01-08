//go:build !no_browser
// +build !no_browser

package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	// Session file stores the active browser session info.
	sessionFile = ".mehrhof/browser.json"
	// Default port range for random port allocation.
	minPort = 9200
	maxPort = 9300
	// Maximum session age before considering it stale (24 hours).
	maxSessionAge = 24 * time.Hour
)

// Session represents an isolated browser instance.
type Session struct {
	PID         int       `json:"pid"`
	Port        int       `json:"port"`
	Host        string    `json:"host"`
	UserDataDir string    `json:"user_data_dir"`
	StartedAt   time.Time `json:"started_at"`
}

// SessionManager handles lifecycle of isolated browser sessions.
type SessionManager struct {
	workspaceDir string
	session      *Session
	config       Config
	mu           sync.RWMutex
}

// NewSessionManager creates a new session manager.
func NewSessionManager(workspaceDir string, config Config) *SessionManager {
	return &SessionManager{
		workspaceDir: workspaceDir,
		config:       config,
	}
}

// ConnectOrCreate tries to connect to existing session, or launches a new isolated browser.
func (sm *SessionManager) ConnectOrCreate(ctx context.Context) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// First, try to load existing session
	session, err := sm.loadSession()
	if err == nil && session != nil {
		// Validate the session
		if err := sm.validateSession(session); err != nil {
			slog.Warn("invalid session, will create new", "error", err)
			sm.cleanupStaleSession()

			return sm.launchBrowserUnlocked(ctx)
		}

		// Verify the session is still alive
		if sm.isProcessAlive(session.PID) {
			slog.Info("reusing existing browser session", "pid", session.PID, "port", session.Port)
			sm.session = session

			return session, nil
		}

		// Session file exists but process is dead, clean it up
		slog.Debug("browser session process not found, will create new", "pid", session.PID)
		sm.cleanupStaleSession()
	}

	// No existing session, create a new one
	return sm.launchBrowserUnlocked(ctx)
}

// validateSession checks if a session is valid and not stale.
func (sm *SessionManager) validateSession(session *Session) error {
	// Check required fields
	if session.PID == 0 {
		return errors.New("invalid PID")
	}
	if session.Port < minPort || session.Port > maxPort {
		return fmt.Errorf("invalid port: %d", session.Port)
	}
	if session.Host == "" {
		return errors.New("empty host")
	}

	// Check session age
	if time.Since(session.StartedAt) > maxSessionAge {
		return fmt.Errorf("session too old: %v", time.Since(session.StartedAt))
	}

	// Verify user data directory exists
	if session.UserDataDir != "" {
		if _, err := os.Stat(session.UserDataDir); os.IsNotExist(err) {
			return fmt.Errorf("user data directory missing: %s", session.UserDataDir)
		}
	}

	return nil
}

// cleanupStaleSession removes stale session files and directories.
func (sm *SessionManager) cleanupStaleSession() {
	sessionPath := sm.sessionPath()

	// Load session to get user data dir
	if data, err := os.ReadFile(sessionPath); err == nil {
		var session Session
		if json.Unmarshal(data, &session) == nil {
			// Remove user data directory
			if session.UserDataDir != "" {
				_ = os.RemoveAll(session.UserDataDir)
			}
		}
	}

	// Remove session file
	_ = os.Remove(sessionPath)
}

// Cleanup terminates the isolated browser if we launched it.
func (sm *SessionManager) Cleanup() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.session == nil {
		return nil
	}

	slog.Info("cleaning up browser session", "pid", sm.session.PID)

	// Kill the browser process group
	if sm.isProcessAlive(sm.session.PID) {
		// Find the process for proper cleanup
		process, err := os.FindProcess(sm.session.PID)
		if err != nil {
			slog.Warn("failed to find browser process", "error", err)
		} else {
			// Try graceful shutdown first
			if err := syscall.Kill(-sm.session.PID, syscall.SIGTERM); err != nil {
				slog.Warn("SIGTERM failed", "error", err)
			}

			// Wait for graceful shutdown with timeout
			done := make(chan struct{}, 1)
			go func() {
				_, _ = process.Wait() // Reap zombie, ignore result
				close(done)
			}()

			select {
			case <-done:
				// Clean exit
				slog.Debug("browser process terminated gracefully")
			case <-time.After(2 * time.Second):
				// Force kill after timeout
				if err := syscall.Kill(-sm.session.PID, syscall.SIGKILL); err != nil {
					slog.Debug("browser process already terminated", "error", err)
				}

				// Wait for process to exit, but with another timeout
				select {
				case <-done:
					slog.Debug("browser process terminated after SIGKILL")
				case <-time.After(1 * time.Second):
					// Process might be an unkillable zombie
					slog.Warn("browser process may be zombie (unable to reap)")
					// Don't block - state is consistent anyway
				}
			}
		}
	}

	// Clean up user data directory
	if sm.session.UserDataDir != "" {
		if err := os.RemoveAll(sm.session.UserDataDir); err != nil {
			slog.Warn("failed to remove user data directory", "error", err)
		}
	}

	// Remove session file
	if err := os.Remove(sm.sessionPath()); err != nil && !os.IsNotExist(err) {
		slog.Warn("failed to remove session file", "error", err)
	}

	sm.session = nil

	return nil
}

// GetSession returns the current session.
func (sm *SessionManager) GetSession() *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.session
}

// launchBrowserUnlocked creates a new isolated browser instance.
// Caller must hold sm.mu.
func (sm *SessionManager) launchBrowserUnlocked(ctx context.Context) (*Session, error) {
	// Try multiple times if port conflicts occur
	maxRetries := 5
	var lastErr error

	for attempt := range maxRetries {
		port := sm.config.Port
		if port == 0 {
			// Find an available port instead of random allocation
			port = sm.findAvailablePort(ctx)
		}

		session, err := sm.tryLaunchBrowser(ctx, port)
		if err == nil {
			return session, nil
		}

		lastErr = err
		slog.Debug("browser launch attempt failed", "attempt", attempt+1, "error", err)

		// If port is fixed by config, don't retry
		if sm.config.Port != 0 {
			break
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	return nil, lastErr
}

// findAvailablePort finds an available port in the configured range.
// Note: This is a best-effort check. The actual port availability is confirmed
// when Chrome tries to bind to it. Port conflicts are handled by retry logic.
func (sm *SessionManager) findAvailablePort(ctx context.Context) int {
	//nolint:gosec // G404 - Port allocation doesn't need cryptographic randomness
	for range 50 {
		port := minPort + rand.Intn(maxPort-minPort)

		listener := net.ListenConfig{}
		ln, err := listener.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			// Port is available (TOCTOU: race possible between Close() and Chrome binding)
			_ = ln.Close()

			return port
		}
	}

	// Fallback to random port if all checks fail
	// Retry logic in launchBrowserUnlocked will handle conflicts
	//nolint:gosec // G404 - Port allocation doesn't need cryptographic randomness
	return minPort + rand.Intn(maxPort-minPort)
}

// tryLaunchBrowser attempts to launch Chrome on a specific port.
func (sm *SessionManager) tryLaunchBrowser(ctx context.Context, port int) (*Session, error) {
	// Create temporary user data directory
	userDataDir, err := os.MkdirTemp("", "mehr-browser-*")
	if err != nil {
		return nil, errLaunch(fmt.Errorf("create user data dir: %w", err))
	}

	// Build Chrome command
	args := []string{
		"--remote-debugging-port=" + strconv.Itoa(port),
		"--no-first-run",
		"--no-default-browser-check",
		"--user-data-dir=" + userDataDir,
	}

	// Add certificate handling flags (default: ignore for local dev)
	// --ignore-certificate-errors: Bypass SSL certificate validation
	// --test-type: Suppress Chrome's unsupported flag warning
	if sm.config.IgnoreCertErrors {
		args = append(args, "--ignore-certificate-errors")
		args = append(args, "--test-type")
	}

	if sm.config.Headless {
		args = append(args,
			"--headless",
			"--disable-gpu",
			"--no-sandbox",
		)
	}

	// Try to find Chrome executable
	chromePath, err := findChrome()
	if err != nil {
		_ = os.RemoveAll(userDataDir)

		return nil, errLaunch(err)
	}

	slog.Info("launching isolated browser", "path", chromePath, "port", port, "headless", sm.config.Headless)

	cmd := exec.CommandContext(ctx, chromePath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(userDataDir)

		return nil, errLaunch(fmt.Errorf("start chrome: %w", err))
	}

	// Wait a moment for Chrome to start
	select {
	case <-ctx.Done():
		// Kill entire process group, not just parent process
		if cmd.Process != nil && cmd.Process.Pid > 0 {
			// Negative PID kills entire process group
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
			time.Sleep(100 * time.Millisecond)
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		_ = os.RemoveAll(userDataDir)

		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
	}

	// Verify Chrome is running (ProcessState is nil for running processes)
	if cmd.ProcessState == nil {
		session := &Session{
			PID:         cmd.Process.Pid,
			Port:        port,
			Host:        sm.config.Host,
			UserDataDir: userDataDir,
			StartedAt:   time.Now(),
		}

		// Save session
		if err := sm.saveSession(session); err != nil {
			slog.Warn("failed to save session", "error", err)
		}

		sm.session = session

		return session, nil
	}

	_ = os.RemoveAll(userDataDir)

	return nil, errLaunch(errors.New("chrome exited immediately"))
}

// loadSession loads existing session from disk.
func (sm *SessionManager) loadSession() (*Session, error) {
	path := sm.sessionPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		// JSON is corrupted, back up the file and remove it
		slog.Warn("corrupted session file, backing up and removing", "error", err)
		backupPath := path + ".corrupted." + strconv.FormatInt(time.Now().Unix(), 10)
		_ = os.Rename(path, backupPath)

		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return &session, nil
}

// saveSession saves session to disk.
func (sm *SessionManager) saveSession(session *Session) error {
	// Ensure directory exists
	sessionDir := filepath.Dir(sm.sessionPath())
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	// Use restricted permissions (0600) - session contains PID/port info
	if err := os.WriteFile(sm.sessionPath(), data, 0o600); err != nil {
		return fmt.Errorf("write session: %w", err)
	}

	return nil
}

// sessionPath returns the path to the session file.
func (sm *SessionManager) sessionPath() string {
	return filepath.Join(sm.workspaceDir, sessionFile)
}

// isProcessAlive checks if a process with the given PID is running.
func (sm *SessionManager) isProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))

	return err == nil
}

// findChrome locates the Chrome executable on the system.
func findChrome() (string, error) {
	// Try PATH first (covers most Linux distributions)
	executables := []string{
		"google-chrome",
		"google-chrome-stable",
		"google-chrome-beta",
		"chromium",
		"chromium-browser",
	}
	for _, exe := range executables {
		if path, err := exec.LookPath(exe); err == nil {
			return path, nil
		}
	}

	// Fallback to paths NOT in PATH
	paths := []string{
		"/opt/hostedtoolcache/setup-chrome/chrome/stable/x64/chrome", // GitHub Actions setup-chrome
		"/snap/bin/chromium", // Snap installs (often not in PATH)
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",                      // macOS .app bundle
		"/Applications/Chromium.app/Contents/MacOS/Chromium",                                // macOS .app bundle
		"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",                        // Windows
		"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",                  // Windows x86
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Google\\Chrome\\Application\\chrome.exe"), // Windows user install
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("chrome not found (tried PATH: %v, plus known locations)", executables)
}
