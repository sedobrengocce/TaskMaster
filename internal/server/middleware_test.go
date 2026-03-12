package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	ts := newTestServer(t)
	token := createTestJWT(t, ts.server.JWTSecret, 1, 5*time.Minute)

	mw := AuthMiddleware(ts.server.Redis, testJWTSecret)

	handlerCalled := false
	handler := func(c echo.Context) error {
		handlerCalled = true
		return c.String(http.StatusOK, "ok")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := ts.server.echo.NewContext(req, rec)

	err := mw(handler)(c)
	require.NoError(t, err)
	assert.True(t, handlerCalled, "handler should be called for valid token")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	ts := newTestServer(t)

	mw := AuthMiddleware(ts.server.Redis, testJWTSecret)

	handler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	rec := httptest.NewRecorder()
	c := ts.server.echo.NewContext(req, rec)

	err := mw(handler)(c)
	require.NoError(t, err)
	assert.Equal(t, 401, rec.Code)
}

func TestAuthMiddleware_TokenInDenylist(t *testing.T) {
	ts := newTestServer(t)
	token := createTestJWT(t, ts.server.JWTSecret, 1, 5*time.Minute)

	// Add token to Redis denylist
	ts.miniredis.Set(token, "revoked")

	mw := AuthMiddleware(ts.server.Redis, testJWTSecret)

	handler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := ts.server.echo.NewContext(req, rec)

	err := mw(handler)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_InvalidJWT(t *testing.T) {
	ts := newTestServer(t)

	mw := AuthMiddleware(ts.server.Redis, testJWTSecret)

	handler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()
	c := ts.server.echo.NewContext(req, rec)

	err := mw(handler)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	ts := newTestServer(t)

	// Create an expired token
	token := createTestJWT(t, ts.server.JWTSecret, 1, -1*time.Hour)

	mw := AuthMiddleware(ts.server.Redis, testJWTSecret)

	handler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := ts.server.echo.NewContext(req, rec)

	err := mw(handler)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
