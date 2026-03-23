package server

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/sedobrengocce/TaskMaster/internal/db"
	"github.com/sedobrengocce/TaskMaster/internal/utils"
)

func getUserIDFromContext(c echo.Context) (int32, error) {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok || token == nil {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "Missing token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "Invalid claims")
	}
	userIDFloat, ok := claims["sub"].(float64)
	if !ok {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "Invalid sub claim")
	}
	return int32(userIDFloat), nil
}

func isProjectOwnerOrShared(ctx echo.Context, querier db.Querier, projectID int32, userID int32) (bool, error) {
	project, err := querier.GetProjectById(ctx.Request().Context(), projectID)
	if err != nil {
		return false, err
	}
	if project.UserID == userID {
		return true, nil
	}
	return querier.IsProjectSharedWithUser(ctx.Request().Context(), db.IsProjectSharedWithUserParams{
		ProjectID:        projectID,
		SharedWithUserID: userID,
	})
}

func isTaskOwnerOrShared(ctx echo.Context, querier db.Querier, taskID int32, userID int32) (bool, error) {
	task, err := querier.GetTaskById(ctx.Request().Context(), taskID)
	if err != nil {
		return false, err
	}
	if task.CreatedByUserID == userID {
		return true, nil
	}
	return querier.IsTaskSharedWithUser(ctx.Request().Context(), db.IsTaskSharedWithUserParams{
		TaskID:           taskID,
		SharedWithUserID: userID,
	})
}

type ClientType string

const (
	ClientTypeWeb    ClientType = "web"
	ClientTypeMobile ClientType = "mobile"
)

func (s *Server) HealthCheckHandler(c echo.Context) error {
	response := struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Status:  "ok",
		Message: "Server is up and running",
	}

	return c.JSON(http.StatusOK, response)
}

type RegisterUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type UserResponse struct {
	ID    int32  `json:"id"`
	Email string `json:"email"`
}

func (s *Server) RegisterUserHandler(c echo.Context) error {
	var req RegisterUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	hashedPassword, err := utils.HashString(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"})
	}

	err = s.DB.CreateUser(c.Request().Context(), db.CreateUserParams{
		Email:        req.Email,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		if err.Error() == "user already exists" {
			return c.JSON(http.StatusConflict, map[string]string{"error": "User already exists"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
	}

	newUser, err := s.DB.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve user"})
	}

	resp := UserResponse{
		ID:    newUser.ID,
		Email: newUser.Email,
	}

	return c.JSON(http.StatusOK, resp)
}

type LoginUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password"`
}

func (s *Server) checkClientType(c echo.Context) ClientType {
	xClientType := c.Request().Header.Get("X-Client-Type")
	xClientSecret := c.Request().Header.Get("X-Client-Secret")
	clientType := ClientTypeWeb
	client, err := s.DB.GetClientByClientID(c.Request().Context(), xClientType)
	if err != nil || client.ClientSecretHash.String == "" {
		clientType = ClientTypeWeb
	} else {
		if utils.CheckStringHash(xClientSecret, client.ClientSecretHash.String) {
			clientType = ClientTypeMobile
		} else {
			clientType = ClientTypeWeb
		}
	}
	return clientType
}

func (s *Server) LoginUserHandler(c echo.Context) error {
	var req LoginUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	user, err := s.DB.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid email or password"})
	}

	if !utils.CheckStringHash(req.Password, user.PasswordHash) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid email or password"})
	}

	jti, err := utils.GenerateRandomString(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token ID"})
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "taskmaster",
		"sub": user.ID,
		"aud": "taskmaster_users",
		"nbf": jwt.NewNumericDate(time.Now()),
		"iat": jwt.NewNumericDate(time.Now()),
		"jti": jti,
		"exp": jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
	})
	tokenString, err := jwtToken.SignedString([]byte(s.JWTSecret))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}

	rjti, err := utils.GenerateRandomString(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate refresh token ID"})
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"iss": "taskmaster",
		"aud": "taskmaster_users",
		"nbf": jwt.NewNumericDate(time.Now()),
		"iat": jwt.NewNumericDate(time.Now()),
		"jti": rjti,
		"exp": jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	})
	refreshTokenString, err := refreshToken.SignedString([]byte(s.RefreshSecret))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate refresh token"})
	}

	hashedRefreshToken := utils.HashToken(refreshTokenString)

	err = s.DB.InsertRefreshToken(c.Request().Context(), db.InsertRefreshTokenParams{
		UserID:    int64(user.ID),
		TokenHash: hashedRefreshToken,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IpAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to store refresh token"})
	}

	resp := map[string]string{
		"jwt": tokenString,
	}

	clientType := s.checkClientType(c)
	if clientType == ClientTypeMobile {
		resp["refresh_token"] = refreshTokenString
	}

	cookie := new(http.Cookie)
	cookie.Name = "refresh_token"
	cookie.Value = refreshTokenString
	cookie.Expires = time.Now().Add(24 * time.Hour)
	cookie.HttpOnly = true
	cookie.Secure = true
	cookie.SameSite = http.SameSiteStrictMode
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) parseToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid token signing method")
		}
		return s.RefreshSecret, nil
	})
}

