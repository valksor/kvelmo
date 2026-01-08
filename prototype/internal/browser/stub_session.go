//go:build no_browser
// +build no_browser

package browser

import (
	"context"
	"sync"
)

// Session represents an isolated browser instance (stub).
type Session struct {
	PID         int
	Port        int
	Host        string
	UserDataDir string
}

// SessionManager handles lifecycle of isolated browser sessions (stub).
type SessionManager struct {
	workspaceDir string
	session      *Session
	config       Config
	mu           sync.RWMutex
}

// NewSessionManager creates a stub session manager.
func NewSessionManager(workspaceDir string, config Config) *SessionManager {
	return &SessionManager{
		workspaceDir: workspaceDir,
		config:       config,
	}
}

// ConnectOrCreate returns an error - browser is disabled.
func (sm *SessionManager) ConnectOrCreate(ctx context.Context) (*Session, error) {
	return nil, ErrDisabled
}

// Cleanup is a no-op.
func (sm *SessionManager) Cleanup() error {
	return nil
}

// GetSession returns nil - no session exists.
func (sm *SessionManager) GetSession() *Session {
	return nil
}
