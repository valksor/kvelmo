package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthStore_AddUser(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	// Add a user
	err = store.AddUser("admin", "password123")
	require.NoError(t, err)

	// Verify user exists
	assert.Equal(t, 1, store.Count())
	users := store.ListUsers()
	assert.Contains(t, users, "admin")

	// Try to add same user again
	err = store.AddUser("admin", "different")
	assert.ErrorIs(t, err, ErrUserExists)
}

func TestAuthStore_RemoveUser(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	// Add and remove user
	require.NoError(t, store.AddUser("admin", "pass"))
	assert.Equal(t, 1, store.Count())

	removed := store.RemoveUser("admin")
	assert.True(t, removed)
	assert.Equal(t, 0, store.Count())

	// Try to remove non-existent user
	removed = store.RemoveUser("admin")
	assert.False(t, removed)
}

func TestAuthStore_ValidatePassword(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	require.NoError(t, store.AddUser("admin", "secret123"))

	// Correct password
	assert.True(t, store.ValidatePassword("admin", "secret123"))

	// Wrong password
	assert.False(t, store.ValidatePassword("admin", "wrongpass"))

	// Non-existent user
	assert.False(t, store.ValidatePassword("nobody", "secret123"))
}

func TestAuthStore_UpdatePassword(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	require.NoError(t, store.AddUser("admin", "oldpass"))
	assert.True(t, store.ValidatePassword("admin", "oldpass"))

	// Update password
	err = store.UpdatePassword("admin", "newpass")
	require.NoError(t, err)

	// Old password should fail
	assert.False(t, store.ValidatePassword("admin", "oldpass"))
	// New password should work
	assert.True(t, store.ValidatePassword("admin", "newpass"))

	// Update non-existent user
	err = store.UpdatePassword("nobody", "pass")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestAuthStore_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	// Create and save store
	store1, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	require.NoError(t, store1.AddUser("admin", "pass"))
	require.NoError(t, store1.AddUser("developer", "devpass"))
	require.NoError(t, store1.Save())

	// Load store fresh
	store2, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	assert.Equal(t, 2, store2.Count())
	assert.True(t, store2.ValidatePassword("admin", "pass"))
	assert.True(t, store2.ValidatePassword("developer", "devpass"))
}

func TestAuthStore_GetUser(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	require.NoError(t, store.AddUser("admin", "pass"))

	// Get existing user
	user, exists := store.GetUser("admin")
	assert.True(t, exists)
	assert.Equal(t, "admin", user.Username)
	assert.NotEmpty(t, user.PasswordHash)
	assert.False(t, user.CreatedAt.IsZero())

	// Get non-existent user
	user, exists = store.GetUser("nobody")
	assert.False(t, exists)
	assert.Nil(t, user)
}

func TestAuthStore_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	// Create empty file
	require.NoError(t, os.WriteFile(path, []byte(""), 0o600))

	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)
	assert.Equal(t, 0, store.Count())
}

func TestGenerateSessionToken(t *testing.T) {
	token1, err := GenerateSessionToken()
	require.NoError(t, err)
	assert.Len(t, token1, 64) // 32 bytes = 64 hex chars

	token2, err := GenerateSessionToken()
	require.NoError(t, err)

	// Tokens should be unique
	assert.NotEqual(t, token1, token2)
}
