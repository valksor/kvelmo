package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateToken(t *testing.T) {
	token1, err := generateToken()
	require.NoError(t, err)
	assert.Len(t, token1, 64) // 32 bytes = 64 hex chars

	token2, err := generateToken()
	require.NoError(t, err)

	// Tokens should be unique
	assert.NotEqual(t, token1, token2)
}
