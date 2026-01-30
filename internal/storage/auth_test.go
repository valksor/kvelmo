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
	err = store.AddUser("admin", "password123", RoleUser)
	require.NoError(t, err)

	// Verify user exists
	assert.Equal(t, 1, store.Count())
	users := store.ListUsers()
	assert.Contains(t, users, "admin")

	// Try to add same user again
	err = store.AddUser("admin", "different", RoleUser)
	assert.ErrorIs(t, err, ErrUserExists)
}

func TestAuthStore_RemoveUser(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	// Add and remove user
	require.NoError(t, store.AddUser("admin", "pass", RoleUser))
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

	require.NoError(t, store.AddUser("admin", "secret123", RoleUser))

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

	require.NoError(t, store.AddUser("admin", "oldpass", RoleUser))
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

	require.NoError(t, store1.AddUser("admin", "pass", RoleUser))
	require.NoError(t, store1.AddUser("developer", "devpass", RoleUser))
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

	require.NoError(t, store.AddUser("admin", "pass", RoleUser))

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

func TestAuthStore_AddUserWithRole(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	// Add user with default role
	err = store.AddUser("user1", "pass1", RoleUser)
	require.NoError(t, err)

	// Add viewer
	err = store.AddUser("viewer1", "pass2", RoleViewer)
	require.NoError(t, err)

	// Verify roles
	users := store.ListUsersDetails()
	assert.Len(t, users, 2)

	// Find user1 and viewer1
	var user1, viewer1 *User
	for i := range users {
		if users[i].Username == "user1" {
			user1 = &users[i]
		}
		if users[i].Username == "viewer1" {
			viewer1 = &users[i]
		}
	}

	require.NotNil(t, user1)
	assert.Equal(t, RoleUser, user1.Role)

	require.NotNil(t, viewer1)
	assert.Equal(t, RoleViewer, viewer1.Role)
}

func TestAuthStore_SetRole(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	// Add user as viewer
	err = store.AddUser("stakeholder", "pass", RoleViewer)
	require.NoError(t, err)

	user, exists := store.GetUser("stakeholder")
	require.True(t, exists)
	assert.Equal(t, RoleViewer, user.Role)

	// Change to user role
	err = store.SetRole("stakeholder", RoleUser)
	require.NoError(t, err)

	user, _ = store.GetUser("stakeholder")
	assert.Equal(t, RoleUser, user.Role)

	// Change non-existent user
	err = store.SetRole("nobody", RoleViewer)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestAuthStore_SetRoleInvalidRole(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	require.NoError(t, store.AddUser("user1", "pass", RoleUser))

	// Try to set invalid role
	err = store.SetRole("user1", "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")
}

func TestAuthStore_ListUsersDetails(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	// Add users with different roles
	require.NoError(t, store.AddUser("admin", "adminpass", RoleUser))
	require.NoError(t, store.AddUser("viewer", "viewerpass", RoleViewer))
	require.NoError(t, store.AddUser("user2", "pass2", RoleUser))

	details := store.ListUsersDetails()
	assert.Len(t, details, 3)

	// Verify all users are returned with details
	usernames := make(map[string]*User)
	for i := range details {
		usernames[details[i].Username] = &details[i]
	}

	assert.Contains(t, usernames, "admin")
	assert.Equal(t, RoleUser, usernames["admin"].Role)
	assert.NotZero(t, usernames["admin"].CreatedAt)

	assert.Contains(t, usernames, "viewer")
	assert.Equal(t, RoleViewer, usernames["viewer"].Role)

	assert.Contains(t, usernames, "user2")
	assert.Equal(t, RoleUser, usernames["user2"].Role)
}

func TestAuthStore_BackwardCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "auth.yaml")

	// Create auth file without role field (old format)
	oldContent := `
users:
  admin:
    password_hash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
    created_at: "2024-01-01T00:00:00Z"
`
	require.NoError(t, os.WriteFile(path, []byte(oldContent), 0o600))

	// Load should work and default to RoleUser
	store, err := LoadAuthStoreFromPath(path)
	require.NoError(t, err)

	user, exists := store.GetUser("admin")
	require.True(t, exists)
	assert.Equal(t, RoleUser, user.Role, "users without role should default to RoleUser")
}

func TestValidRole(t *testing.T) {
	// Valid roles
	assert.True(t, ValidRole("user"))
	assert.True(t, ValidRole("viewer"))

	// Invalid roles
	assert.False(t, ValidRole(""))
	assert.False(t, ValidRole("admin"))
	assert.False(t, ValidRole("root"))
	assert.False(t, ValidRole("User")) // case sensitive
}
