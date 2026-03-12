package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sedobrengocce/TaskMaster/internal/db"
)

const (
	testJWTSecret     = "test-jwt-secret"
	testRefreshSecret = "test-refresh-secret"
)

type testServer struct {
	server    *Server
	mockDB    *db.MockQuerier
	miniredis *miniredis.Miniredis
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	mockDB := &db.MockQuerier{}

	e := echo.New()
	e.Validator = NewValidator()

	srv := &Server{
		DB:            mockDB,
		echo:          e,
		JWTSecret:     []byte(testJWTSecret),
		RefreshSecret: []byte(testRefreshSecret),
		Redis:         redisClient,
	}

	return &testServer{
		server:    srv,
		mockDB:    mockDB,
		miniredis: mr,
	}
}

func newEchoContext(e *echo.Echo, method, path string, body []byte) (echo.Context, *httptest.ResponseRecorder) {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func createTestJWT(t *testing.T, secret []byte, userID int32, expiry time.Duration) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "taskmaster",
		"sub": float64(userID),
		"aud": "taskmaster_users",
		"nbf": jwt.NewNumericDate(time.Now()),
		"iat": jwt.NewNumericDate(time.Now()),
		"jti": "test-jti",
		"exp": jwt.NewNumericDate(time.Now().Add(expiry)),
	})
	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to create test JWT: %v", err)
	}
	return tokenString
}