func (s *Server) RefreshTokenHandler(c echo.Context) error {
	var token *jwt.Token
	var err error
	isMobile := s.checkClientType(c) == ClientTypeMobile

	if isMobile {
		var req struct {
			RefreshToken string `json:"refresh_token" validate:"required"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
		}
		if err := c.Validate(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		token, err = s.parseToken(req.RefreshToken)
	} else {
		cookie, err := c.Cookie("refresh_token")
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing refresh token"})
		}
		token, err = s.parseToken(cookie.Value)
	}
	if err != nil || !token.Valid {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or expired refresh token"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token claims"})
	}

	expirationTime, err := claims.GetExpirationTime()
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token expiration"})
	}

	if time.Now().After(expirationTime.Time) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Refresh token has expired"})
	}

	userIDFloat, ok := claims["sub"].(float64)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token claims"})
	}
	userID := int64(userIDFloat)

	storedToken, err := s.DB.GetRefreshToken(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Refresh token not found"})
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Refresh token has expired"})
	}

	if !utils.CheckTokenHash(token.Raw, storedToken.TokenHash) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid refresh token"})
	}

	user, err := s.DB.GetUserByID(c.Request().Context(), int32(userID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve user"})
	}

	jti, err := utils.GenerateRandomString(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token ID"})
	}
	newJwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"iss": "taskmaster",
		"aud": "taskmaster_users",
		"nbf": jwt.NewNumericDate(time.Now()),
		"iat": jwt.NewNumericDate(time.Now()),
		"jti": jti,
		"exp": jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
	})
	newTokenString, err := newJwtToken.SignedString([]byte(s.JWTSecret))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}

	rjti, err := utils.GenerateRandomString(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate refresh token ID"})
	}
	newRefreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"iss": "taskmaster",
		"aud": "taskmaster_users",
		"nbf": jwt.NewNumericDate(time.Now()),
		"iat": jwt.NewNumericDate(time.Now()),
		"jti": rjti,
		"exp": jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	})
	newRefreshTokenString, err := newRefreshToken.SignedString([]byte(s.RefreshSecret))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate refresh token"})
	}
	hashedNewRefreshToken := utils.HashToken(newRefreshTokenString)
	err = s.DB.InsertRefreshToken(c.Request().Context(), db.InsertRefreshTokenParams{
		UserID:    int64(user.ID),
		TokenHash: hashedNewRefreshToken,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IpAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to store refresh token"})
	}

	resp := map[string]string{
		"jwt": newTokenString,
	}

	if isMobile {
		resp["refresh_token"] = newRefreshTokenString
		return c.JSON(http.StatusOK, resp)
	}

	newCookie := new(http.Cookie)
	newCookie.Name = "refresh_token"
	newCookie.Value = newRefreshTokenString
	newCookie.Expires = time.Now().Add(24 * time.Hour)
	newCookie.HttpOnly = true
	newCookie.Secure = true
	newCookie.SameSite = http.SameSiteStrictMode
	c.SetCookie(newCookie)

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) LogoutUserHandler(c echo.Context) error {
	authHeader := c.Request().Header.Get("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse del token per ottenere la sua data di scadenza
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.JSON(http.StatusBadRequest, "Invalid token claims")
	}

	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return c.JSON(http.StatusBadRequest, "Invalid expiration claim")
	}

	expTime := time.Unix(int64(expFloat), 0)
	ttl := time.Until(expTime)

	if ttl <= 0 {
		return c.JSON(http.StatusOK, map[string]string{"message": "Logged out successfully"})
	}

	err = s.Redis.Set(c.Request().Context(), tokenString, "revoked", ttl).Err()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Could not log out")
	}
	userIDFloat, ok := claims["sub"].(float64)
	if !ok {
		return c.JSON(http.StatusBadRequest, "Invalid token claims")
	}
	userID := int64(userIDFloat)

	err = s.DB.RevokeRefreshToken(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to revoke refresh tokens"})
	}
	clientType := s.checkClientType(c)
	response := map[string]string{
		"message": "Logged out successfully",
	}

	switch clientType {
	case ClientTypeWeb:
		expiredCookie := new(http.Cookie)
		expiredCookie.Name = "refresh_token"
		expiredCookie.Value = ""
		expiredCookie.Expires = time.Now().Add(-1 * time.Hour)
		expiredCookie.HttpOnly = true
		expiredCookie.Secure = true
		expiredCookie.SameSite = http.SameSiteStrictMode
		c.SetCookie(expiredCookie)
	case ClientTypeMobile:
		response["action_required"] = "clear_local_tokens"
	}

	return c.JSON(http.StatusOK, response)
}

type CreateProjectRequest struct {
	Name     string  `json:"name" validate:"required,min=3,max=100"`
	UserID   int32   `json:"user_id" validate:"required"`
	ColorHex *string `json:"color_hex" validate:"omitempty,len=7"`
}

func (s *Server) CreateProject(c echo.Context) error {
	var req CreateProjectRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	colorHex := sql.NullString{}
	if req.ColorHex != nil {
		colorHex = sql.NullString{String: *req.ColorHex, Valid: true}
	}

	err := s.DB.CreateProject(c.Request().Context(), db.CreateProjectParams{
		Name:     req.Name,
		UserID:   req.UserID,
		ColorHex: colorHex,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create project"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Project created successfully"})

}

func (s *Server) ListProjects(c echo.Context) error {
	userID, err := strconv.ParseInt(c.QueryParam("user_id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user_id parameter"})
	}

	projects, err := s.DB.GetProjectsByUserId(c.Request().Context(), db.GetProjectsByUserIdParams{
		UserID: int32(userID),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve projects"})
	}

	return c.JSON(http.StatusOK, projects)
}

func (s *Server) GetProject(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid project ID"})
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}
	allowed, err := isProjectOwnerOrShared(c, s.DB, int32(projectID), authUserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check authorization"})
	}
	if !allowed {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
	}

	project, err := s.DB.GetProjectById(c.Request().Context(), int32(projectID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve project"})
	}

	return c.JSON(http.StatusOK, project)
}

type UpdateProjectRequest struct {
	Name     string  `json:"name" validate:"required,min=3,max=100"`
	ColorHex *string `json:"color_hex" validate:"omitempty,len=7"`
}

func (s *Server) UpdateProject(c echo.Context) error {
	var req UpdateProjectRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid project ID"})
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}
	project, err := s.DB.GetProjectById(c.Request().Context(), int32(projectID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve project"})
	}
	if project.UserID != authUserID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
	}

	colorHex := sql.NullString{}
	if req.ColorHex != nil {
		colorHex = sql.NullString{String: *req.ColorHex, Valid: true}
	}
	err = s.DB.UpdateProject(c.Request().Context(), db.UpdateProjectParams{
		ID:       int32(projectID),
		Name:     req.Name,
		ColorHex: colorHex,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update project"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Project updated successfully"})
}

func (s *Server) DeleteProject(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid project ID"})
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}
	project, err := s.DB.GetProjectById(c.Request().Context(), int32(projectID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve project"})
	}
	if project.UserID != authUserID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
	}

	err = s.DB.DeleteProject(c.Request().Context(), int32(projectID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete project"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Project deleted successfully"})
}

type ShareRequest struct {
	ID int32 `json:"id" validate:"required"`
}

func (s *Server) SharedProjectHandler(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid project ID"})
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}
	project, err := s.DB.GetProjectById(c.Request().Context(), int32(projectID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve project"})
	}
	if project.UserID != authUserID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
	}

	var req ShareRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err = s.DB.ShareProjectWithUser(c.Request().Context(), db.ShareProjectWithUserParams{
		ProjectID:        int32(projectID),
		SharedWithUserID: req.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to share project"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Project shared successfully"})
}

func (s *Server) UnshareProjectHandler(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid project ID"})
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}
	project, err := s.DB.GetProjectById(c.Request().Context(), int32(projectID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve project"})
	}
	if project.UserID != authUserID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
	}

	var req ShareRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err = s.DB.UnshareProjectWithUser(c.Request().Context(), db.UnshareProjectWithUserParams{
		ProjectID:        int32(projectID),
		SharedWithUserID: req.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to unshare project"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Project unshared successfully"})
}

func (s *Server) ShareTaskHandler(c echo.Context) error {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid task ID"})
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}
	task, err := s.DB.GetTaskById(c.Request().Context(), int32(taskID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve task"})
	}
	if task.CreatedByUserID != authUserID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
	}

	var req ShareRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err = s.DB.ShareTaskWithUser(c.Request().Context(), db.ShareTaskWithUserParams{
		TaskID:           int32(taskID),
		SharedWithUserID: req.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to share task"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Task shared successfully"})
}

func (s *Server) UnshareTaskHandler(c echo.Context) error {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid task ID"})
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}
	task, err := s.DB.GetTaskById(c.Request().Context(), int32(taskID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve task"})
	}
	if task.CreatedByUserID != authUserID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
	}

	var req ShareRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err = s.DB.UnshareTaskWithUser(c.Request().Context(), db.UnshareTaskWithUserParams{
		TaskID:           int32(taskID),
		SharedWithUserID: req.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to unshare task"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Task unshared successfully"})
}

type CreateTaskRequest struct {
	ProjectID   *int32  `json:"project_id" validate:"omitempty"`
	Title       string  `json:"title" validate:"required,min=1,max=255"`
	Description *string `json:"description" validate:"omitempty"`
	TaskType    string  `json:"task_type" validate:"required,oneof=single repetitive"`
	Priority    *int32  `json:"priority" validate:"omitempty"`
	UserID      int32   `json:"user_id" validate:"required"`
}

type UpdateTaskRequest struct {
	ProjectID   *int32  `json:"project_id" validate:"omitempty"`
	Title       string  `json:"title" validate:"required,min=1,max=255"`
	Description *string `json:"description" validate:"omitempty"`
	TaskType    string  `json:"task_type" validate:"required,oneof=single repetitive"`
	Priority    *int32  `json:"priority" validate:"omitempty"`
}

func (s *Server) CreateTaskHandler(c echo.Context) error {
	var req CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	projectID := sql.NullInt32{}
	if req.ProjectID != nil {
		projectID = sql.NullInt32{Int32: *req.ProjectID, Valid: true}
	}
	description := sql.NullString{}
	if req.Description != nil {
		description = sql.NullString{String: *req.Description, Valid: true}
	}
	priority := sql.NullInt32{}
	if req.Priority != nil {
		priority = sql.NullInt32{Int32: *req.Priority, Valid: true}
	}

	err := s.DB.CreateTask(c.Request().Context(), db.CreateTaskParams{
		ProjectID:       projectID,
		Title:           req.Title,
		Description:     description,
		TaskType:        db.TasksTaskType(req.TaskType),
		Priority:        priority,
		CreatedByUserID: req.UserID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create task"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Task created successfully"})
}

func (s *Server) ListTasksByProjectHandler(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid project ID"})
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}
	allowed, err := isProjectOwnerOrShared(c, s.DB, int32(projectID), authUserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check authorization"})
	}
	if !allowed {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
	}

	tasks, err := s.DB.GetTaskListByProjectId(c.Request().Context(), sql.NullInt32{Int32: int32(projectID), Valid: true})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve tasks"})
	}

	return c.JSON(http.StatusOK, tasks)
}

func (s *Server) ListTasksByUserHandler(c echo.Context) error {
	userID, err := strconv.ParseInt(c.QueryParam("user_id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user_id parameter"})
	}

	tasks, err := s.DB.GetTasksByUserId(c.Request().Context(), db.GetTasksByUserIdParams{
		UserID: int32(userID),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve tasks"})
	}

	return c.JSON(http.StatusOK, tasks)
}

func (s *Server) UpdateTaskHandler(c echo.Context) error {
	var req UpdateTaskRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid task ID"})
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}
	task, err := s.DB.GetTaskById(c.Request().Context(), int32(taskID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve task"})
	}
	if task.CreatedByUserID != authUserID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
	}

	projectID := sql.NullInt32{}
	if req.ProjectID != nil {
		projectID = sql.NullInt32{Int32: *req.ProjectID, Valid: true}
	}
	description := sql.NullString{}
	if req.Description != nil {
		description = sql.NullString{String: *req.Description, Valid: true}
	}
	priority := sql.NullInt32{}
	if req.Priority != nil {
		priority = sql.NullInt32{Int32: *req.Priority, Valid: true}
	}

	err = s.DB.UpdateTask(c.Request().Context(), db.UpdateTaskParams{
		ID:          int32(taskID),
		ProjectID:   projectID,
		Title:       req.Title,
		Description: description,
		TaskType:    db.TasksTaskType(req.TaskType),
		Priority:    priority,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update task"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Task updated successfully"})
}

func (s *Server) DeleteTaskHandler(c echo.Context) error {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid task ID"})
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}
	task, err := s.DB.GetTaskById(c.Request().Context(), int32(taskID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve task"})
	}
	if task.CreatedByUserID != authUserID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
	}

	err = s.DB.DeleteTask(c.Request().Context(), int32(taskID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete task"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Task deleted successfully"})
}

type CompleteTaskRequest struct {
	UserID int32 `json:"user_id" validate:"required"`
}

func (s *Server) CompleteTaskHandler(c echo.Context) error {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid task ID"})
	}

	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}
	allowed, err := isTaskOwnerOrShared(c, s.DB, int32(taskID), authUserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check authorization"})
	}
	if !allowed {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
	}

	var req CompleteTaskRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err = s.DB.CompleteTask(c.Request().Context(), db.CompleteTaskParams{
		TaskID:            int32(taskID),
		CompletedByUserID: req.UserID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to complete task"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Task completed successfully"})
}

func (s *Server) UncompleteTaskHandler(c echo.Context) error {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid task ID"})
	}

	userID, err := strconv.ParseInt(c.QueryParam("user_id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user_id parameter"})
	}

	err = s.DB.UncompleteTask(c.Request().Context(), db.UncompleteTaskParams{
		TaskID:            int32(taskID),
		CompletedByUserID: int32(userID),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to uncomplete task"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Task uncompleted successfully"})
}

// ── Weekly View ─────────────────────────────────────────────────

type WeeklyTaskDay struct {
	Date      string `json:"date"`
	Weekday   string `json:"weekday"`
	Completed bool   `json:"completed"`
}

type WeeklyTask struct {
	ID          int32           `json:"id"`
	Title       string          `json:"title"`
	Description *string         `json:"description,omitempty"`
	TaskType    string          `json:"task_type"`
	Priority    *int32          `json:"priority,omitempty"`
	ProjectID   *int32          `json:"project_id,omitempty"`
	Days        []WeeklyTaskDay `json:"days"`
}

type WeeklyViewResponse struct {
	WeekStart string       `json:"week_start"`
	WeekEnd   string       `json:"week_end"`
	Tasks     []WeeklyTask `json:"tasks"`
}

func (s *Server) GetWeeklyViewHandler(c echo.Context) error {
	authUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}

	var refDate time.Time
	weekParam := c.QueryParam("week")
	if weekParam == "" {
		refDate = time.Now()
	} else {
		refDate, err = time.Parse("2006-01-02", weekParam)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid week parameter, expected YYYY-MM-DD"})
		}
	}

	offset := (int(refDate.Weekday()) - int(time.Monday) + 7) % 7
	monday := time.Date(refDate.Year(), refDate.Month(), refDate.Day()-offset, 0, 0, 0, 0, time.UTC)
	sunday := monday.AddDate(0, 0, 7)

	tasks, err := s.DB.GetTasksByUserId(c.Request().Context(), db.GetTasksByUserIdParams{
		UserID: authUserID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve tasks"})
	}

	completions, err := s.DB.GetCompletionsForWeek(c.Request().Context(), db.GetCompletionsForWeekParams{
		UserID:    authUserID,
		StartDate: sql.NullTime{Time: monday, Valid: true},
		EndDate:   sql.NullTime{Time: sunday, Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve completions"})
	}

	completionMap := make(map[int32]map[string]bool)
	for _, comp := range completions {
		if comp.CompletedAt.Valid {
			dateStr := comp.CompletedAt.Time.Format("2006-01-02")
			if completionMap[comp.TaskID] == nil {
				completionMap[comp.TaskID] = make(map[string]bool)
			}
			completionMap[comp.TaskID][dateStr] = true
		}
	}

	weeklyTasks := make([]WeeklyTask, 0, len(tasks))
	for _, t := range tasks {
		wt := WeeklyTask{
			ID:       t.ID,
			Title:    t.Title,
			TaskType: string(t.TaskType),
		}
		if t.Description.Valid {
			wt.Description = &t.Description.String
		}
		if t.Priority.Valid {
			wt.Priority = &t.Priority.Int32
		}
		if t.ProjectID.Valid {
			wt.ProjectID = &t.ProjectID.Int32
		}

		days := make([]WeeklyTaskDay, 7)
		for i := 0; i < 7; i++ {
			day := monday.AddDate(0, 0, i)
			dateStr := day.Format("2006-01-02")
			days[i] = WeeklyTaskDay{
				Date:      dateStr,
				Weekday:   day.Weekday().String(),
				Completed: completionMap[t.ID][dateStr],
			}
		}
		wt.Days = days
		weeklyTasks = append(weeklyTasks, wt)
	}

	return c.JSON(http.StatusOK, WeeklyViewResponse{
		WeekStart: monday.Format("2006-01-02"),
		WeekEnd:   monday.AddDate(0, 0, 6).Format("2006-01-02"),
		Tasks:     weeklyTasks,
	})
}

func (s *Server) GetTaskCompletionsHandler(c echo.Context) error {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid task ID"})
	}

	completions, err := s.DB.GetTaskCompletions(c.Request().Context(), int32(taskID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve completions"})
	}

	if completions == nil {
		completions = []db.TaskLog{}
	}

	return c.JSON(http.StatusOK, completions)
}
