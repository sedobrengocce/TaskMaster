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
	ts.mockDB.On("IsProjectSharedWithUser", mock.Anything, mock.Anything).Maybe().Return(false, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects/1", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

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
	setAuthUser(c, 1)

	err := ts.server.GetProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── UpdateProject ───────────────────────────────────────────────

func TestUpdateProject_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1, Name: "Old"}, nil)
	ts.mockDB.On("UpdateProject", mock.Anything, db.UpdateProjectParams{
		ID:   1,
		Name: "Updated Project",
	}).Return(nil)

	body := []byte(`{"name":"Updated Project"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPut, "/api/projects/1", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.UpdateProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

// ── DeleteProject ───────────────────────────────────────────────

func TestDeleteProject_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)
	ts.mockDB.On("DeleteProject", mock.Anything, int32(1)).Return(nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/projects/1", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

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

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)
	ts.mockDB.On("ShareProjectWithUser", mock.Anything, db.ShareProjectWithUserParams{
		ProjectID:        1,
		SharedWithUserID: 2,
	}).Return(nil)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/projects/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.SharedProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestSharedProjectHandler_InvalidProjectID(t *testing.T) {
	ts := newTestServer(t)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/projects/abc/share", body)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := ts.server.SharedProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSharedProjectHandler_InvalidBody(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)

	body := []byte(`{invalid}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/projects/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.SharedProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSharedProjectHandler_DBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)
	ts.mockDB.On("ShareProjectWithUser", mock.Anything, mock.Anything).Return(errors.New("db error"))

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/projects/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.SharedProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── UnshareProject ──────────────────────────────────────────────

func TestUnshareProjectHandler_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)
	ts.mockDB.On("UnshareProjectWithUser", mock.Anything, db.UnshareProjectWithUserParams{
		ProjectID:        1,
		SharedWithUserID: 2,
	}).Return(nil)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/projects/1/unshare", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.UnshareProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestUnshareProjectHandler_InvalidProjectID(t *testing.T) {
	ts := newTestServer(t)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/projects/abc/share", body)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := ts.server.UnshareProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUnshareProjectHandler_DBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)
	ts.mockDB.On("UnshareProjectWithUser", mock.Anything, mock.Anything).Return(errors.New("db error"))

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/projects/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.UnshareProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
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

// ── CreateTask ──────────────────────────────────────────────────

