package access

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStore_CreateAndValidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	store := New(path)

	plaintext, err := store.Create(RoleOperator, "test-token", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if plaintext == "" {
		t.Fatal("expected non-empty plaintext")
	}

	token, err := store.Validate(plaintext)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	if token.Role != RoleOperator {
		t.Errorf("expected role operator, got %q", token.Role)
	}

	if token.Label != "test-token" {
		t.Errorf("expected label 'test-token', got %q", token.Label)
	}

	// Invalid token should fail.
	_, err = store.Validate("invalid-token-value")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestStore_Revoke(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	store := New(path)

	plaintext, err := store.Create(RoleViewer, "revoke-me", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Get the token ID.
	token, err := store.Validate(plaintext)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	if err := store.Revoke(token.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	// Token should no longer validate.
	_, err = store.Validate(plaintext)
	if err == nil {
		t.Error("expected error after revoke")
	}
}

func TestStore_List(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	store := New(path)

	_, err := store.Create(RoleOperator, "token-1", nil)
	if err != nil {
		t.Fatalf("create 1: %v", err)
	}

	_, err = store.Create(RoleViewer, "token-2", nil)
	if err != nil {
		t.Fatalf("create 2: %v", err)
	}

	tokens, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}

	// Hash should be cleared in list output.
	for _, tok := range tokens {
		if tok.Hash != "" {
			t.Errorf("expected hash to be cleared for token %q", tok.ID)
		}
	}
}

func TestStore_Expiry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	store := New(path)

	// Create a token that already expired.
	expired := time.Now().Add(-1 * time.Hour)

	plaintext, err := store.Create(RoleOperator, "expired-token", &expired)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err = store.Validate(plaintext)
	if err == nil {
		t.Error("expected error for expired token")
	}

	// Create a token with future expiry.
	future := time.Now().Add(24 * time.Hour)

	plaintext, err = store.Create(RoleOperator, "valid-token", &future)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	token, err := store.Validate(plaintext)
	if err != nil {
		t.Fatalf("validate future token: %v", err)
	}

	if token.Label != "valid-token" {
		t.Errorf("expected label 'valid-token', got %q", token.Label)
	}
}
