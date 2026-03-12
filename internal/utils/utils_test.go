package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashString(t *testing.T) {
	hash, err := HashString("password123")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "password123", hash)

	// Different calls produce different hashes (bcrypt salt)
	hash2, err := HashString("password123")
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash2)
}

func TestCheckStringHash(t *testing.T) {
	password := "mysecretpassword"
	hash, err := HashString(password)
	require.NoError(t, err)

	assert.True(t, CheckStringHash(password, hash))
	assert.False(t, CheckStringHash("wrongpassword", hash))
	assert.False(t, CheckStringHash("", hash))
}

func TestHashToken(t *testing.T) {
	hash := HashToken("my-long-token-string")
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA-256 hex = 64 chars

	// Same input produces same hash (deterministic)
	hash2 := HashToken("my-long-token-string")
	assert.Equal(t, hash, hash2)

	// Different input produces different hash
	hash3 := HashToken("different-token")
	assert.NotEqual(t, hash, hash3)
}

func TestCheckTokenHash(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test-payload.signature"
	hash := HashToken(token)

	assert.True(t, CheckTokenHash(token, hash))
	assert.False(t, CheckTokenHash("wrong-token", hash))
	assert.False(t, CheckTokenHash("", hash))
}

func TestGenerateRandomString(t *testing.T) {
	s, err := GenerateRandomString(32)
	require.NoError(t, err)
	assert.Len(t, s, 32)

	// Two random strings should differ
	s2, err := GenerateRandomString(32)
	require.NoError(t, err)
	assert.NotEqual(t, s, s2)

	// Length 0
	s3, err := GenerateRandomString(0)
	require.NoError(t, err)
	assert.Len(t, s3, 0)
}