func TestCreateTask_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("CreateTask", mock.Anything, db.CreateTaskParams{
		ProjectID:       sql.NullInt32{},
		Title:           "My Task",
		Description:     sql.NullString{},
		TaskType:        db.TasksTaskTypeSingle,
		Priority:        sql.NullInt32{},
		CreatedByUserID: 1,
	}).Return(nil)

	body := []byte(`{"title":"My Task","task_type":"single","user_id":1}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks", body)

	err := ts.server.CreateTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestCreateTask_InvalidJSON(t *testing.T) {
	ts := newTestServer(t)

	body := []byte(`{invalid}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks", body)

	err := ts.server.CreateTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateTask_DBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("CreateTask", mock.Anything, mock.Anything).Return(errors.New("db error"))

	body := []byte(`{"title":"My Task","task_type":"single","user_id":1}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks", body)

	err := ts.server.CreateTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── ListTasksByProject ──────────────────────────────────────────

func TestListTasksByProject_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)
	tasks := []db.Task{
		{ID: 1, Title: "Task 1", TaskType: db.TasksTaskTypeSingle, CreatedByUserID: 1},
		{ID: 2, Title: "Task 2", TaskType: db.TasksTaskTypeRepetitive, CreatedByUserID: 1},
	}
	ts.mockDB.On("GetTaskListByProjectId", mock.Anything, sql.NullInt32{Int32: 1, Valid: true}).Return(tasks, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects/1/tasks", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.ListTasksByProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []db.Task
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
	ts.mockDB.AssertExpectations(t)
}

func TestListTasksByProject_InvalidID(t *testing.T) {
	ts := newTestServer(t)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects/abc/tasks", nil)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := ts.server.ListTasksByProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ── ListTasksByUser ─────────────────────────────────────────────

func TestListTasksByUser_Success(t *testing.T) {
	ts := newTestServer(t)

	tasks := []db.Task{
		{ID: 1, Title: "Task 1", TaskType: db.TasksTaskTypeSingle, CreatedByUserID: 1},
		{ID: 2, Title: "Task 2", TaskType: db.TasksTaskTypeRepetitive, CreatedByUserID: 1},
	}
	ts.mockDB.On("GetTasksByUserId", mock.Anything, db.GetTasksByUserIdParams{UserID: 1}).Return(tasks, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/tasks?user_id=1", nil)

	err := ts.server.ListTasksByUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []db.Task
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
	ts.mockDB.AssertExpectations(t)
}

func TestListTasksByUser_InvalidUserID(t *testing.T) {
	ts := newTestServer(t)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/tasks?user_id=abc", nil)

	err := ts.server.ListTasksByUserHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ── UpdateTask ──────────────────────────────────────────────────

func TestUpdateTask_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)
	ts.mockDB.On("UpdateTask", mock.Anything, db.UpdateTaskParams{
		ID:          1,
		ProjectID:   sql.NullInt32{},
		Title:       "Updated Task",
		Description: sql.NullString{},
		TaskType:    db.TasksTaskTypeSingle,
		Priority:    sql.NullInt32{},
	}).Return(nil)

	body := []byte(`{"title":"Updated Task","task_type":"single"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPut, "/api/tasks/1", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.UpdateTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

// ── DeleteTask ──────────────────────────────────────────────────

func TestDeleteTask_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)
	ts.mockDB.On("DeleteTask", mock.Anything, int32(1)).Return(nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/tasks/1", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.DeleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestDeleteTask_InvalidID(t *testing.T) {
	ts := newTestServer(t)

	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/tasks/abc", nil)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := ts.server.DeleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ── CompleteTask ────────────────────────────────────────────────

func TestCompleteTask_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)
	ts.mockDB.On("CompleteTask", mock.Anything, db.CompleteTaskParams{
		TaskID:            1,
		CompletedByUserID: 2,
	}).Return(nil)

	body := []byte(`{"user_id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks/1/complete", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.CompleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestCompleteTask_InvalidTaskID(t *testing.T) {
	ts := newTestServer(t)

	body := []byte(`{"user_id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks/abc/complete", body)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := ts.server.CompleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCompleteTask_InvalidJSON(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)

	body := []byte(`{invalid}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks/1/complete", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.CompleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCompleteTask_DBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)
	ts.mockDB.On("CompleteTask", mock.Anything, mock.Anything).Return(errors.New("db error"))

	body := []byte(`{"user_id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks/1/complete", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.CompleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── UncompleteTask ──────────────────────────────────────────────

func TestUncompleteTask_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("UncompleteTask", mock.Anything, db.UncompleteTaskParams{
		TaskID:            1,
		CompletedByUserID: 2,
	}).Return(nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/tasks/1/complete?user_id=2", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ts.server.UncompleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestUncompleteTask_InvalidTaskID(t *testing.T) {
	ts := newTestServer(t)

	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/tasks/abc/complete?user_id=2", nil)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := ts.server.UncompleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUncompleteTask_InvalidUserID(t *testing.T) {
	ts := newTestServer(t)

	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/tasks/1/complete?user_id=abc", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ts.server.UncompleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUncompleteTask_DBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("UncompleteTask", mock.Anything, mock.Anything).Return(errors.New("db error"))

	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/tasks/1/complete?user_id=2", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ts.server.UncompleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── GetTaskCompletions ──────────────────────────────────────────

func TestGetTaskCompletions_Success(t *testing.T) {
	ts := newTestServer(t)

	completions := []db.TaskLog{
		{ID: 1, TaskID: 1, CompletedByUserID: 2},
		{ID: 2, TaskID: 1, CompletedByUserID: 3},
	}
	ts.mockDB.On("GetTaskCompletions", mock.Anything, int32(1)).Return(completions, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/tasks/1/completions", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ts.server.GetTaskCompletionsHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []db.TaskLog
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
	ts.mockDB.AssertExpectations(t)
}

func TestGetTaskCompletions_InvalidID(t *testing.T) {
	ts := newTestServer(t)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/tasks/abc/completions", nil)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := ts.server.GetTaskCompletionsHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetTaskCompletions_DBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskCompletions", mock.Anything, int32(1)).Return([]db.TaskLog{}, errors.New("db error"))

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/tasks/1/completions", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ts.server.GetTaskCompletionsHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── ShareTask ───────────────────────────────────────────────────

func TestShareTaskHandler_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)
	ts.mockDB.On("ShareTaskWithUser", mock.Anything, db.ShareTaskWithUserParams{
		TaskID:           1,
		SharedWithUserID: 2,
	}).Return(nil)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.ShareTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestShareTaskHandler_InvalidTaskID(t *testing.T) {
	ts := newTestServer(t)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks/abc/share", body)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := ts.server.ShareTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestShareTaskHandler_InvalidBody(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)

	body := []byte(`{invalid}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.ShareTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestShareTaskHandler_DBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)
	ts.mockDB.On("ShareTaskWithUser", mock.Anything, mock.Anything).Return(errors.New("db error"))

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.ShareTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── UnshareTask ─────────────────────────────────────────────────

func TestUnshareTaskHandler_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)
	ts.mockDB.On("UnshareTaskWithUser", mock.Anything, db.UnshareTaskWithUserParams{
		TaskID:           1,
		SharedWithUserID: 2,
	}).Return(nil)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/tasks/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.UnshareTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	ts.mockDB.AssertExpectations(t)
}

func TestUnshareTaskHandler_InvalidTaskID(t *testing.T) {
	ts := newTestServer(t)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/tasks/abc/share", body)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := ts.server.UnshareTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUnshareTaskHandler_DBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)
	ts.mockDB.On("UnshareTaskWithUser", mock.Anything, mock.Anything).Return(errors.New("db error"))

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/tasks/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 1)

	err := ts.server.UnshareTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── Authorization: Forbidden tests (ownership-only) ─────────────

func TestUpdateProject_Forbidden(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)

	body := []byte(`{"name":"Updated Project"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPut, "/api/projects/1", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.UpdateProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeleteProject_Forbidden(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/projects/1", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.DeleteProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSharedProjectHandler_Forbidden(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/projects/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.SharedProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUnshareProjectHandler_Forbidden(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/projects/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.UnshareProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUpdateTask_Forbidden(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)

	body := []byte(`{"title":"Updated Task","task_type":"single"}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPut, "/api/tasks/1", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.UpdateTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeleteTask_Forbidden(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/tasks/1", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.DeleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestShareTaskHandler_Forbidden(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.ShareTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUnshareTaskHandler_Forbidden(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)

	body := []byte(`{"id":2}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodDelete, "/api/tasks/1/share", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.UnshareTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// ── Authorization: Owner OR Shared access tests ─────────────────

func TestGetProject_Forbidden(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)
	ts.mockDB.On("IsProjectSharedWithUser", mock.Anything, db.IsProjectSharedWithUserParams{
		ProjectID: 1, SharedWithUserID: 99,
	}).Return(false, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects/1", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.GetProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestGetProject_SharedAccess(t *testing.T) {
	ts := newTestServer(t)

	project := db.Project{ID: 1, UserID: 1, Name: "Shared Project"}
	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(project, nil)
	ts.mockDB.On("IsProjectSharedWithUser", mock.Anything, db.IsProjectSharedWithUserParams{
		ProjectID: 1, SharedWithUserID: 99,
	}).Return(true, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects/1", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.GetProject(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListTasksByProject_Forbidden(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)
	ts.mockDB.On("IsProjectSharedWithUser", mock.Anything, db.IsProjectSharedWithUserParams{
		ProjectID: 1, SharedWithUserID: 99,
	}).Return(false, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects/1/tasks", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.ListTasksByProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestListTasksByProject_SharedAccess(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetProjectById", mock.Anything, int32(1)).Return(db.Project{ID: 1, UserID: 1}, nil)
	ts.mockDB.On("IsProjectSharedWithUser", mock.Anything, db.IsProjectSharedWithUserParams{
		ProjectID: 1, SharedWithUserID: 99,
	}).Return(true, nil)
	ts.mockDB.On("GetTaskListByProjectId", mock.Anything, sql.NullInt32{Int32: 1, Valid: true}).Return([]db.Task{}, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/projects/1/tasks", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.ListTasksByProjectHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCompleteTask_Forbidden(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)
	ts.mockDB.On("IsTaskSharedWithUser", mock.Anything, db.IsTaskSharedWithUserParams{
		TaskID: 1, SharedWithUserID: 99,
	}).Return(false, nil)

	body := []byte(`{"user_id":99}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks/1/complete", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.CompleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCompleteTask_SharedAccess(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTaskById", mock.Anything, int32(1)).Return(db.Task{ID: 1, CreatedByUserID: 1}, nil)
	ts.mockDB.On("IsTaskSharedWithUser", mock.Anything, db.IsTaskSharedWithUserParams{
		TaskID: 1, SharedWithUserID: 99,
	}).Return(true, nil)
	ts.mockDB.On("CompleteTask", mock.Anything, db.CompleteTaskParams{
		TaskID: 1, CompletedByUserID: 99,
	}).Return(nil)

	body := []byte(`{"user_id":99}`)
	c, rec := newEchoContext(ts.server.echo, http.MethodPost, "/api/tasks/1/complete", body)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 99)

	err := ts.server.CompleteTaskHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ── GetWeeklyView ───────────────────────────────────────────────

func TestGetWeeklyViewHandler_Success(t *testing.T) {
	ts := newTestServer(t)

	tasks := []db.Task{
		{ID: 1, Title: "Task A", TaskType: db.TasksTaskTypeSingle, CreatedByUserID: 1},
		{ID: 2, Title: "Task B", TaskType: db.TasksTaskTypeRepetitive, CreatedByUserID: 1},
	}
	ts.mockDB.On("GetTasksByUserId", mock.Anything, db.GetTasksByUserIdParams{UserID: 1}).Return(tasks, nil)

	monday := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	sunday := monday.AddDate(0, 0, 7)
	completions := []db.TaskLog{
		{ID: 1, TaskID: 1, CompletedByUserID: 1, CompletedAt: sql.NullTime{Time: time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC), Valid: true}},
		{ID: 2, TaskID: 2, CompletedByUserID: 1, CompletedAt: sql.NullTime{Time: time.Date(2026, 3, 11, 14, 0, 0, 0, time.UTC), Valid: true}},
	}
	ts.mockDB.On("GetCompletionsForWeek", mock.Anything, db.GetCompletionsForWeekParams{
		UserID:    1,
		StartDate: sql.NullTime{Time: monday, Valid: true},
		EndDate:   sql.NullTime{Time: sunday, Valid: true},
	}).Return(completions, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/weekly?week=2026-03-09", nil)
	setAuthUser(c, 1)

	err := ts.server.GetWeeklyViewHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp WeeklyViewResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "2026-03-09", resp.WeekStart)
	assert.Equal(t, "2026-03-15", resp.WeekEnd)
	assert.Len(t, resp.Tasks, 2)
	assert.Len(t, resp.Tasks[0].Days, 7)
	// Task A completed on Monday 2026-03-09
	assert.True(t, resp.Tasks[0].Days[0].Completed)
	assert.False(t, resp.Tasks[0].Days[1].Completed)
	// Task B completed on Wednesday 2026-03-11
	assert.False(t, resp.Tasks[1].Days[0].Completed)
	assert.True(t, resp.Tasks[1].Days[2].Completed)
	ts.mockDB.AssertExpectations(t)
}

func TestGetWeeklyViewHandler_DefaultCurrentWeek(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTasksByUserId", mock.Anything, db.GetTasksByUserIdParams{UserID: 1}).Return([]db.Task{}, nil)
	ts.mockDB.On("GetCompletionsForWeek", mock.Anything, mock.Anything).Return([]db.TaskLog{}, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/weekly", nil)
	setAuthUser(c, 1)

	err := ts.server.GetWeeklyViewHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp WeeklyViewResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// Verify week_start is a Monday
	parsed, parseErr := time.Parse("2006-01-02", resp.WeekStart)
	require.NoError(t, parseErr)
	assert.Equal(t, time.Monday, parsed.Weekday())
}

func TestGetWeeklyViewHandler_InvalidWeekParam(t *testing.T) {
	ts := newTestServer(t)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/weekly?week=not-a-date", nil)
	setAuthUser(c, 1)

	err := ts.server.GetWeeklyViewHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetWeeklyViewHandler_MidWeekDate(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTasksByUserId", mock.Anything, db.GetTasksByUserIdParams{UserID: 1}).Return([]db.Task{}, nil)
	monday := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	sunday := monday.AddDate(0, 0, 7)
	ts.mockDB.On("GetCompletionsForWeek", mock.Anything, db.GetCompletionsForWeekParams{
		UserID:    1,
		StartDate: sql.NullTime{Time: monday, Valid: true},
		EndDate:   sql.NullTime{Time: sunday, Valid: true},
	}).Return([]db.TaskLog{}, nil)

	// Wednesday 2026-03-11 should normalize to Monday 2026-03-09
	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/weekly?week=2026-03-11", nil)
	setAuthUser(c, 1)

	err := ts.server.GetWeeklyViewHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp WeeklyViewResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "2026-03-09", resp.WeekStart)
}

func TestGetWeeklyViewHandler_TasksDBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTasksByUserId", mock.Anything, mock.Anything).Return([]db.Task{}, errors.New("db error"))

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/weekly?week=2026-03-09", nil)
	setAuthUser(c, 1)

	err := ts.server.GetWeeklyViewHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetWeeklyViewHandler_CompletionsDBError(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTasksByUserId", mock.Anything, mock.Anything).Return([]db.Task{}, nil)
	ts.mockDB.On("GetCompletionsForWeek", mock.Anything, mock.Anything).Return([]db.TaskLog{}, errors.New("db error"))

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/weekly?week=2026-03-09", nil)
	setAuthUser(c, 1)

	err := ts.server.GetWeeklyViewHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetWeeklyViewHandler_Unauthorized(t *testing.T) {
	ts := newTestServer(t)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/weekly?week=2026-03-09", nil)
	// no setAuthUser

	err := ts.server.GetWeeklyViewHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetWeeklyViewHandler_NoTasks(t *testing.T) {
	ts := newTestServer(t)

	ts.mockDB.On("GetTasksByUserId", mock.Anything, db.GetTasksByUserIdParams{UserID: 1}).Return([]db.Task{}, nil)
	ts.mockDB.On("GetCompletionsForWeek", mock.Anything, mock.Anything).Return([]db.TaskLog{}, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/weekly?week=2026-03-09", nil)
	setAuthUser(c, 1)

	err := ts.server.GetWeeklyViewHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp WeeklyViewResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, []WeeklyTask{}, resp.Tasks)
}

func TestGetWeeklyViewHandler_NoCompletions(t *testing.T) {
	ts := newTestServer(t)

	tasks := []db.Task{
		{ID: 1, Title: "Task A", TaskType: db.TasksTaskTypeSingle, CreatedByUserID: 1},
	}
	ts.mockDB.On("GetTasksByUserId", mock.Anything, db.GetTasksByUserIdParams{UserID: 1}).Return(tasks, nil)
	ts.mockDB.On("GetCompletionsForWeek", mock.Anything, mock.Anything).Return([]db.TaskLog{}, nil)

	c, rec := newEchoContext(ts.server.echo, http.MethodGet, "/api/weekly?week=2026-03-09", nil)
	setAuthUser(c, 1)

	err := ts.server.GetWeeklyViewHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp WeeklyViewResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Tasks, 1)
	for _, day := range resp.Tasks[0].Days {
		assert.False(t, day.Completed)
	}
}
