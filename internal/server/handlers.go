package server

import (
	"net/http"

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

	hashedPassword, err := utils.HashPassword(req.Password)
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

