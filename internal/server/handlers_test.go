package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sedobrengocce/TaskMaster/internal/db"
	"github.com/sedobrengocce/TaskMaster/internal/utils"
)

// ── HealthCheck ─────────────────────────────────────────────────

func TestHealthCheckHandler(t *testing.T) {
	ts := newTestServer(t)
	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/healthcheck", nil)

	err := ts.server.HealthCheckHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
}

// ── RegisterUser ────────────────────────────────────────────────

func TestRegisterUserHandler_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("CreateUser", mock.Anything, mock.MatchedBy(func(p db.CreateUserParams) bool {
		return p.Email == "test@example.com" && p.PasswordHash != ""
	})).Return(nil)

	ts.mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(db.User{
		ID:    1,
		Email: "test@example.com",
	}, nil)

	body := []byte(`{"email":"test@example.com","password":"password123"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/register", body)

	err := ts.server.RegisterUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp UserResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, int32(1), resp.ID)
	assert.Equal(t, "test@example.com", resp.Email)
	ts.mockDB.AssertExpectations(t)
}

func TestRegisterUserHandler_InvalidJSON(t *testing.T) {
	ts := newTestServer(t)

	body := []byte(`{invalid}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/register", body)

	err := ts.server.RegisterUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterUserHandler_DBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("CreateUser", mock.Anything, mock.Anything).Return(errors.New("db connection failed"))

	body := []byte(`{"email":"test@example.com","password":"password123"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/register", body)

	err := ts.server.RegisterUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRegisterUserHandler_DuplicateEmail(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("CreateUser", mock.Anything, mock.Anything).Return(errors.New("user already exists"))

	body := []byte(`{"email":"dup@example.com","password":"password123"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/register", body)

	err := ts.server.RegisterUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

// ── LoginUser ───────────────────────────────────────────────────

func setupLoginMocks(ts *testServer, email, password string) {
	hashed, _ := utils.HashString(password)
	ts.mockDB.On("GetUserByEmail", mock.Anything, email).Return(db.User{
		ID:           1,
		Email:        email,
		PasswordHash: hashed,
	}, nil)
	ts.mockDB.On("InsertRefreshToken", mock.Anything, mock.Anything).Return(nil)
	ts.mockDB.On("GetClientByClientID", mock.Anything, mock.Anything).Return(db.GetClientByClientIDRow{}, errors.New("not found"))
}

func TestLoginUserHandler_SuccessWeb(t *testing.T) {
	ts := newTestServer(t)
	setupLoginMocks(ts, "user@example.com", "password123")

	body := []byte(`{"email":"user@example.com","password":"password123"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/login", body)

	err := ts.server.LoginUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["jwt"])
	assert.Empty(t, resp["refresh_token"], "web clients should not get refresh token in body")

	// Should have refresh_token cookie
	cookies := rec.Result().Cookies()
	var found bool
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			found = true
			assert.True(t, cookie.HttpOnly)
			break
		}
	}
	assert.True(t, found, "should set refresh_token cookie")
	ts.mockDB.AssertExpectations(t)
}

func TestLoginUserHandler_SuccessMobile(t *testing.T) {
	ts := newTestServer(t)

	hashed, _ := utils.HashString("password123")
	ts.mockDB.On("GetUserByEmail", mock.Anything, "user@example.com").Return(db.User{
		ID:           1,
		Email:        "user@example.com",
		PasswordHash: hashed,
	}, nil)
	ts.mockDB.On("InsertRefreshToken", mock.Anything, mock.Anything).Return(nil)

	clientSecretHash, _ := utils.HashString("mobile-secret")
	ts.mockDB.On("GetClientByClientID", mock.Anything, "mobile-app").Return(db.GetClientByClientIDRow{
		ClientID:         "mobile-app",
		ClientSecretHash: sql.NullString{String: clientSecretHash, Valid: true},
		ClientType:       db.ClientsClientTypeConfidential,
	}, nil)

	body := []byte(`{"email":"user@example.com","password":"password123"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/login", body)
	c.Request().Header.Set("X-Client-Type", "mobile-app")
	c.Request().Header.Set("X-Client-Secret", "mobile-secret")

	err := ts.server.LoginUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["jwt"])
	assert.NotEmpty(t, resp["refresh_token"], "mobile clients should get refresh token in body")
	ts.mockDB.AssertExpectations(t)
}

func TestLoginUserHandler_WrongCredentials(t *testing.T) {
	ts := newTestServer(t)

	hashed, _ := utils.HashString("correct_password")
	ts.mockDB.On("GetUserByEmail", mock.Anything, "user@example.com").Return(db.User{
		ID:           1,
		Email:        "user@example.com",
		PasswordHash: hashed,
	}, nil)

	body := []byte(`{"email":"user@example.com","password":"wrong_password"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/login", body)

	err := ts.server.LoginUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLoginUserHandler_UserNotFound(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetUserByEmail", mock.Anything, "nonexistent@example.com").Return(db.User{}, errors.New("not found"))

	body := []byte(`{"email":"nonexistent@example.com","password":"password123"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/login", body)

	err := ts.server.LoginUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ── LogoutUser ──────────────────────────────────────────────────

func TestLogoutUserHandler_Success(t *testing.T) {
	ts := newTestServer(t)

	token := createTestJWT(t, ts.server.JWTSecret, 1, 5*time.Minute)

	ts.mockDB.On("RevokeRefreshToken", mock.Anything, int64(1)).Return(nil)
	ts.mockDB.On("GetClientByClientID", mock.Anything, mock.Anything).Return(db.GetClientByClientIDRow{}, errors.New("not found"))

	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/logout", nil)
	c.Request().Header.Set("Authorization", "Bearer "+token)

	err := ts.server.LogoutUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Token should be in Redis denylist
	exists := ts.miniredis.Exists(token)
	assert.True(t, exists, "token should be added to Redis denylist")
	ts.mockDB.AssertExpectations(t)
}

func TestLogoutUserHandler_TokenAlreadyInDenylist(t *testing.T) {
	ts := newTestServer(t)

	token := createTestJWT(t, ts.server.JWTSecret, 1, 5*time.Minute)

	// Pre-add to denylist
	ts.miniredis.Set(token, "revoked")

	ts.mockDB.On("RevokeRefreshToken", mock.Anything, int64(1)).Return(nil)
	ts.mockDB.On("GetClientByClientID", mock.Anything, mock.Anything).Return(db.GetClientByClientIDRow{}, errors.New("not found"))

	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/logout", nil)
	c.Request().Header.Set("Authorization", "Bearer "+token)

	err := ts.server.LogoutUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ── RefreshToken ────────────────────────────────────────────────

func TestRefreshTokenHandler_SuccessWeb(t *testing.T) {
	ts := newTestServer(t)

	refreshToken := createTestJWT(t, ts.server.RefreshSecret, 1, 24*time.Hour)
	hashedRefresh := utils.HashToken(refreshToken)

	ts.mockDB.On("GetClientByClientID", mock.Anything, mock.Anything).Return(db.GetClientByClientIDRow{}, errors.New("not found"))
	ts.mockDB.On("GetRefreshToken", mock.Anything, int64(1)).Return(db.GetRefreshTokenRow{
		UserID:    1,
		TokenHash: hashedRefresh,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil)
	ts.mockDB.On("GetUserByID", mock.Anything, int32(1)).Return(db.User{
		ID:    1,
		Email: "user@example.com",
	}, nil)
	ts.mockDB.On("InsertRefreshToken", mock.Anything, mock.Anything).Return(nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/refresh_token", nil)
	c.Request().AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})

	err := ts.server.RefreshTokenHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["jwt"])
	ts.mockDB.AssertExpectations(t)
}

func TestRefreshTokenHandler_SuccessMobile(t *testing.T) {
	ts := newTestServer(t)

	refreshToken := createTestJWT(t, ts.server.RefreshSecret, 1, 24*time.Hour)
	hashedRefresh := utils.HashToken(refreshToken)

	clientSecretHash, _ := utils.HashString("mobile-secret")
	ts.mockDB.On("GetClientByClientID", mock.Anything, "mobile-app").Return(db.GetClientByClientIDRow{
		ClientID:         "mobile-app",
		ClientSecretHash: sql.NullString{String: clientSecretHash, Valid: true},
		ClientType:       db.ClientsClientTypeConfidential,
	}, nil)
	ts.mockDB.On("GetRefreshToken", mock.Anything, int64(1)).Return(db.GetRefreshTokenRow{
		UserID:    1,
		TokenHash: hashedRefresh,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil)
	ts.mockDB.On("GetUserByID", mock.Anything, int32(1)).Return(db.User{
		ID:    1,
		Email: "user@example.com",
	}, nil)
	ts.mockDB.On("InsertRefreshToken", mock.Anything, mock.Anything).Return(nil)

	body := []byte(fmt.Sprintf(`{"refresh_token":"%s"}`, refreshToken))
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/refresh_token", body)
	c.Request().Header.Set("X-Client-Type", "mobile-app")
	c.Request().Header.Set("X-Client-Secret", "mobile-secret")

	err := ts.server.RefreshTokenHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["jwt"])
	assert.NotEmpty(t, resp["refresh_token"])
	ts.mockDB.AssertExpectations(t)
}

func TestRefreshTokenHandler_ExpiredToken(t *testing.T) {
	ts := newTestServer(t)

	// Create an expired refresh token
	expiredToken := createTestJWT(t, ts.server.RefreshSecret, 1, -1*time.Hour)

	ts.mockDB.On("GetClientByClientID", mock.Anything, mock.Anything).Return(db.GetClientByClientIDRow{}, errors.New("not found"))

	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/refresh_token", nil)
	c.Request().AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: expiredToken,
	})

	err := ts.server.RefreshTokenHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRefreshTokenHandler_MissingCookie(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetClientByClientID", mock.Anything, mock.Anything).Return(db.GetClientByClientIDRow{}, errors.New("not found"))

	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/refresh_token", nil)

	err := ts.server.RefreshTokenHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ── CreateProject ───────────────────────────────────────────────

func TestCreateProject_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("CreateProject", mock.Anything, db.CreateProjectParams{
		Name:     "My Project",
		UserID:   1,
		ColorHex: sql.NullString{},
	}).Return(nil)

	body := []byte(`{"name":"My Project","user_id":1}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/projects", body)

	err := ts.server.CreateProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestCreateProject_DBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("CreateProject", mock.Anything, mock.Anything).Return(errors.New("db error"))

	body := []byte(`{"name":"My Project","user_id":1}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/projects", body)

	err := ts.server.CreateProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── ListProjects ────────────────────────────────────────────────

func TestListProjects_Success(t *testing.T) {
	ts := newTestServer(t)

	projects := []db.Project{
		{ID: 1, UserID: 1, Name: "Project 1"},
		{ID: 2, UserID: 1, Name: "Project 2"},
	}
	ts.mockDB.On("GetProjectsByUserId", mock.Anything, db.GetProjectsByUserIdParams{UserID: 1}).Return(projects, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects?user_id=1", nil)

	err := ts.server.ListProjects(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []db.Project
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
	ts.mockDB.AssertExpectations(t)
}

func TestListProjects_InvalidUserID(t *testing.T) {
	ts := newTestServer(t)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects?user_id=abc", nil)

	err := ts.server.ListProjects(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ── GetProject ──────────────────────────────────────────────────

func TestGetProject_Success(t *testing.T) {
	ts := newTestServer(t)

	project := db.Project{ID: 1, UserID: 1, Name: "My Project"}
	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(project, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects/1", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ts.server.GetProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestGetProject_InvalidID(t *testing.T) {
	ts := newTestServer(t)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects/abc", nil)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := ts.server.GetProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetProject_NotFound(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(999)).Return(db.Project{}, errors.New("not found"))

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects/999", nil)
	c.SetParamNames("id")
	c.SetParamValues("999")

	err := ts.server.GetProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── UpdateProject ───────────────────────────────────────────────

func TestUpdateProject_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("UpdateProject", mock.Anything, db.UpdateProjectParams{
		ID:   1,
		Name: "Updated Project",
	}).Return(nil)

	body := []byte(`{"name":"Updated Project"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPut, "/api/projects/1", body)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ts.server.UpdateProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

// ── DeleteProject ───────────────────────────────────────────────

func TestDeleteProject_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("DeleteProject", mock.Anything, int32(1)).Return(nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/projects/1", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ts.server.DeleteProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestDeleteProject_InvalidID(t *testing.T) {
	ts := newTestServer(t)

	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/projects/abc", nil)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := ts.server.DeleteProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ── ShareProject ────────────────────────────────────────────────

func TestSharedProjectHandler_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("ShareProjectWithUser", mock.Anything, db.ShareProjectWithUserParams{
		ProjectID:        1,
		SharedWithUserID: 2,
	}).Return(nil)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/projects/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ts.server.SharedProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestSharedProjectHandler_DBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("ShareProjectWithUser", mock.Anything, mock.Anything).Return(errors.New("db error"))

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/projects/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ts.server.SharedProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── UnshareProject ──────────────────────────────────────────────

func TestUnshareProjectHandler_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("UnshareProjectWithUser", mock.Anything, db.UnshareProjectWithUserParams{
		ProjectID:        1,
		SharedWithUserID: 2,
	}).Return(nil)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/projects/1/unshare", body)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ts.server.UnshareProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

// ── checkClientType ─────────────────────────────────────────────

func TestCheckClientType_DefaultsToWeb(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetClientByClientID", mock.Anything, "").Return(db.GetClientByClientIDRow{}, errors.New("not found"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := ts.server.echo.NewContext(req, rec)

	clientType := ts.server.checkClientType(c)
	assert.Equal(t, ClientTypeWeb, clientType)
}

// ── parseToken ──────────────────────────────────────────────────

func TestParseToken_ValidToken(t *testing.T) {
	ts := newTestServer(t)
	tokenString := createTestJWT(t, ts.server.RefreshSecret, 1, 24*time.Hour)

	token, err := ts.server.parseToken(tokenString)
	require.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, float64(1), claims["sub"])
}

func TestParseToken_InvalidToken(t *testing.T) {
	ts := newTestServer(t)

	_, err := ts.server.parseToken("invalid-token")
	assert.Error(t, err)
}
