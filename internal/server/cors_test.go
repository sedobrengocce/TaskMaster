package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORSOriginsFromString_Empty(t *testing.T) {
	origins := parseCORSOrigins("")
	assert.Equal(t, []string{"*"}, origins)
}

func TestCORSOriginsFromString_Single(t *testing.T) {
	origins := parseCORSOrigins("https://example.com")
	assert.Equal(t, []string{"https://example.com"}, origins)
}

func TestCORSOriginsFromString_Multiple(t *testing.T) {
	origins := parseCORSOrigins("https://a.com,https://b.com")
	assert.Equal(t, []string{"https://a.com", "https://b.com"}, origins)
}
