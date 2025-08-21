package server

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/sedobrengocce/TaskMaster/internal/db"
	"github.com/sedobrengocce/TaskMaster/internal/utils"
)

func (s *Server)HealthCheckHandler(c echo.Context) error {
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
	Email	string `json:"email" validate:"required,email"`
	Password	string `json:"password" validate:"required,min=8"`
}

func (s *Server)RegisterUserHandler(c echo.Context) error {
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
		Email:    req.Email,
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


	return c.JSON(http.StatusOK, newUser)
}

type LoginUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password"`
}

func (s *Server)LoginUserHandler(c echo.Context) error {
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

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Email,
		"id":    user.ID,
		"exp":	 jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
	})
	tokenString, err := jwtToken.SignedString([]byte(s.JWTSecret))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": user.ID,
		"exp": jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	})
	refreshTokenString, err := refreshToken.SignedString([]byte(s.JWTSecret))
	if err != nil {	
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate refresh token"})
	}

	hashedRefreshToken, err := utils.HashString(refreshTokenString)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to hash refresh token"})
	}

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

	cookie := new(http.Cookie)
	cookie.Name = "refresh_token"
	cookie.Value = refreshTokenString
	cookie.Expires = time.Now().Add(24 * time.Hour)
	cookie.HttpOnly = true
	cookie.Secure = true
	cookie.SameSite = http.SameSiteStrictMode
	c.SetCookie(cookie)


	return c.JSON(http.StatusOK, map[string]string{"jwt": tokenString})
}

