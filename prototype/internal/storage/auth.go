// Package storage provides auth store for web UI authentication.
package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// Role represents a user's access level.
type Role string

const (
	// RoleUser represents a user with full access.
	RoleUser Role = "user"
	// RoleViewer represents a user with read-only access.
	RoleViewer Role = "viewer"
)

// ValidRole checks if a role string is valid.
func ValidRole(role string) bool {
	switch Role(role) {
	case RoleUser, RoleViewer:
		return true
	}

	return false
}

// AuthStore manages user credentials for web UI authentication.
type AuthStore struct {
	Version string           `yaml:"version"`
	Users   map[string]*User `yaml:"users"`

	mu   sync.RWMutex
	path string
}

// User represents a user credential.
type User struct {
	Username     string    `yaml:"username"`
	PasswordHash string    `yaml:"password_hash"`
	Role         Role      `yaml:"role,omitempty"`
	CreatedAt    time.Time `yaml:"created_at"`
}

// UnmarshalYAML implements custom YAML unmarshaling with backward compatibility.
// If role is not set, it defaults to RoleUser.
func (u *User) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Define an alias type to avoid infinite recursion
	type userAlias User
	alias := userAlias{Role: RoleUser} // Default to RoleUser

	if err := unmarshal(&alias); err != nil {
		return err
	}

	// Validate role if set
	if alias.Role != "" && !ValidRole(string(alias.Role)) {
		return fmt.Errorf("invalid role: %s", alias.Role)
	}

	*u = User(alias)

	return nil
}

// AuthSession represents an active user session for web UI authentication.
type AuthSession struct {
	Token     string
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// ErrUserExists is returned when trying to add a user that already exists.
var ErrUserExists = errors.New("user already exists")

// ErrUserNotFound is returned when a user is not found.
var ErrUserNotFound = errors.New("user not found")

// LoadAuthStore loads the auth store from the default location.
// Creates an empty store if the file doesn't exist.
func LoadAuthStore() (*AuthStore, error) {
	path, err := authStorePath()
	if err != nil {
		return nil, err
	}

	return LoadAuthStoreFromPath(path)
}

// LoadAuthStoreFromPath loads the auth store from a specific path.
func LoadAuthStoreFromPath(path string) (*AuthStore, error) {
	store := &AuthStore{
		Version: "1",
		Users:   make(map[string]*User),
		path:    path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}

		return nil, fmt.Errorf("read auth store: %w", err)
	}

	if err := yaml.Unmarshal(data, store); err != nil {
		return nil, fmt.Errorf("parse auth store: %w", err)
	}

	if store.Users == nil {
		store.Users = make(map[string]*User)
	}

	return store, nil
}

// Save persists the auth store to disk.
func (a *AuthStore) Save() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	data, err := yaml.Marshal(a)
	if err != nil {
		return fmt.Errorf("marshal auth store: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(a.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write atomically
	tmpPath := a.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write auth store: %w", err)
	}

	if err := os.Rename(tmpPath, a.path); err != nil {
		return fmt.Errorf("rename auth store: %w", err)
	}

	return nil
}

// AddUser adds a new user with the given password and role.
// Role defaults to RoleUser if empty.
// Returns ErrUserExists if the user already exists.
func (a *AuthStore) AddUser(username, password string, role Role) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.Users[username]; exists {
		return ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Default to RoleUser if not specified
	if role == "" {
		role = RoleUser
	}

	a.Users[username] = &User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    time.Now(),
	}

	return nil
}

// RemoveUser removes a user from the store.
// Returns false if the user doesn't exist.
func (a *AuthStore) RemoveUser(username string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.Users[username]; !exists {
		return false
	}

	delete(a.Users, username)

	return true
}

// UpdatePassword updates a user's password.
// Returns ErrUserNotFound if the user doesn't exist.
func (a *AuthStore) UpdatePassword(username, newPassword string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	user, exists := a.Users[username]
	if !exists {
		return ErrUserNotFound
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.PasswordHash = string(hash)

	return nil
}

// SetRole updates a user's role.
// Returns ErrUserNotFound if the user doesn't exist.
// Returns an error if the role is invalid.
func (a *AuthStore) SetRole(username string, role Role) error {
	if !ValidRole(string(role)) {
		return fmt.Errorf("invalid role: %s", role)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	user, exists := a.Users[username]
	if !exists {
		return ErrUserNotFound
	}

	user.Role = role

	return nil
}

// ValidatePassword checks if the password is correct for the user.
func (a *AuthStore) ValidatePassword(username, password string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	user, exists := a.Users[username]
	if !exists {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))

	return err == nil
}

// ListUsers returns a list of all usernames.
func (a *AuthStore) ListUsers() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	users := make([]string, 0, len(a.Users))
	for username := range a.Users {
		users = append(users, username)
	}

	return users
}

// ListUsersDetails returns a list of all users with their details.
func (a *AuthStore) ListUsersDetails() []User {
	a.mu.RLock()
	defer a.mu.RUnlock()

	users := make([]User, 0, len(a.Users))
	for _, user := range a.Users {
		users = append(users, *user)
	}

	return users
}

// Count returns the number of registered users.
func (a *AuthStore) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return len(a.Users)
}

// GetUser returns a user by username.
func (a *AuthStore) GetUser(username string) (*User, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	user, exists := a.Users[username]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent mutation
	userCopy := *user

	return &userCopy, true
}

// GenerateSessionToken creates a cryptographically secure session token.
func GenerateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

// authStorePath returns the default path for the auth store.
func authStorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	return filepath.Join(home, ".valksor", "mehrhof", "auth.yaml"), nil
}
