package server

import (
	"net/http"
	"strings"
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

	jti, err := utils.GenerateRandomString(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token ID"})
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Email,
		"id":    user.ID,
		"iss": "taskmaster",
		"sub": "user_auth",
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

	rjti , err := utils.GenerateRandomString(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate refresh token ID"})
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": user.ID,
		"iss": "taskmaster",
		"sub": "user_refresh",
		"aud": "taskmaster_users",
		"nbf": jwt.NewNumericDate(time.Now()),
		"iat": jwt.NewNumericDate(time.Now()),
		"jti": rjti,
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

	resp := map[string]string{
		"jwt": tokenString,
	}

	xClientType := c.Request().Header.Get("X-Client-Type")
	if xClientType == "mobile" {
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

func parseToken(tokenString string, secret []byte) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid token signing method")
		}
		return secret, nil
	})
}

func (s *Server)RefreshTokenHandler(c echo.Context) error {
	var token *jwt.Token
	var err error
	isMobile := c.Request().Header.Get("X-Client-Type") == "mobile"

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
		token, err = parseToken(req.RefreshToken, s.RefreshSecret)
	} else {
		cookie, err := c.Cookie("refresh_token")
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing refresh token"})
		}
		token, err = parseToken(cookie.Value, s.RefreshSecret)
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

	userIDFloat, ok := claims["id"].(float64)
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

	if !utils.CheckStringHash(token.Raw, storedToken.TokenHash) {
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
		"email": user.Email,
		"id":    user.ID,
		"iss": "taskmaster",
		"sub": "user_auth",
		"aud": "taskmaster_users",
		"nbf": jwt.NewNumericDate(time.Now()),
		"iat": jwt.NewNumericDate(time.Now()),
		"jti": jti,
		"exp":	 jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
	})
	newTokenString, err := newJwtToken.SignedString([]byte(s.JWTSecret))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}

	rjti , err := utils.GenerateRandomString(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate refresh token ID"})
	}
	newRefreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": user.ID,
		"iss": "taskmaster",
		"sub": "user_refresh",
		"aud": "taskmaster_users",
		"nbf": jwt.NewNumericDate(time.Now()),
		"iat": jwt.NewNumericDate(time.Now()),
		"jti": rjti,
		"exp": jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	})
	newRefreshTokenString, err := newRefreshToken.SignedString([]byte(s.JWTSecret))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate refresh token"})
	}
	hashedNewRefreshToken, err := utils.HashString(newRefreshTokenString)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to hash refresh token"})
	}
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

func (s *Server)LogoutUserHandler(c echo.Context) error {
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
	userIDFloat, ok := claims["id"].(float64)
	if !ok {
		return c.JSON(http.StatusBadRequest, "Invalid token claims")
	}
	userID := int64(userIDFloat)

	err = s.DB.RevokeRefreshToken(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to revoke refresh tokens"})
	}
	

	if c.Request().Header.Get("X-Client-Type") != "mobile" {
		expiredCookie := new(http.Cookie)
		expiredCookie.Name = "refresh_token"
		expiredCookie.Value = ""
		expiredCookie.Expires = time.Now().Add(-1 * time.Hour)
		expiredCookie.HttpOnly = true
		expiredCookie.Secure = true
		expiredCookie.SameSite = http.SameSiteStrictMode
		c.SetCookie(expiredCookie)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

