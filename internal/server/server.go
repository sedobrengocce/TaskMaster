package server

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/sedobrengocce/TaskMaster/internal/db"
)

type Server struct {
	conn 			*sql.DB
	DB 				*db.Queries
	echo			*echo.Echo
	JWTSecret 		[]byte
	RefreshSecret 	[]byte
	Redis			*redis.Client
	port			string
}

func NewServer(conn *sql.DB, redis *redis.Client, jwtSecret string, refreshSecret string, port string) *Server {
	return &Server{
		conn: conn,
		DB:   db.New(conn),
		echo: echo.New(),
		JWTSecret: []byte(jwtSecret),
		RefreshSecret: []byte(refreshSecret),
		Redis: redis,
		port: port,
	}
}

func (s *Server) Run() error {
	s.echo.Use(middleware.Logger())  
	s.echo.Use(middleware.Recover()) 

	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Al momento permette tutte le origini, ma in produzione dovresti specificare i domini consentiti in una file di configurazione yml o toml
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	s.echo.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey: s.JWTSecret,
		Skipper: func(c echo.Context) bool {
			if c.Path() == "/api/register" || c.Path() == "/api/login" || c.Path() == "/healthcheck" || c.Path() == "/api/refresh_token" {
				return true
			}
			return false
		},
		ErrorHandler: func(c echo.Context, err error) error {
			if err.Error() == "missing or malformed jwt" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing or malformed token"})
			} else if err.Error() == "invalid or expired jwt" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Token is invalid or expired"}) // Gestione del refresh token
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		},
	}))

	s.echo.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:    "cookie:_csrf",
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteStrictMode,
		CookieSecure:   true,
	}))

	s.RegisterRoutes()

	return s.echo.Start(":" + s.port)
}

